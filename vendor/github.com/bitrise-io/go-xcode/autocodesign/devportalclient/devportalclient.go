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

const notConnected = `Bitrise Apple service connection not found.
Most likely because there is no configured Bitrise Apple service connection.
Read more: https://devcenter.bitrise.io/getting-started/configuring-bitrise-steps-that-require-apple-developer-account-data/`

// ClientType ...
type ClientType int

const (
	// APIKeyClient ...
	APIKeyClient ClientType = iota
	// AppleIDClient ...
	AppleIDClient
)

// ClientFactory ...
type ClientFactory struct {
}

// NewClientFactory ...
func NewClientFactory() ClientFactory {
	return ClientFactory{}
}

// CreateBitriseConnection ...
func (f ClientFactory) CreateBitriseConnection(buildURL, buildAPIToken string) (devportalservice.AppleDeveloperConnection, error) {
	fmt.Println()
	log.Infof("Fetching Apple service connection")
	connectionProvider := devportalservice.NewBitriseClient(retry.NewHTTPClient().StandardClient(), buildURL, buildAPIToken)
	conn, err := connectionProvider.GetAppleDeveloperConnection()
	if err != nil {
		if networkErr, ok := err.(devportalservice.NetworkError); ok && networkErr.Status == http.StatusUnauthorized {
			fmt.Println()
			log.Warnf("Unauthorized to query Bitrise Apple service connection. This happens by design, with a public app's PR build, to protect secrets.")
			return devportalservice.AppleDeveloperConnection{}, err
		}

		fmt.Println()
		log.Errorf("Failed to activate Bitrise Apple service connection")
		log.Warnf("Read more: https://devcenter.bitrise.io/getting-started/configuring-bitrise-steps-that-require-apple-developer-account-data/")

		return devportalservice.AppleDeveloperConnection{}, err
	}

	if len(conn.DuplicatedTestDevices) != 0 {
		log.Debugf("Devices with duplicated UDID are registered on Bitrise, will be ignored:")
		for _, d := range conn.DuplicatedTestDevices {
			log.Debugf("- %s, %s, UDID (%s), added at %s", d.Title, d.DeviceType, d.DeviceID, d.UpdatedAt)
		}
	}

	return *conn, nil
}

// CreateClient ...
func (f ClientFactory) CreateClient(clientType ClientType, teamID string, conn devportalservice.AppleDeveloperConnection) (autocodesign.DevPortalClient, error) {
	var authSource appleauth.Source
	if clientType == APIKeyClient {
		authSource = &appleauth.ConnectionAPIKeySource{}
	} else {
		authSource = &appleauth.ConnectionAppleIDFastlaneSource{}
	}

	authConfig, err := appleauth.Select(&conn, []appleauth.Source{authSource}, appleauth.Inputs{})
	if err != nil {
		if conn.APIKeyConnection == nil && conn.AppleIDConnection == nil {
			fmt.Println()
			log.Warnf("%s", notConnected)
		}
		return nil, fmt.Errorf("could not configure Apple service authentication: %v", err)
	}

	if authConfig.APIKey != nil {
		log.Donef("Using Apple service connection with API key.")
	} else if authConfig.AppleID != nil {
		log.Donef("Using Apple service connection with Apple ID.")
	} else {
		panic("No Apple authentication credentials found.")
	}

	// create developer portal client
	fmt.Println()
	log.Infof("Initializing Developer Portal client")
	var devportalClient autocodesign.DevPortalClient
	if authConfig.APIKey != nil {
		httpClient := appstoreconnect.NewRetryableHTTPClient()
		client := appstoreconnect.NewClient(httpClient, authConfig.APIKey.KeyID, authConfig.APIKey.IssuerID, []byte(authConfig.APIKey.PrivateKey))
		client.EnableDebugLogs = false // Turn off client debug logs including HTTP call debug logs
		devportalClient = appstoreconnectclient.NewAPIDevPortalClient(client)
		log.Donef("App Store Connect API client created with base URL: %s", client.BaseURL)
	} else if authConfig.AppleID != nil {
		client, err := spaceship.NewClient(*authConfig.AppleID, teamID)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Apple ID client: %v", err)
		}
		devportalClient = spaceship.NewSpaceshipDevportalClient(client)
		log.Donef("Apple ID client created")
	}

	return devportalClient, nil
}
