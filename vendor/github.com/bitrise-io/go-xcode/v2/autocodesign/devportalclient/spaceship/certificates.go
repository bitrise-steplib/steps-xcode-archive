package spaceship

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/bitrise-io/go-xcode/certificateutil"
	"github.com/bitrise-io/go-xcode/v2/autocodesign"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/devportalclient/appstoreconnect"
)

// CertificateSource ...
type CertificateSource struct {
	client       *Client
	certificates map[appstoreconnect.CertificateType][]autocodesign.Certificate
}

// NewSpaceshipCertificateSource ...
func NewSpaceshipCertificateSource(client *Client) *CertificateSource {
	return &CertificateSource{
		client: client,
	}
}

// QueryCertificateBySerial ...
func (s *CertificateSource) QueryCertificateBySerial(serial big.Int) (autocodesign.Certificate, error) {
	if s.certificates == nil {
		if err := s.downloadAll(); err != nil {
			return autocodesign.Certificate{}, err
		}
	}

	allCerts := append(s.certificates[appstoreconnect.IOSDevelopment], s.certificates[appstoreconnect.IOSDistribution]...)
	for _, cert := range allCerts {
		if serial.Cmp(cert.CertificateInfo.Certificate.SerialNumber) == 0 {
			return cert, nil
		}
	}

	return autocodesign.Certificate{}, fmt.Errorf("can not find certificate with serial (%s)", serial.Text(16))
}

// QueryAllIOSCertificates ...
func (s *CertificateSource) QueryAllIOSCertificates() (map[appstoreconnect.CertificateType][]autocodesign.Certificate, error) {
	if s.certificates == nil {
		if err := s.downloadAll(); err != nil {
			return nil, err
		}
	}

	return s.certificates, nil
}

func (s *CertificateSource) downloadAll() error {
	fmt.Printf("Fetching developer certificates")

	devCerts, err := s.getCertificates(true)
	if err != nil {
		return err
	}

	fmt.Printf("Fetching distribution certificates")

	distCers, err := s.getCertificates(false)
	if err != nil {
		return err
	}

	s.certificates = map[appstoreconnect.CertificateType][]autocodesign.Certificate{
		appstoreconnect.IOSDevelopment:  devCerts,
		appstoreconnect.IOSDistribution: distCers,
	}

	return nil
}

type certificatesResponse struct {
	Data []struct {
		Content string `json:"content"`
		ID      string `json:"id"`
	} `json:"data"`
}

func (s *CertificateSource) getCertificates(devCerts bool) ([]autocodesign.Certificate, error) {
	var output string
	var err error
	if devCerts {
		output, err = s.client.runSpaceshipCommand("list_dev_certs")
	} else {
		output, err = s.client.runSpaceshipCommand("list_dist_certs")
	}
	if err != nil {
		return nil, err
	}

	var certificates certificatesResponse
	if err := json.Unmarshal([]byte(output), &certificates); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	var certInfos []autocodesign.Certificate
	for _, certInfo := range certificates.Data {
		pemContent, err := base64.StdEncoding.DecodeString(certInfo.Content)
		if err != nil {
			return nil, err
		}

		cert, err := certificateutil.CeritifcateFromPemContent(pemContent)
		if err != nil {
			return nil, err
		}

		certInfos = append(certInfos, autocodesign.Certificate{
			CertificateInfo: certificateutil.NewCertificateInfo(*cert, nil),
			ID:              certInfo.ID,
		})
	}

	return certInfos, nil
}
