package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/env"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-io/go-xcode/appleauth"
	"github.com/bitrise-io/go-xcode/autocodesign"
	"github.com/bitrise-io/go-xcode/autocodesign/certdownloader"
	"github.com/bitrise-io/go-xcode/autocodesign/codesignasset"
	"github.com/bitrise-io/go-xcode/autocodesign/devportalclient"
	"github.com/bitrise-io/go-xcode/autocodesign/keychain"
	"github.com/bitrise-io/go-xcode/autocodesign/projectmanager"
	"github.com/bitrise-io/go-xcode/devportalservice"
)

const notConnected = `Bitrise Apple service connection not found.
Most likely because there is no configured Bitrise Apple service connection.
Read more: https://devcenter.bitrise.io/getting-started/configuring-bitrise-steps-that-require-apple-developer-account-data/`

type CodeSignOpts struct {
	CodeSigningStrategy

	ProjectPath       string
	Scheme            string
	Configuration     string
	ExportMethod      string
	XcodeMajorVersion int

	CertificateURLList        string
	CertificatePassphraseList stepconf.Secret

	AppleServiceConnection devportalservice.AppleDeveloperConnection
	KeychainPath           string
	KeychainPassword       stepconf.Secret
}

type CodeSigningStrategy int

const (
	noCodeSign CodeSigningStrategy = iota
	codeSigningXcode
	codeSigningBitriseAPIKey
	codeSigningBitriseAppleID
)

func manageCodeSigning(opts CodeSignOpts) (*devportalservice.APIKeyConnection, error) {
	strategy, err := selectCodeSigningStrategy(opts)
	if err != nil {
		return nil, err
	}

	switch strategy {
	case noCodeSign:
		logger.Infof("Skip downloading any Code Signing assets")
		return nil, nil
	case codeSigningXcode:
		{
			logger.Infof("Xcode-managed Code Signing selected")

			authConfig, err := appleauth.Select(&opts.AppleServiceConnection, []appleauth.Source{&appleauth.ConnectionAPIKeySource{}}, appleauth.Inputs{})
			if err != nil {
				if authConfig.APIKey == nil {
					fmt.Println()
					logger.Warnf("%s", notConnected)
				}

				if errors.Is(err, &appleauth.MissingAuthConfigError{}) {
					return nil, nil
				}

				return nil, fmt.Errorf("could not configure Apple service authentication: %v", err)
			}

			logger.Infof("Downloading certificates from Bitrise")
			if err := downloadAndInstallCertificates(opts.CertificateURLList, opts.CertificatePassphraseList, opts.KeychainPath, opts.KeychainPassword); err != nil {
				return nil, err
			}

			return authConfig.APIKey, nil
		}
	case codeSigningBitriseAPIKey:
		{
			logger.Infof("Bitrise Code Signing with Apple API key")
			if err := bitriseCodeSign(opts, devportalclient.APIKeyClient); err != nil {
				return nil, err
			}

			return nil, nil
		}
	case codeSigningBitriseAppleID:
		{
			logger.Infof("Bitrise Code Signing with Apple ID")
			if err := bitriseCodeSign(opts, devportalclient.AppleIDClient); err != nil {
				return nil, err
			}

			return nil, nil
		}
	}

	return nil, nil
}

func selectCodeSigningStrategy(opts CodeSignOpts) (CodeSigningStrategy, error) {
	if opts.CodeSigningStrategy == codeSigningBitriseAppleID {
		if opts.AppleServiceConnection.AppleIDConnection == nil {
			return noCodeSign, fmt.Errorf("Apple ID authentication is selected in step inputs, but connection is not set up properly.")
		}
		return codeSigningBitriseAppleID, nil
	}

	if opts.CodeSigningStrategy == codeSigningBitriseAPIKey {
		if opts.AppleServiceConnection.APIKeyConnection == nil {
			return noCodeSign, fmt.Errorf("Apple API key authentication is selected in step inputs, but connection is not set up properly.")
		}

		if opts.XcodeMajorVersion < 13 {
			return codeSigningBitriseAPIKey, nil
		}

		project, err := projectmanager.NewProject(projectmanager.InitParams{
			ProjectOrWorkspacePath: opts.ProjectPath,
			SchemeName:             opts.Scheme,
			ConfigurationName:      opts.Configuration,
		})
		if err != nil {
			return noCodeSign, err
		}

		managedSigning, err := project.IsSigningManagedAutomatically()
		if err != nil {
			return noCodeSign, err
		}

		if managedSigning {
			return codeSigningXcode, nil
		}

		return codeSigningBitriseAPIKey, nil
	}

	return noCodeSign, nil
}

func downloadAndInstallCertificates(urls string, passphrases stepconf.Secret, keychainPath string, keychainPassword stepconf.Secret) error {
	certificateAndPassphrase, err := Certificates(urls, passphrases)
	if err != nil {
		return err
	}

	downloader := certdownloader.NewDownloader(certificateAndPassphrase, retry.NewHTTPClient().StandardClient())
	certificates, err := downloader.GetCertificates()
	if err != nil {
		return fmt.Errorf("failed to download certificates: %s", err)
	}

	repository := env.NewRepository()
	keychainWriter, err := keychain.New(keychainPath, keychainPassword, command.NewFactory(repository))
	if err != nil {
		return fmt.Errorf("failed to open Keychain: %s", err)
	}

	logger.Infof("Installing downloaded certificates:")
	for _, cert := range certificates {
		// Empty passphrase provided, as already parsed certificate + private key
		if err := keychainWriter.InstallCertificate(cert, ""); err != nil {
			return err
		}
		logger.Infof("- %s (serial: %s", cert.CommonName, cert.Serial)
	}

	return nil
}

// TODO: Does not register devices
func bitriseCodeSign(opts CodeSignOpts, authType devportalclient.ClientType) error {
	minProfileValidity := 30
	verboseLog := true

	// Parse certificate list inputs
	certificateAndPassphrase, err := Certificates(opts.CertificateURLList, opts.CertificatePassphraseList)
	if err != nil {
		return err
	}

	// Analyze project
	fmt.Println()
	log.Infof("Analyzing project")
	project, err := projectmanager.NewProject(projectmanager.InitParams{
		ProjectOrWorkspacePath: opts.ProjectPath,
		SchemeName:             opts.Scheme,
		ConfigurationName:      opts.Configuration,
	})
	if err != nil {
		return err
	}

	appLayout, err := project.GetAppLayout(false)
	if err != nil {
		return err
	}

	clientFactory := devportalclient.NewClientFactory()
	devPortalClient, err := clientFactory.CreateClient(authType, appLayout.TeamID, opts.AppleServiceConnection)
	if err != nil {
		return err
	}

	// Create codesign manager
	if opts.KeychainPath == "" {
		return fmt.Errorf("no keychain path specified")
	}
	keychain, err := keychain.New(opts.KeychainPath, opts.KeychainPassword, command.NewFactory(env.NewRepository()))
	if err != nil {
		return fmt.Errorf("failed to initialize keychain: %s", err)
	}

	certDownloader := certdownloader.NewDownloader(certificateAndPassphrase, retry.NewHTTPClient().StandardClient())
	manager := autocodesign.NewCodesignAssetManager(devPortalClient, certDownloader, codesignasset.NewWriter(*keychain))

	// Auto codesign
	distribution := autocodesign.DistributionType(opts.ExportMethod)
	codesignAssetsByDistributionType, err := manager.EnsureCodesignAssets(appLayout, autocodesign.CodesignAssetsOpts{
		DistributionType:       distribution,
		BitriseTestDevices:     []devportalservice.TestDevice{},
		MinProfileValidityDays: minProfileValidity,
		VerboseLog:             verboseLog,
	})
	if err != nil {
		return fmt.Errorf("automatic code signing failed: %s", err)
	}

	if err := project.ForceCodesignAssets(distribution, codesignAssetsByDistributionType); err != nil {
		return fmt.Errorf("failed to force codesign settings: %s", err)
	}

	return nil
}

func writePrivateKey(contents []byte) (string, error) {
	privatekeyFile, err := os.CreateTemp("", "apiKey*.p8")
	if err != nil {
		return "", fmt.Errorf("failed to create private key file: %s", err)
	}

	if _, err := privatekeyFile.Write(contents); err != nil {
		return "", fmt.Errorf("failed to write private key: %s", err)
	}

	if err := privatekeyFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close private key file: %s", err)
	}

	return privatekeyFile.Name(), nil
}
