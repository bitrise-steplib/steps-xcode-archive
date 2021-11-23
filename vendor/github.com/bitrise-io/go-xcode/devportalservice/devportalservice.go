package devportalservice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
)

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// AppleDeveloperConnectionProvider ...
type AppleDeveloperConnectionProvider interface {
	GetAppleDeveloperConnection() (*AppleDeveloperConnection, error)
}

// BitriseClient implements AppleDeveloperConnectionProvider through the Bitrise.io API.
type BitriseClient struct {
	httpClient              httpClient
	buildURL, buildAPIToken string

	readBytesFromFile func(pth string) ([]byte, error)
}

// NewBitriseClient creates a new instance of BitriseClient.
func NewBitriseClient(client httpClient, buildURL, buildAPIToken string) *BitriseClient {
	return &BitriseClient{
		httpClient:        client,
		buildURL:          buildURL,
		buildAPIToken:     buildAPIToken,
		readBytesFromFile: fileutil.ReadBytesFromFile,
	}
}

const appleDeveloperConnectionPath = "apple_developer_portal_data.json"

func privateKeyWithHeader(privateKey string) string {
	if strings.HasPrefix(privateKey, "-----BEGIN PRIVATE KEY----") {
		return privateKey
	}

	return fmt.Sprint(
		"-----BEGIN PRIVATE KEY-----\n",
		privateKey,
		"\n-----END PRIVATE KEY-----",
	)
}

// GetAppleDeveloperConnection fetches the Bitrise.io Apple Developer connection.
func (c *BitriseClient) GetAppleDeveloperConnection() (*AppleDeveloperConnection, error) {
	var rawCreds []byte
	var err error

	if strings.HasPrefix(c.buildURL, "file://") {
		rawCreds, err = c.readBytesFromFile(strings.TrimPrefix(c.buildURL, "file://"))
	} else {
		rawCreds, err = c.download()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch authentication credentials: %v", err)
	}

	type data struct {
		*AppleIDConnection
		*APIKeyConnection
		TestDevices []TestDevice `json:"test_devices"`
	}
	var d data
	if err := json.Unmarshal([]byte(rawCreds), &d); err != nil {
		return nil, fmt.Errorf("failed to unmarshal authentication credentials from response (%s): %s", rawCreds, err)
	}

	if d.APIKeyConnection != nil {
		if d.APIKeyConnection.IssuerID == "" {
			return nil, fmt.Errorf("invalid authentication credentials, empty issuer_id in response (%s)", rawCreds)
		}
		if d.APIKeyConnection.KeyID == "" {
			return nil, fmt.Errorf("invalid authentication credentials, empty key_id in response (%s)", rawCreds)
		}
		if d.APIKeyConnection.PrivateKey == "" {
			return nil, fmt.Errorf("invalid authentication credentials, empty private_key in response (%s)", rawCreds)
		}

		d.APIKeyConnection.PrivateKey = privateKeyWithHeader(d.APIKeyConnection.PrivateKey)
	}

	testDevices, duplicatedDevices := validateTestDevice(d.TestDevices)

	return &AppleDeveloperConnection{
		AppleIDConnection:     d.AppleIDConnection,
		APIKeyConnection:      d.APIKeyConnection,
		TestDevices:           testDevices,
		DuplicatedTestDevices: duplicatedDevices,
	}, nil
}

func (c *BitriseClient) download() ([]byte, error) {
	url := fmt.Sprintf("%s/%s", c.buildURL, appleDeveloperConnectionPath)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for URL (%s): %s", url, err)
	}
	req.Header.Add("BUILD_API_TOKEN", c.buildAPIToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// On error, any Response can be ignored
		return nil, fmt.Errorf("failed to perform request: %s", err)
	}

	// The client must close the response body when finished with it
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Warnf("Failed to close response body: %s", cerr)
		}
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, NetworkError{Status: resp.StatusCode}
	}

	return body, nil
}

type cookie struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Value     string `json:"value"`
	Domain    string `json:"domain"`
	Secure    bool   `json:"secure"`
	Expires   string `json:"expires,omitempty"`
	MaxAge    int    `json:"max_age,omitempty"`
	Httponly  bool   `json:"httponly"`
	ForDomain *bool  `json:"for_domain,omitempty"`
}

// AppleIDConnection represents a Bitrise.io Apple ID-based Apple Developer connection.
type AppleIDConnection struct {
	AppleID             string              `json:"apple_id"`
	Password            string              `json:"password"`
	AppSpecificPassword string              `json:"app_specific_password"`
	SessionExpiryDate   *time.Time          `json:"connection_expiry_date"`
	SessionCookies      map[string][]cookie `json:"session_cookies"`
}

// APIKeyConnection represents a Bitrise.io API key-based Apple Developer connection.
type APIKeyConnection struct {
	KeyID      string `json:"key_id"`
	IssuerID   string `json:"issuer_id"`
	PrivateKey string `json:"private_key"`
}

// TestDevice ...
type TestDevice struct {
	ID     int `json:"id"`
	UserID int `json:"user_id"`
	// DeviceID is the Apple device UDID
	DeviceID   string    `json:"device_identifier"`
	Title      string    `json:"title"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	DeviceType string    `json:"device_type"`
}

// IsEqualUDID compares two UDIDs (stored in the DeviceID field of TestDevice)
func IsEqualUDID(UDID string, otherUDID string) bool {
	return normalizeDeviceUDID(UDID) == normalizeDeviceUDID(otherUDID)
}

// AppleDeveloperConnection represents a Bitrise.io Apple Developer connection.
// https://devcenter.bitrise.io/getting-started/configuring-bitrise-steps-that-require-apple-developer-account-data/
type AppleDeveloperConnection struct {
	AppleIDConnection                  *AppleIDConnection
	APIKeyConnection                   *APIKeyConnection
	TestDevices, DuplicatedTestDevices []TestDevice
}

// FastlaneLoginSession returns the Apple ID login session in a ruby/object:HTTP::Cookie format.
// The session can be used as a value for FASTLANE_SESSION environment variable: https://docs.fastlane.tools/best-practices/continuous-integration/#two-step-or-two-factor-auth.
func (c *AppleIDConnection) FastlaneLoginSession() (string, error) {
	var rubyCookies []string
	for _, cookie := range c.SessionCookies["https://idmsa.apple.com"] {
		if rubyCookies == nil {
			rubyCookies = append(rubyCookies, "---"+"\n")
		}

		if cookie.ForDomain == nil {
			b := true
			cookie.ForDomain = &b
		}

		tmpl, err := template.New("").Parse(`- !ruby/object:HTTP::Cookie
  name: {{.Name}}
  value: {{.Value}}
  domain: {{.Domain}}
  for_domain: {{.ForDomain}}
  path: "{{.Path}}"
`)
		if err != nil {
			return "", fmt.Errorf("failed to parse template: %s", err)
		}

		var b bytes.Buffer
		err = tmpl.Execute(&b, cookie)
		if err != nil {
			return "", fmt.Errorf("failed to execute template on cookie: %s: %s", cookie.Name, err)
		}

		rubyCookies = append(rubyCookies, b.String()+"\n")
	}
	return strings.Join(rubyCookies, ""), nil
}

func validDeviceUDID(udid string) string {
	r := regexp.MustCompile("[^a-zA-Z0-9-]")
	return r.ReplaceAllLiteralString(udid, "")
}

func normalizeDeviceUDID(udid string) string {
	return strings.ToLower(strings.ReplaceAll(validDeviceUDID(udid), "-", ""))
}

// validateTestDevice filters out duplicated devices
// it does not change UDID casing or remove '-' separator, only to filter out whitespace or unsupported characters
func validateTestDevice(deviceList []TestDevice) (validDevices, duplicatedDevices []TestDevice) {
	bitriseDevices := make(map[string]bool)
	for _, device := range deviceList {
		normalizedID := normalizeDeviceUDID(device.DeviceID)
		if _, ok := bitriseDevices[normalizedID]; ok {
			duplicatedDevices = append(duplicatedDevices, device)

			continue
		}

		bitriseDevices[normalizedID] = true
		device.DeviceID = validDeviceUDID(device.DeviceID)
		validDevices = append(validDevices, device)
	}

	return validDevices, duplicatedDevices
}

// WritePrivateKeyToFile ...
func (c *APIKeyConnection) WritePrivateKeyToFile() (string, error) {
	privatekeyFile, err := os.CreateTemp("", "apiKey*.p8")
	if err != nil {
		return "", fmt.Errorf("failed to create private key file: %s", err)
	}

	if _, err := privatekeyFile.Write([]byte(c.PrivateKey)); err != nil {
		return "", fmt.Errorf("failed to write private key: %s", err)
	}

	if err := privatekeyFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close private key file: %s", err)
	}

	return privatekeyFile.Name(), nil
}
