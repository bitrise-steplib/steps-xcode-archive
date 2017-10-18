package certificateutil

import (
	"crypto/x509"
	"strings"
	"time"
)

// CertificateInfoModel ...
type CertificateInfoModel struct {
	CommonName string
	TeamName   string
	TeamID     string
	EndDate    time.Time

	Serial string

	certificate x509.Certificate
}

// IsExpired ...
func (info CertificateInfoModel) IsExpired() bool {
	if info.EndDate.IsZero() {
		return false
	}
	return info.EndDate.Before(time.Now())
}

// NewCertificateInfo ...
func NewCertificateInfo(certificate x509.Certificate) CertificateInfoModel {
	return CertificateInfoModel{
		CommonName:  certificate.Subject.CommonName,
		TeamName:    strings.Join(certificate.Subject.Organization, " "),
		TeamID:      strings.Join(certificate.Subject.OrganizationalUnit, " "),
		EndDate:     certificate.NotAfter,
		Serial:      certificate.SerialNumber.String(),
		certificate: certificate,
	}
}

// CertificateInfos ...
func CertificateInfos(certificates []*x509.Certificate) []CertificateInfoModel {
	infos := []CertificateInfoModel{}
	for _, certificate := range certificates {
		if certificate != nil {
			info := NewCertificateInfo(*certificate)
			infos = append(infos, info)
		}
	}

	return infos
}

// NewCertificateInfosFromPKCS12 ...
func NewCertificateInfosFromPKCS12(pkcs12Pth, password string) ([]CertificateInfoModel, error) {
	certificates, err := CertificatesFromPKCS12File(pkcs12Pth, password)
	if err != nil {
		return nil, err
	}
	return CertificateInfos(certificates), nil
}

// InstalledCodesigningCertificateInfos ...
func InstalledCodesigningCertificateInfos() ([]CertificateInfoModel, error) {
	certificates, err := InstalledCodesigningCertificates()
	if err != nil {
		return nil, err
	}
	return CertificateInfos(certificates), nil
}
