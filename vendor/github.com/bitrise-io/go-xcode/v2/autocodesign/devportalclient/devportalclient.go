// Package devportalclient contains glue code to select and initialize a autocodesign.DevPortalClient either using Apple ID or API key authentication.
package devportalclient

import (
	"fmt"
	"net/http"

	"github.com/bitrise-io/go-steputils/v2/ruby"
	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/appleauth"
	"github.com/bitrise-io/go-xcode/devportalservice"
	"github.com/bitrise-io/go-xcode/v2/autocodesign"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/devportalclient/appstoreconnect"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/devportalclient/appstoreconnectclient"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/devportalclient/spaceship"
)

const (
	// NotConnectedWarning ...
	NotConnectedWarning = `Bitrise Apple Service connection not found.
Most likely because there is no configured Bitrise Apple Service connection.
Read more: https://devcenter.bitrise.io/getting-started/configuring-bitrise-steps-that-require-apple-developer-account-data/`
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
	f.logger.Infof("Fetching Apple Service connection")
	connectionProvider := devportalservice.NewBitriseClient(retry.NewHTTPClient().StandardClient(), buildURL, buildAPIToken)
	conn, err := connectionProvider.GetAppleDeveloperConnection()
	if err != nil {
		if networkErr, ok := err.(devportalservice.NetworkError); ok && networkErr.Status == http.StatusUnauthorized {
			f.logger.Println()
			f.logger.Warnf("Unauthorized to query Bitrise Apple Service connection. This happens by design, with a public app's PR build, to protect secrets.")
			return nil, err
		}

		f.logger.Println()
		f.logger.Errorf("Failed to activate Bitrise Apple Service connection")
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
		f.logger.Debugf("App Store Connect API client created with base URL: %s", client.BaseURL)
	} else if credentials.AppleID != nil {
		cmdFactory, err := ruby.NewCommandFactory(command.NewFactory(env.NewRepository()), env.NewCommandLocator())
		if err != nil {
			return nil, err
		}

		client, err := spaceship.NewClient(*credentials.AppleID, teamID, cmdFactory)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Apple ID client: %v", err)
		}
		devportalClient = spaceship.NewSpaceshipDevportalClient(client)
		f.logger.Donef("Apple ID client created")
	}

	return devportalClient, nil
}
