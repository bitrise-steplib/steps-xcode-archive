// Package appstoreconnect implements a client for the App Store Connect API.
//
// It contains type definitions, authentication and API calls, without business logic built on those API calls.
package appstoreconnect

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/httputil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/retry"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/go-querystring/query"
	"github.com/hashicorp/go-retryablehttp"
)

const (
	clientBaseURL = "https://api.appstoreconnect.apple.com/"
	tokenAudience = "appstoreconnect-v1"

	clientBaseEnterpiseURL  = "https://api.enterprise.developer.apple.com/"
	tokenEnterpriseAudience = "apple-developer-enterprise-v1"

	apiVersion = "v1"
)

var (
	// A given token can be reused for up to 20 minutes:
	// https://developer.apple.com/documentation/appstoreconnectapi/generating_tokens_for_api_requests
	//
	// We use 18 minutes to make sure time inaccuracies at token validation does not cause issues.
	jwtDuration = 18 * time.Minute
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
	audience          string

	token       *jwt.Token
	signedToken string

	client  HTTPClient
	BaseURL *url.URL

	common       service // Reuse a single struct instead of allocating one for each service on the heap.
	Provisioning *ProvisioningService

	tracker Tracker
}

// NewRetryableHTTPClient create a new http client with retry settings.
func NewRetryableHTTPClient() *http.Client {
	client := retry.NewHTTPClient()
	client.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			log.Debugf("Received HTTP 401 (Unauthorized), retrying request...")
			return true, nil
		}

		if resp != nil && resp.StatusCode == http.StatusForbidden {
			var apiError *ErrorResponse
			if ok := errors.As(checkResponse(resp), &apiError); ok {
				if apiError.IsRequiredAgreementMissingOrExpired() {
					log.Warnf("Received error FORBIDDEN.REQUIRED_AGREEMENTS_MISSING_OR_EXPIRED (status 403), retrying request...")
					return true, nil
				}
			}
		}

		if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
			message := "Received HTTP 429 Too Many Requests"
			if rateLimit := resp.Header.Get("X-Rate-Limit"); rateLimit != "" {
				message += " (" + rateLimit + ")"
			}

			if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
				message += ", retrying the request in " + retryAfter + " seconds..."
			} else {
				message += ", retrying the request..."
			}

			log.Warnf(message)

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
func NewClient(httpClient HTTPClient, keyID, issuerID string, privateKey []byte, isEnterpise bool, tracker Tracker) *Client {
	targetURL := clientBaseURL
	targetAudience := tokenAudience
	if isEnterpise {
		targetURL = clientBaseEnterpiseURL
		targetAudience = tokenEnterpriseAudience
	}

	baseURL, err := url.Parse(targetURL)
	if err != nil {
		panic("invalid api base url: " + err.Error())
	}

	c := &Client{
		keyID:             keyID,
		issuerID:          issuerID,
		privateKeyContent: privateKey,
		audience:          targetAudience,

		client:  httpClient,
		BaseURL: baseURL,
		tracker: tracker,
	}
	c.common.client = c
	c.Provisioning = (*ProvisioningService)(&c.common)

	return c
}

// ensureSignedToken makes sure that the JWT auth token is not expired
// and return a signed key
func (c *Client) ensureSignedToken() (string, error) {
	if c.token != nil {
		err := c.token.Claims.Valid()
		if err == nil {
			return c.signedToken, nil
		}

		log.Debugf("JWT token is invalid: %s, regenerating...", err)
	} else {
		log.Debugf("Generating JWT token")
	}

	c.token = createToken(c.keyID, c.issuerID, c.audience)
	var err error
	if c.signedToken, err = signToken(c.token, c.privateKeyContent); err != nil {
		c.tracker.TrackAuthError(fmt.Sprintf("JWT signing: %s", err.Error()))
		return "", err
	}
	return c.signedToken, nil
}

// NewRequestWithRelationshipURL ...
func (c *Client) NewRequestWithRelationshipURL(method, endpoint string, body interface{}) (*http.Request, error) {
	endpoint = strings.TrimPrefix(endpoint, c.BaseURL.String()+apiVersion+"/")

	return c.NewRequest(method, endpoint, body)
}

// NewRequest creates a new http.Request
func (c *Client) NewRequest(method, endpoint string, body interface{}) (*http.Request, error) {
	endpoint = apiVersion + "/" + endpoint

	return c.newRequest(method, endpoint, body)
}

func (c *Client) newRequest(method, endpoint string, body interface{}) (*http.Request, error) {
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
	startTime := time.Now()

	c.Debugf("Request:")
	if c.EnableDebugLogs {
		if err := httputil.PrintRequest(req); err != nil {
			c.Debugf("Failed to print request: %s", err)
		}
	}

	resp, err := c.client.Do(req)
	duration := time.Since(startTime)

	c.Debugf("Response:")
	if c.EnableDebugLogs {
		if err := httputil.PrintResponse(resp); err != nil {
			c.Debugf("Failed to print response: %s", err)
		}
	}

	if err != nil {
		c.tracker.TrackAPIError(req.Method, req.URL.Host, req.URL.Path, 0, err.Error())
		return nil, err
	}

	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Warnf("Failed to close response body: %s", cerr)
		}
	}()

	if err := checkResponse(resp); err != nil {
		c.tracker.TrackAPIRequest(req.Method, req.URL.Host, req.URL.Path, resp.StatusCode, duration)
		c.tracker.TrackAPIError(req.Method, req.URL.Host, req.URL.Path, resp.StatusCode, err.Error())
		return resp, err
	}

	c.tracker.TrackAPIRequest(req.Method, req.URL.Host, req.URL.Path, resp.StatusCode, duration)


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
