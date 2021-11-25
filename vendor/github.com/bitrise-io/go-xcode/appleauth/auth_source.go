package appleauth

import (
	"fmt"
	"time"

	"github.com/bitrise-io/go-xcode/devportalservice"
)

// Source returns a specific kind (Apple ID/API Key) Apple authentication data from a specific source (Bitrise Apple Developer Connection, Step inputs)
type Source interface {
	Fetch(connection *devportalservice.AppleDeveloperConnection, inputs Inputs) (*Credentials, error)
	Description() string
}

// ConnectionAPIKeySource provides API Key from Bitrise Apple Developer Connection
type ConnectionAPIKeySource struct{}

// InputAPIKeySource provides API Key from Step inputs
type InputAPIKeySource struct{}

// ConnectionAppleIDSource provides Apple ID from Bitrise Apple Developer Connection
type ConnectionAppleIDSource struct{}

// InputAppleIDSource provides Apple ID from Step inputs
type InputAppleIDSource struct{}

// ConnectionAppleIDFastlaneSource provides Apple ID from Bitrise Apple Developer Connection, includes Fastlane specific session
type ConnectionAppleIDFastlaneSource struct{}

// InputAppleIDFastlaneSource provides Apple ID from Step inputs, includes Fastlane specific session
type InputAppleIDFastlaneSource struct{}

// Description ...
func (*ConnectionAPIKeySource) Description() string {
	return "Bitrise Apple Developer Connection with API key found"
}

// Fetch ...
func (*ConnectionAPIKeySource) Fetch(conn *devportalservice.AppleDeveloperConnection, inputs Inputs) (*Credentials, error) {
	if conn == nil || conn.APIKeyConnection == nil { // Not configured
		return nil, nil
	}

	return &Credentials{
		APIKey: conn.APIKeyConnection,
	}, nil
}

//

// Description ...
func (*InputAPIKeySource) Description() string {
	return "Inputs with API key authentication found"
}

// Fetch ...
func (*InputAPIKeySource) Fetch(conn *devportalservice.AppleDeveloperConnection, inputs Inputs) (*Credentials, error) {
	if inputs.APIKeyPath == "" { // Not configured
		return nil, nil
	}

	privateKey, keyID, err := fetchPrivateKey(inputs.APIKeyPath)
	if err != nil {
		return nil, fmt.Errorf("could not fetch private key (%s) specified as input: %v", inputs.APIKeyPath, err)
	}
	if len(privateKey) == 0 {
		return nil, fmt.Errorf("private key (%s) is empty", inputs.APIKeyPath)
	}

	return &Credentials{
		APIKey: &devportalservice.APIKeyConnection{
			IssuerID:   inputs.APIIssuer,
			KeyID:      keyID,
			PrivateKey: string(privateKey),
		},
	}, nil
}

//

// Description ...
func (*ConnectionAppleIDSource) Description() string {
	return "Bitrise Apple Developer Connection with Apple ID found."
}

// Fetch ...
func (*ConnectionAppleIDSource) Fetch(conn *devportalservice.AppleDeveloperConnection, inputs Inputs) (*Credentials, error) {
	if conn == nil || conn.AppleIDConnection == nil { // No Apple ID configured
		return nil, nil
	}

	return &Credentials{
		AppleID: &AppleID{
			Username:            conn.AppleIDConnection.AppleID,
			Password:            conn.AppleIDConnection.Password,
			Session:             "",
			AppSpecificPassword: appSpecificPasswordFavouringConnection(conn.AppleIDConnection, inputs.AppSpecificPassword),
		},
	}, nil
}

//

// Description ...
func (*InputAppleIDSource) Description() string {
	return "Inputs with Apple ID authentication found."
}

// Fetch ...
func (*InputAppleIDSource) Fetch(conn *devportalservice.AppleDeveloperConnection, inputs Inputs) (*Credentials, error) {
	if inputs.Username == "" { // Not configured
		return nil, nil
	}

	return &Credentials{
		AppleID: &AppleID{
			Username:            inputs.Username,
			Password:            inputs.Password,
			AppSpecificPassword: inputs.AppSpecificPassword,
		},
	}, nil
}

//

// Description ...
func (*ConnectionAppleIDFastlaneSource) Description() string {
	return "Bitrise Apple Developer Connection with Apple ID found."
}

// Fetch ...
func (*ConnectionAppleIDFastlaneSource) Fetch(conn *devportalservice.AppleDeveloperConnection, inputs Inputs) (*Credentials, error) {
	if conn == nil || conn.AppleIDConnection == nil { // No Apple ID configured
		return nil, nil
	}

	appleIDConn := conn.AppleIDConnection
	if appleIDConn.SessionExpiryDate != nil && appleIDConn.SessionExpiryDate.Before(time.Now()) {
		return nil, fmt.Errorf("2FA session saved in Bitrise Developer Connection is expired, was valid until %s", appleIDConn.SessionExpiryDate.String())
	}
	session, err := appleIDConn.FastlaneLoginSession()
	if err != nil {
		return nil, fmt.Errorf("could not prepare Fastlane session cookie object: %v", err)
	}

	return &Credentials{
		AppleID: &AppleID{
			Username:            conn.AppleIDConnection.AppleID,
			Password:            conn.AppleIDConnection.Password,
			Session:             session,
			AppSpecificPassword: appSpecificPasswordFavouringConnection(conn.AppleIDConnection, inputs.AppSpecificPassword),
		},
	}, nil
}

//

// Description ...
func (*InputAppleIDFastlaneSource) Description() string {
	return "Inputs with Apple ID authentication found. This method does not support TFA enabled Apple IDs."
}

// Fetch ...
func (*InputAppleIDFastlaneSource) Fetch(conn *devportalservice.AppleDeveloperConnection, inputs Inputs) (*Credentials, error) {
	if inputs.Username == "" { // Not configured
		return nil, nil
	}

	return &Credentials{
		AppleID: &AppleID{
			Username:            inputs.Username,
			Password:            inputs.Password,
			AppSpecificPassword: inputs.AppSpecificPassword,
		},
	}, nil
}

func appSpecificPasswordFavouringConnection(conn *devportalservice.AppleIDConnection, passwordFromInput string) string {
	appSpecificPassword := passwordFromInput

	// AppSpecifcPassword from the connection overwrites the one from the input
	if conn != nil && conn.AppSpecificPassword != "" {
		appSpecificPassword = conn.AppSpecificPassword
	}

	return appSpecificPassword
}
