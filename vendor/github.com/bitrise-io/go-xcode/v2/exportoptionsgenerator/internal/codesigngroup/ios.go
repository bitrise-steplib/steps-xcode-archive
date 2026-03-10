package codesigngroup

import (
	"github.com/bitrise-io/go-xcode/certificateutil"
	"github.com/bitrise-io/go-xcode/profileutil"
)

// Ios ...
type Ios struct {
	certificate        certificateutil.CertificateInfoModel
	bundleIDProfileMap map[string]profileutil.ProvisioningProfileInfoModel
}

// NewIOSGroup ...
func NewIOSGroup(certificate certificateutil.CertificateInfoModel, bundleIDProfileMap map[string]profileutil.ProvisioningProfileInfoModel) *Ios {
	return &Ios{
		certificate:        certificate,
		bundleIDProfileMap: bundleIDProfileMap,
	}
}

// Certificate ...
func (signGroup *Ios) Certificate() certificateutil.CertificateInfoModel {
	return signGroup.certificate
}

// InstallerCertificate ...
func (signGroup *Ios) InstallerCertificate() *certificateutil.CertificateInfoModel {
	return nil
}

// BundleIDProfileMap ...
func (signGroup *Ios) BundleIDProfileMap() map[string]profileutil.ProvisioningProfileInfoModel {
	return signGroup.bundleIDProfileMap
}
