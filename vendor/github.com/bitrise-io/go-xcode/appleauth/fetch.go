package appleauth

import (
	"fmt"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-xcode/devportalservice"
)

// Credentials contains either Apple ID or APIKey auth info
type Credentials struct {
	AppleID *AppleID
	APIKey  *devportalservice.APIKeyConnection
}

// AppleID contains Apple ID auth info
//
// Without 2FA:
//   Required: username, password
// With 2FA:
//   Required: username, password, appSpecificPassword
//			   session (Only for Fastlane, set as FASTLANE_SESSION)
//
// As Fastlane spaceship uses:
//  - iTMSTransporter: it requires Username + Password (or App-specific password with 2FA)
//  - TunesAPI: it requires Username + Password (+ 2FA session with 2FA)
type AppleID struct {
	Username, Password           string
	Session, AppSpecificPassword string
}

// MissingAuthConfigError is returned in case no usable Apple App Store Connect / Developer Portal authenticaion is found
type MissingAuthConfigError struct {
}

func (*MissingAuthConfigError) Error() string {
	return "no credentials provided"
}

// Select return valid Apple ID or API Key based authentication data, from the provided Bitrise Apple Developer Connection or Inputs
// authSources: required, array of checked sources (in order, the first set one will be used)
//	 for example: []AppleAuthSource{&SourceConnectionAPIKey{}, &SourceConnectionAppleID{}, &SourceInputAPIKey{}, &SourceInputAppleID{}}
// inputs: optional, user provided inputs that are not centrally managed (by setting up connections)
func Select(conn *devportalservice.AppleDeveloperConnection, authSources []Source, inputs Inputs) (Credentials, error) {
	for _, source := range authSources {
		auth, err := source.Fetch(conn, inputs)
		if err != nil {
			return Credentials{}, err
		}

		if auth != nil {
			fmt.Println()
			log.Infof("%s", source.Description())

			return *auth, nil
		}
	}

	return Credentials{}, &MissingAuthConfigError{}
}
