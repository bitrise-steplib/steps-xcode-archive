// Package certdownloader implements a autocodesign.CertificateProvider which fetches Bitrise hosted Xcode codesigning certificates.
package certdownloader

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/filedownloader"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-io/go-xcode/certificateutil"
	"github.com/bitrise-io/go-xcode/v2/autocodesign"
)

// CertificateAndPassphrase contains a p12 file URL and passphrase
type CertificateAndPassphrase struct {
	URL, Passphrase string
}

type downloader struct {
	certs        []CertificateAndPassphrase
	logger       log.Logger
	fileProvider stepconf.FileProvider
}

// NewDownloader ...
func NewDownloader(certs []CertificateAndPassphrase, logger log.Logger) autocodesign.CertificateProvider {
	fileDownloader := filedownloader.NewDownloader(logger)
	fileProvider := stepconf.NewFileProvider(fileDownloader, fileutil.NewFileManager(), pathutil.NewPathProvider(), pathutil.NewPathModifier())

	return downloader{
		certs:        certs,
		logger:       logger,
		fileProvider: fileProvider,
	}
}

// GetCertificates ...
func (d downloader) GetCertificates() ([]certificateutil.CertificateInfoModel, error) {
	var certInfos []certificateutil.CertificateInfoModel

	for i, p12 := range d.certs {
		d.logger.Debugf("Downloading p12 file number %d from %s", i, p12.URL)

		certInfo, err := d.downloadAndParsePKCS12(p12.URL, p12.Passphrase)
		if err != nil {
			return nil, err
		}

		d.logger.Debugf("Codesign identities included:\n%s", certsToString(certInfo))
		certInfos = append(certInfos, certInfo...)
	}

	return certInfos, nil
}

// downloadAndParsePKCS12 downloads a pkcs12 format file and parses certificates and matching private keys.
func (d downloader) downloadAndParsePKCS12(certificateURL, passphrase string) ([]certificateutil.CertificateInfoModel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	contentReader, err := d.fileProvider.Contents(ctx, certificateURL)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := contentReader.Close(); err != nil {
			d.logger.Warnf("Failed to close certificate reader: %s", err)
		}
	}()
	contents, err := io.ReadAll(contentReader)
	if err != nil {
		return nil, err
	}
	if len(contents) == 0 {
		return nil, fmt.Errorf("certificate is empty: %s", certificateURL)
	}

	info, err := certificateutil.CertificatesFromPKCS12Content(contents, passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate (%s), err: %s", certificateURL, err)
	}

	return info, nil
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
