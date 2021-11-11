package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-xcode/appleauth"
	"github.com/bitrise-io/go-xcode/devportalservice"
)

const notConnected = `Bitrise Apple service connection not found.
Most likely because there is no configured Bitrise Apple service connection.
Read more: https://devcenter.bitrise.io/getting-started/configuring-bitrise-steps-that-require-apple-developer-account-data/`

func manageCodeSigning(opts RunOpts) (*devportalservice.APIKeyConnection, error) {
	if opts.XcodeMajorVersion < 13 {
		log.Warnf("Skipping Code Signing, at least Xcode 13 is required for Cloud Signing")

		return nil, nil
	}

	authConfig, err := appleauth.Select(&opts.AppleServiceConnection, []appleauth.Source{&appleauth.ConnectionAPIKeySource{}}, appleauth.Inputs{})
	if err != nil {
		if authConfig.APIKey == nil {
			fmt.Println()
			log.Warnf("%s", notConnected)
		}

		if errors.Is(err, &appleauth.MissingAuthConfigError{}) {
			return nil, nil
		}

		return nil, fmt.Errorf("could not configure Apple service authentication: %v", err)
	}

	logger.Infof("API Key found")
	return authConfig.APIKey, nil
}

func writePrivateKey(contents []byte) (string, error) {
	privatekeyFile, err := os.CreateTemp("", "apiKey*.p8")
	if err != nil {
		return "", fmt.Errorf("failed to create private key file: %s", err)
	}

	if _, err := privatekeyFile.Write([]byte(contents)); err != nil {
		return "", fmt.Errorf("failed to write private key: %s", err)
	}

	if err := privatekeyFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close private key file: %s", err)
	}

	return privatekeyFile.Name(), nil
}
