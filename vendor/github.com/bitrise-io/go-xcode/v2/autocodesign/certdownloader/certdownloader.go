// Package certdownloader implements a autocodesign.CertificateProvider which fetches Bitrise hosted Xcode codesigning certificates.
package certdownloader

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bitrise-io/go-steputils/input"
	"github.com/bitrise-io/go-utils/filedownloader"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-xcode/certificateutil"
	"github.com/bitrise-io/go-xcode/v2/autocodesign"
)

// CertificateAndPassphrase contains a p12 file URL and passphrase
type CertificateAndPassphrase struct {
	URL, Passphrase string
}

type downloader struct {
	certs  []CertificateAndPassphrase
	client *http.Client
}

// NewDownloader ...
func NewDownloader(certs []CertificateAndPassphrase, client *http.Client) autocodesign.CertificateProvider {
	return downloader{
		certs:  certs,
		client: client,
	}
}

// GetCertificates ...
func (d downloader) GetCertificates() ([]certificateutil.CertificateInfoModel, error) {
	var certInfos []certificateutil.CertificateInfoModel

	for i, p12 := range d.certs {
		log.Debugf("Downloading p12 file number %d from %s", i, p12.URL)

		certInfo, err := downloadAndParsePKCS12(d.client, p12.URL, p12.Passphrase)
		if err != nil {
			return nil, err
		}

		log.Debugf("Codesign identities included:\n%s", certsToString(certInfo))
		certInfos = append(certInfos, certInfo...)
	}

	return certInfos, nil
}

// downloadAndParsePKCS12 downloads a pkcs12 format file and parses certificates and matching private keys.
func downloadAndParsePKCS12(httpClient *http.Client, certificateURL, passphrase string) ([]certificateutil.CertificateInfoModel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	downloader := filedownloader.NewWithContext(ctx, httpClient)
	fileProvider := input.NewFileProvider(downloader)

	contents, err := fileProvider.Contents(certificateURL)
	if err != nil {
		return nil, err
	} else if contents == nil {
		return nil, fmt.Errorf("certificate (%s) is empty", certificateURL)
	}

	infos, err := certificateutil.CertificatesFromPKCS12Content(contents, passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate (%s), err: %s", certificateURL, err)
	}

	return infos, nil
}

func certsToString(certs []certificateutil.CertificateInfoModel) (s string) {
	for i, cert := range certs {
		s += "- "
		s += cert.String()
		if i < len(certs)-1 {
			s += "\n"
		}
	}
	return
}
