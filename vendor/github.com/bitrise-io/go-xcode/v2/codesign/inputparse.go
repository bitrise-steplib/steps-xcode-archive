package codesign

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/retryhttp"
	"github.com/bitrise-io/go-xcode/v2/autocodesign"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/certdownloader"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/codesignasset"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/keychain"
	"github.com/bitrise-io/go-xcode/v2/devportalservice"
)

// Input ...
type Input struct {
	AuthType                     AuthType
	DistributionMethod           string
	CertificateURLList           string
	CertificatePassphraseList    stepconf.Secret
	KeychainPath                 string
	KeychainPassword             stepconf.Secret
	FallbackProvisioningProfiles string
}

// ConnectionOverrideInputs are used in steps to control the API key based auth credentials
// This overrides the global API connection defined on Bitrise.io
type ConnectionOverrideInputs struct {
	APIKeyPath              stepconf.Secret
	APIKeyID                string
	APIKeyIssuerID          string
	APIKeyEnterpriseAccount bool
}

// Config ...
type Config struct {
	CertificatesAndPassphrases   []certdownloader.CertificateAndPassphrase
	Keychain                     keychain.Keychain
	DistributionMethod           autocodesign.DistributionType
	FallbackProvisioningProfiles []string
}

// ParseConfig validates and parses step inputs related to code signing and returns with a Config
func ParseConfig(input Input, cmdFactory command.Factory) (Config, error) {
	certificatesAndPassphrases, err := parseCertificatesAndPassphrases(input.CertificateURLList, string(input.CertificatePassphraseList))
	if err != nil {
		return Config{}, err
	}

	if strings.TrimSpace(input.KeychainPath) == "" {
		return Config{}, fmt.Errorf("keychain path is not specified")
	}
	if strings.TrimSpace(string(input.KeychainPassword)) == "" {
		return Config{}, fmt.Errorf("keychain password is not specified")
	}

	keychainWriter, err := keychain.New(input.KeychainPath, input.KeychainPassword, cmdFactory)
	if err != nil {
		return Config{}, fmt.Errorf("failed to open keychain: %w", err)
	}

	fallbackProfiles, err := parseFallbackProvisioningProfiles(input.FallbackProvisioningProfiles)
	if err != nil {
		return Config{}, err
	}

	return Config{
		CertificatesAndPassphrases:   certificatesAndPassphrases,
		Keychain:                     *keychainWriter,
		DistributionMethod:           autocodesign.DistributionType(input.DistributionMethod),
		FallbackProvisioningProfiles: fallbackProfiles,
	}, nil
}

// parseConnectionOverrideConfig validates and parses the step input-level connection parameters
func parseConnectionOverrideConfig(keyPathOrURL stepconf.Secret, keyID, keyIssuerID string, isEnterpriseAccount bool, logger log.Logger) (*devportalservice.APIKeyConnection, error) {
	var key []byte
	if strings.HasPrefix(string(keyPathOrURL), "https://") {
		resp, err := retryhttp.NewClient(logger).Get(string(keyPathOrURL))
		if err != nil {
			return nil, fmt.Errorf("failed to download App Store Connect API key: %w", err)
		}

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				logger.Warnf(err.Error())
			}
		}(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("downloading App Store Connect API key failed with exit status %d: %s", resp.StatusCode, resp.Body)
		}

		key, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read App Store Connect API key download response: %w", err)
		}
	} else {
		trimmedPath := string(keyPathOrURL)
		if strings.HasPrefix(string(keyPathOrURL), "file://") {
			trimmedPath = strings.TrimPrefix(string(keyPathOrURL), "file://")
		}
		var err error
		key, err = os.ReadFile(trimmedPath)
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("App Store Connect API does not exist at %s", trimmedPath)
		} else if err != nil {
			return nil, fmt.Errorf("failed to read App Store Connect API at %s: %w", trimmedPath, err)
		}
	}

	return &devportalservice.APIKeyConnection{
		KeyID:             strings.TrimSpace(keyID),
		IssuerID:          strings.TrimSpace(keyIssuerID),
		PrivateKey:        string(key),
		EnterpriseAccount: isEnterpriseAccount,
	}, nil
}

// parseCertificatesAndPassphrases returns an array of p12 file URLs and passphrases
func parseCertificatesAndPassphrases(certificateURLList, certificatePassphraseList string) ([]certdownloader.CertificateAndPassphrase, error) {
	if strings.TrimSpace(certificateURLList) == "" {
		return nil, fmt.Errorf("code signing certificate URL is not specified")
	}

	pfxURLs, passphrases, err := splitCertificatesAndPassphrases(certificateURLList, string(certificatePassphraseList))
	if err != nil {
		return nil, err
	}

	files := make([]certdownloader.CertificateAndPassphrase, len(pfxURLs))
	for i, pfxURL := range pfxURLs {
		files[i] = certdownloader.CertificateAndPassphrase{
			URL:        pfxURL,
			Passphrase: passphrases[i],
		}
	}

	return files, nil
}

// splitCertificatesAndPassphrases validates if the number of certificate URLs matches those of passphrases
func splitCertificatesAndPassphrases(certURLList string, certPassphraseList string) ([]string, []string, error) {
	pfxURLs := splitAndClean(certURLList, "|", true)
	passphrases := splitAndClean(certPassphraseList, "|", false) // allow empty items because passphrase can be empty

	if len(pfxURLs) != len(passphrases) {
		return nil, nil, fmt.Errorf("code signing certificate count (%d) and passphrase count (%d) should match", len(pfxURLs), len(passphrases))
	}

	return pfxURLs, passphrases, nil
}

// SplitAndClean ...
func splitAndClean(list string, sep string, omitEmpty bool) (items []string) {
	return sliceutil.CleanWhitespace(strings.Split(list, sep), omitEmpty)
}

// parseFallbackProvisioningProfiles validates and expands profilesList.
// profilesList must be a list of paths separated either by `|` or `\n`.
// List items must be a remote (https://) or local (file://) file paths,
// or a local directory (with no scheme).
// For directory list items, the contained profiles' path will be returned.
func parseFallbackProvisioningProfiles(profilesList string) ([]string, error) {
	profiles := splitAndClean(profilesList, "\n", true)
	if len(profiles) == 1 {
		profiles = splitAndClean(profiles[0], "|", true)
	}

	var validProfiles []string
	for _, profile := range profiles {
		profileURL, err := url.Parse(profile)
		if err != nil {
			return []string{}, fmt.Errorf("invalid provisioning profile URL specified (%s): %w", profile, err)
		}

		// When file or https scheme provided, will fetch as a file
		if profileURL.Scheme != "" {
			validProfiles = append(validProfiles, profile)
			continue
		}

		// If no scheme is provided, assuming it is a local directory
		profilesInDir, err := listProfilesInDirectory(profile)
		if err != nil {
			return []string{}, err
		}

		validProfiles = append(validProfiles, profilesInDir...)
	}

	return validProfiles, nil
}

func listProfilesInDirectory(dir string) ([]string, error) {
	exists, err := pathutil.IsDirExists(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to check if provisioning profile path exists (%s): %w", dir, err)
	} else if !exists {
		return nil, fmt.Errorf("directory of provisioning profiles does not exist (%s)", dir)
	}

	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to list entries of the provisioning profiles directory (%s): %w", dir, err)
	}

	var profiles []string
	for _, dirEntry := range dirEntries {
		if dirEntry.Type().IsDir() || !dirEntry.Type().IsRegular() {
			continue
		}

		if strings.HasSuffix(dirEntry.Name(), codesignasset.ProfileIOSExtension) {
			profileURL := fmt.Sprintf("file://%s", filepath.Join(dir, dirEntry.Name()))
			profiles = append(profiles, profileURL)
		}
	}

	return profiles, nil
}
