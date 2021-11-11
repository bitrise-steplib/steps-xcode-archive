package main

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-io/go-xcode/autocodesign/certdownloader"
)

// ValidateCertificates validates if the number of certificate URLs matches those of passphrases
func ValidateCertificates(certificateList string, passphraseList stepconf.Secret) ([]string, []string, error) {
	pfxURLs := splitAndClean(certificateList, "|", true)
	if len(pfxURLs) == 0 {
		return []string{}, []string{}, nil
	}

	passphrases := splitAndClean(string(passphraseList), "|", false)
	if len(pfxURLs) != len(passphrases) {
		return nil, nil, fmt.Errorf("certificates count (%d) and passphrases count (%d) should match", len(pfxURLs), len(passphrases))
	}

	return pfxURLs, passphrases, nil
}

// Certificates returns an array of p12 file URLs and passphrases
func Certificates(certificateList string, passphraseList stepconf.Secret) ([]certdownloader.CertificateAndPassphrase, error) {
	pfxURLs, passphrases, err := ValidateCertificates(certificateList, passphraseList)
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
