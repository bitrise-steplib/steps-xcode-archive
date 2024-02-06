// Package appstoreconnect implements a client for the App Store Connect API.
//
// It contains type definitions, authentication and API calls, without business logic built on those API calls.
package appstoreconnect

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"time"

	"github.com/bitrise-io/go-utils/httputil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/retry"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/go-querystring/query"
	"github.com/hashicorp/go-retryablehttp"
)

const (
	baseURL    = "https://api.appstoreconnect.apple.com/"
	apiVersion = "v1"
)

var (
	// A given token can be reused for up to 20 minutes:
	// https://developer.apple.com/documentation/appstoreconnectapi/generating_tokens_for_api_requests
	//
	// Using 19 minutes to make sure time inaccuracies at token validation does not cause issues.
	jwtDuration    = 19 * time.Minute
	jwtReserveTime = 2 * time.Minute
)

// HTTPClient ...
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type service struct {
	client *Client
}

// Client communicate with the Apple API
type Client struct {
	EnableDebugLogs bool

	keyID             string
	issuerID          string
	privateKeyContent []byte

	token       *jwt.Token
	signedToken string

	client  HTTPClient
	BaseURL *url.URL

	common       service // Reuse a single struct instead of allocating one for each service on the heap.
	Provisioning *ProvisioningService
}

// NewRetryableHTTPClient create a new http client with retry settings.
func NewRetryableHTTPClient() *http.Client {
	client := retry.NewHTTPClient()
	client.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			log.Debugf("Received HTTP 401 (Unauthorized), retrying request...")
			return true, nil
		}

		shouldRetry, err := retryablehttp.DefaultRetryPolicy(ctx, resp, err)
		if shouldRetry && resp != nil {
			log.Debugf("Retry network error: %d", resp.StatusCode)
		}

		return shouldRetry, err
	}
	return client.StandardClient()
}

// NewClient creates a new client
func NewClient(httpClient HTTPClient, keyID, issuerID string, privateKey []byte) *Client {
	baseURL, err := url.Parse(baseURL)
	if err != nil {
		panic("invalid api base url: " + err.Error())
	}

	c := &Client{
		keyID:             keyID,
		issuerID:          issuerID,
		privateKeyContent: privateKey,

		client:  httpClient,
		BaseURL: baseURL,
	}
	c.common.client = c
	c.Provisioning = (*ProvisioningService)(&c.common)

	return c
}

// ensureSignedToken makes sure that the JWT auth token is not expired
// and return a signed key
func (c *Client) ensureSignedToken() (string, error) {
	if c.token != nil {
		claim, ok := c.token.Claims.(claims)
		if !ok {
			return "", fmt.Errorf("failed to cast claim for token")
		}
		expiration := time.Unix(int64(claim.Expiration), 0)

		// A given token can be reused for up to 20 minutes:
		// https://developer.apple.com/documentation/appstoreconnectapi/generating_tokens_for_api_requests
		//
		// The step generates a new token 2 minutes before the expiry.
		if time.Until(expiration) > jwtReserveTime {
			return c.signedToken, nil
		}

		log.Debugf("JWT token expired, regenerating")
	} else {
		log.Debugf("Generating JWT token")
	}

	c.token = createToken(c.keyID, c.issuerID)
	var err error
	if c.signedToken, err = signToken(c.token, c.privateKeyContent); err != nil {
		return "", err
	}
	return c.signedToken, nil
}

// NewRequest creates a new http.Request
func (c *Client) NewRequest(method, endpoint string, body interface{}) (*http.Request, error) {
	endpoint = apiVersion + "/" + endpoint
	u, err := c.BaseURL.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parsing endpoint failed: %v", err)
	}

	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(body); err != nil {
			return nil, fmt.Errorf("encoding body failed: %v", err)
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, fmt.Errorf("preparing request failed: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if _, ok := c.client.(*http.Client); ok {
		signedToken, err := c.ensureSignedToken()
		if err != nil {
			return nil, fmt.Errorf("ensuring JWT token failed: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+signedToken)
	}

	return req, nil
}

func checkResponse(r *http.Response) error {
	if r.StatusCode >= 200 && r.StatusCode <= 299 {
		return nil
	}

	errorResponse := &ErrorResponse{Response: r}
	data, err := io.ReadAll(r.Body)
	if err == nil && data != nil {
		if err := json.Unmarshal(data, errorResponse); err != nil {
			log.Errorf("Failed to unmarshal response (%s): %s", string(data), err)
		}
	}
	return errorResponse
}

// Debugf ...
func (c *Client) Debugf(format string, v ...interface{}) {
	if c.EnableDebugLogs {
		log.Debugf(format, v...)
	}
}

// Do ...
func (c *Client) Do(req *http.Request, v interface{}) (*http.Response, error) {
	c.Debugf("Request:")
	if c.EnableDebugLogs {
		if err := httputil.PrintRequest(req); err != nil {
			c.Debugf("Failed to print request: %s", err)
		}
	}

	resp, err := c.client.Do(req)

	c.Debugf("Response:")
	if c.EnableDebugLogs {
		if err := httputil.PrintResponse(resp); err != nil {
			c.Debugf("Failed to print response: %s", err)
		}
	}

	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Warnf("Failed to close response body: %s", cerr)
		}
	}()

	if err := checkResponse(resp); err != nil {
		return resp, err
	}

	if v != nil {
		decErr := json.NewDecoder(resp.Body).Decode(v)
		if decErr == io.EOF {
			decErr = nil // ignore EOF errors caused by empty response body
		}
		if decErr != nil {
			err = decErr
		}
	}

	return resp, err
}

// PagingOptions ...
type PagingOptions struct {
	Limit  int    `url:"limit,omitempty"`
	Cursor string `url:"cursor,omitempty"`
	Next   string `url:"-"`
}

// UpdateCursor ...
func (opt *PagingOptions) UpdateCursor() error {
	if opt != nil && opt.Next != "" {
		u, err := url.Parse(opt.Next)
		if err != nil {
			return err
		}
		cursor := u.Query().Get("cursor")
		opt.Cursor = cursor
	}
	return nil
}

// addOptions adds the parameters in opt as URL query parameters to s. opt
// must be a struct whose fields may contain "url" tags.
func addOptions(s string, opt interface{}) (string, error) {
	v := reflect.ValueOf(opt)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return s, nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	qs, err := query.Values(opt)
	if err != nil {
		return s, err
	}

	u.RawQuery = qs.Encode()
	return u.String(), nil
}
