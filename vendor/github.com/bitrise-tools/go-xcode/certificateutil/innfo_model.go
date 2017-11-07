package certificateutil

import (
	"crypto/x509"
	"fmt"
	"strings"
	"time"
)

// CertificateInfoModel ...
type CertificateInfoModel struct {
	CommonName string
	TeamName   string
	TeamID     string
	EndDate    time.Time
	StartDate  time.Time

	Serial string

	certificate x509.Certificate
}

// CheckValidity ...
func (info CertificateInfoModel) CheckValidity() error {
	timeNow := time.Now()
	if !timeNow.After(info.StartDate) {
		return fmt.Errorf("Certificate is not yet valid - validity starts at: %s", info.StartDate)
	}
	if !timeNow.Before(info.EndDate) {
		return fmt.Errorf("Certificate is not valid anymore - validity ended at: %s", info.EndDate)
	}
	return nil
}

// NewCertificateInfo ...
func NewCertificateInfo(certificate x509.Certificate) CertificateInfoModel {
	return CertificateInfoModel{
		CommonName:  certificate.Subject.CommonName,
		TeamName:    strings.Join(certificate.Subject.Organization, " "),
		TeamID:      strings.Join(certificate.Subject.OrganizationalUnit, " "),
		EndDate:     certificate.NotAfter,
		StartDate:   certificate.NotBefore,
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

// FilterValidCertificateInfos ...
func FilterValidCertificateInfos(certificateInfos []CertificateInfoModel) []CertificateInfoModel {
	certificateInfosByName := map[string]CertificateInfoModel{}

	for _, certificateInfo := range certificateInfos {
		if certificateInfo.CheckValidity() == nil {
			activeCertificate, ok := certificateInfosByName[certificateInfo.CommonName]
			if !ok || certificateInfo.EndDate.After(activeCertificate.EndDate) {
				certificateInfosByName[certificateInfo.CommonName] = certificateInfo
			}
		}
	}

	validCertificates := []CertificateInfoModel{}
	for _, validCertificate := range certificateInfosByName {
		validCertificates = append(validCertificates, validCertificate)
	}
	return validCertificates
}
