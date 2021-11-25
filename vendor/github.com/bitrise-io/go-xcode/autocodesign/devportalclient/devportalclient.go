// Package devportalclient contains glue code to select and initialize a autocodesign.DevPortalClient either using Apple ID or API key authentication.
package devportalclient

import (
	"fmt"
	"net/http"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-io/go-xcode/appleauth"
	"github.com/bitrise-io/go-xcode/autocodesign"
	"github.com/bitrise-io/go-xcode/autocodesign/devportalclient/appstoreconnect"
	"github.com/bitrise-io/go-xcode/autocodesign/devportalclient/appstoreconnectclient"
	"github.com/bitrise-io/go-xcode/autocodesign/devportalclient/spaceship"
	"github.com/bitrise-io/go-xcode/devportalservice"
)

const (
	// NotConnectedWarning ...
	NotConnectedWarning = `Bitrise Apple Service connection not found.
Most likely because there is no configured Bitrise Apple service connection.
Read more: https://devcenter.bitrise.io/getting-started/configuring-bitrise-steps-that-require-apple-developer-account-data/`
	// NotConnectedLocalTestingInfo ...
	NotConnectedLocalTestingInfo = `For testing purposes please provide BITRISE_BUILD_URL as json file (file://path-to-json) while setting BITRISE_BUILD_API_TOKEN to any non-empty string.`
)

// Factory ...
type Factory struct {
	logger log.Logger
}

// NewFactory ...
func NewFactory(logger log.Logger) Factory {
	return Factory{
		logger: logger,
	}
}

// CreateBitriseConnection ...
func (f Factory) CreateBitriseConnection(buildURL, buildAPIToken string) (*devportalservice.AppleDeveloperConnection, error) {
	f.logger.Println()
	f.logger.Infof("Fetching Apple service connection")
	connectionProvider := devportalservice.NewBitriseClient(retry.NewHTTPClient().StandardClient(), buildURL, buildAPIToken)
	conn, err := connectionProvider.GetAppleDeveloperConnection()
	if err != nil {
		if networkErr, ok := err.(devportalservice.NetworkError); ok && networkErr.Status == http.StatusUnauthorized {
			f.logger.Println()
			f.logger.Warnf("Unauthorized to query Bitrise Apple service connection. This happens by design, with a public app's PR build, to protect secrets.")
			return nil, err
		}

		f.logger.Println()
		f.logger.Errorf("Failed to activate Bitrise Apple service connection")
		f.logger.Warnf("Read more: https://devcenter.bitrise.io/getting-started/configuring-bitrise-steps-that-require-apple-developer-account-data/")

		return nil, err
	}

	if len(conn.DuplicatedTestDevices) != 0 {
		f.logger.Debugf("Devices with duplicated UDID are registered on Bitrise, will be ignored:")
		for _, d := range conn.DuplicatedTestDevices {
			f.logger.Debugf("- %s, %s, UDID (%s), added at %s", d.Title, d.DeviceType, d.DeviceID, d.UpdatedAt)
		}
	}

	return conn, nil
}

// Create ...
func (f Factory) Create(credentials appleauth.Credentials, teamID string) (autocodesign.DevPortalClient, error) {
	f.logger.Println()
	f.logger.Infof("Initializing Developer Portal client")
	var devportalClient autocodesign.DevPortalClient
	if credentials.APIKey != nil {
		httpClient := appstoreconnect.NewRetryableHTTPClient()
		client := appstoreconnect.NewClient(httpClient, credentials.APIKey.KeyID, credentials.APIKey.IssuerID, []byte(credentials.APIKey.PrivateKey))
		client.EnableDebugLogs = false // Turn off client debug logs including HTTP call debug logs
		devportalClient = appstoreconnectclient.NewAPIDevPortalClient(client)
		f.logger.Donef("App Store Connect API client created with base URL: %s", client.BaseURL)
	} else if credentials.AppleID != nil {
		client, err := spaceship.NewClient(*credentials.AppleID, teamID)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Apple ID client: %v", err)
		}
		devportalClient = spaceship.NewSpaceshipDevportalClient(client)
		f.logger.Donef("Apple ID client created")
	}

	return devportalClient, nil
}
