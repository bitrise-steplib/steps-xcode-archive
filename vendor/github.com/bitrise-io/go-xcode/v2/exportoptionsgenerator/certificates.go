package exportoptionsgenerator

import "github.com/bitrise-io/go-xcode/certificateutil"

// CodesignIdentityProvider can list certificate infos.
type CodesignIdentityProvider interface {
	ListCodesignIdentities() ([]certificateutil.CertificateInfoModel, error)
}

// LocalCodesignIdentityProvider ...
type LocalCodesignIdentityProvider struct{}

// ListCodesignIdentities ...
func (p LocalCodesignIdentityProvider) ListCodesignIdentities() ([]certificateutil.CertificateInfoModel, error) {
	certs, err := certificateutil.InstalledCodesigningCertificateInfos()
	if err != nil {
		return nil, err
	}
	certInfo := certificateutil.FilterValidCertificateInfos(certs)
	return append(certInfo.ValidCertificates, certInfo.DuplicatedCertificates...), nil
}
