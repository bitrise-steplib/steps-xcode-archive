package codesign

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-xcode/v2/autocodesign"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/certdownloader"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/keychain"
)

// Input ...
type Input struct {
	AuthType                  AuthType
	DistributionMethod        string
	CertificateURLList        string
	CertificatePassphraseList stepconf.Secret
	KeychainPath              string
	KeychainPassword          stepconf.Secret
}

// Config ...
type Config struct {
	CertificatesAndPassphrases []certdownloader.CertificateAndPassphrase
	Keychain                   keychain.Keychain
	DistributionMethod         autocodesign.DistributionType
}

// ParseConfig validates and parses step inputs related to code signing and returns with a Config
func ParseConfig(input Input, cmdFactory command.Factory) (Config, error) {
	certificatesAndPassphrases, err := parseCertificates(input)
	if err != nil {
		return Config{}, fmt.Errorf("failed to parse certificate URL and passphrase inputs: %s", err)
	}

	keychainWriter, err := keychain.New(input.KeychainPath, input.KeychainPassword, cmdFactory)
	if err != nil {
		return Config{}, fmt.Errorf("failed to open keychain: %s", err)
	}

	return Config{
		CertificatesAndPassphrases: certificatesAndPassphrases,
		Keychain:                   *keychainWriter,
		DistributionMethod:         autocodesign.DistributionType(input.DistributionMethod),
	}, nil
}

// parseCertificates returns an array of p12 file URLs and passphrases
func parseCertificates(input Input) ([]certdownloader.CertificateAndPassphrase, error) {
	if strings.TrimSpace(input.CertificateURLList) == "" {
		return nil, fmt.Errorf("code signing certificate URL: required input is not present")
	}
	if strings.TrimSpace(input.KeychainPath) == "" {
		return nil, fmt.Errorf("keychain path: required input is not present")
	}
	if strings.TrimSpace(string(input.KeychainPassword)) == "" {
		return nil, fmt.Errorf("keychain password: required input is not present")
	}

	pfxURLs, passphrases, err := validateCertificates(input.CertificateURLList, string(input.CertificatePassphraseList))
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

// validateCertificates validates if the number of certificate URLs matches those of passphrases
func validateCertificates(certURLList string, certPassphraseList string) ([]string, []string, error) {
	pfxURLs := splitAndClean(certURLList, "|", true)
	passphrases := splitAndClean(certPassphraseList, "|", false) // allow empty items because passphrase can be empty

	if len(pfxURLs) != len(passphrases) {
		return nil, nil, fmt.Errorf("certificate count (%d) and passphrase count (%d) should match", len(pfxURLs), len(passphrases))
	}

	return pfxURLs, passphrases, nil
}

// SplitAndClean ...
func splitAndClean(list string, sep string, omitEmpty bool) (items []string) {
	return sliceutil.CleanWhitespace(strings.Split(list, sep), omitEmpty)
}
