package codesign

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-io/go-xcode/autocodesign"
	"github.com/bitrise-io/go-xcode/autocodesign/certdownloader"
	"github.com/bitrise-io/go-xcode/autocodesign/keychain"
)

// StepInputParser ...
type StepInputParser struct {
	AuthType                  AuthType
	DistributionMethod        string
	CertificateURLList        string
	CertificatePassphraseList stepconf.Secret
	KeychainPath              string
	KeychainPassword          stepconf.Secret

	CommandFactory command.Factory
}

// ParsedConfig ...
type ParsedConfig struct {
	CertificatesAndPassphrases []certdownloader.CertificateAndPassphrase
	Keychain                   keychain.Keychain
	DistributionMethod         autocodesign.DistributionType
}

// Parse validates and parses step inputs related to code signing, and returns with a ParsedConfig
func (p StepInputParser) Parse() (ParsedConfig, error) {
	var (
		certificatesAndPassphrases []certdownloader.CertificateAndPassphrase
		keychainWriter             keychain.Keychain
	)

	if p.AuthType != NoAuth {
		if strings.TrimSpace(p.CertificateURLList) == "" {
			return ParsedConfig{}, fmt.Errorf("Code signing certificate URL: required variable is not present")
		}
		if strings.TrimSpace(p.KeychainPath) == "" {
			return ParsedConfig{}, fmt.Errorf("Keychain path: required variable is not present")
		}
		if strings.TrimSpace(string(p.KeychainPassword)) == "" {
			return ParsedConfig{}, fmt.Errorf("Keychain password: required variable is not present")
		}

		var err error
		certificatesAndPassphrases, err = p.ParseCertificates()
		if err != nil {
			return ParsedConfig{}, fmt.Errorf("failed to parse certificate URL and passphrase inputs: %s", err)
		}

		keychainP, err := keychain.New(p.KeychainPath, p.KeychainPassword, p.CommandFactory)
		if err != nil {
			return ParsedConfig{}, fmt.Errorf("failed to open keychain: %s", err)
		}
		keychainWriter = *keychainP
	}

	return ParsedConfig{
		CertificatesAndPassphrases: certificatesAndPassphrases,
		Keychain:                   keychainWriter,
		DistributionMethod:         p.ParseDistributionMethod(),
	}, nil

}

// ParseDistributionMethod ...
func (p StepInputParser) ParseDistributionMethod() autocodesign.DistributionType {
	return autocodesign.DistributionType(p.DistributionMethod)
}

// validateCertificates validates if the number of certificate URLs matches those of passphrases
func (p StepInputParser) validateCertificates() ([]string, []string, error) {
	pfxURLs := splitAndClean(p.CertificateURLList, "|", true)
	passphrases := splitAndClean(string(p.CertificatePassphraseList), "|", false)

	if len(pfxURLs) != len(passphrases) {
		return nil, nil, fmt.Errorf("certificates count (%d) and passphrases count (%d) should match", len(pfxURLs), len(passphrases))
	}

	return pfxURLs, passphrases, nil
}

// ParseCertificates returns an array of p12 file URLs and passphrases
func (p StepInputParser) ParseCertificates() ([]certdownloader.CertificateAndPassphrase, error) {
	pfxURLs, passphrases, err := p.validateCertificates()
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

// SplitAndClean ...
func splitAndClean(list string, sep string, omitEmpty bool) (items []string) {
	return sliceutil.CleanWhitespace(strings.Split(list, sep), omitEmpty)
}
