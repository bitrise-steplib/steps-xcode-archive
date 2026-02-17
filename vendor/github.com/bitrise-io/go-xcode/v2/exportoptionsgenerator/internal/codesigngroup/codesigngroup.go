package codesigngroup

import (
	"sort"

	"github.com/bitrise-io/go-xcode/certificateutil"
	"github.com/bitrise-io/go-xcode/profileutil"
	"github.com/ryanuber/go-glob"
)

// CodeSignGroup ...
type CodeSignGroup interface {
	Certificate() certificateutil.CertificateInfoModel
	InstallerCertificate() *certificateutil.CertificateInfoModel
	BundleIDProfileMap() map[string]profileutil.ProvisioningProfileInfoModel
}

// SelectableCodeSignGroup ...
type SelectableCodeSignGroup struct {
	Certificate         certificateutil.CertificateInfoModel
	BundleIDProfilesMap map[string][]profileutil.ProvisioningProfileInfoModel
}

func containsCertificate(installedCertificates []certificateutil.CertificateInfoModel, certificate certificateutil.CertificateInfoModel) bool {
	for _, cert := range installedCertificates {
		if cert.Serial == certificate.Serial {
			return true
		}
	}
	return false
}

// BuildFilterableList ...
func BuildFilterableList(installedCertificates []certificateutil.CertificateInfoModel, profiles []profileutil.ProvisioningProfileInfoModel, bundleIDs []string) []SelectableCodeSignGroup {
	groups := []SelectableCodeSignGroup{}

	serialToProfiles := map[string][]profileutil.ProvisioningProfileInfoModel{}
	serialToCertificate := map[string]certificateutil.CertificateInfoModel{}
	for _, profile := range profiles {
		for _, certificate := range profile.DeveloperCertificates {
			if !containsCertificate(installedCertificates, certificate) {
				continue
			}

			certificateProfiles, ok := serialToProfiles[certificate.Serial]
			if !ok {
				certificateProfiles = []profileutil.ProvisioningProfileInfoModel{}
			}
			certificateProfiles = append(certificateProfiles, profile)
			serialToProfiles[certificate.Serial] = certificateProfiles
			serialToCertificate[certificate.Serial] = certificate
		}
	}

	for serial, profiles := range serialToProfiles {
		certificate := serialToCertificate[serial]

		bundleIDToProfiles := map[string][]profileutil.ProvisioningProfileInfoModel{}
		for _, bundleID := range bundleIDs {

			matchingProfiles := []profileutil.ProvisioningProfileInfoModel{}
			for _, profile := range profiles {
				if !glob.Glob(profile.BundleID, bundleID) {
					continue
				}

				matchingProfiles = append(matchingProfiles, profile)
			}

			if len(matchingProfiles) > 0 {
				sort.Sort(ByBundleIDLength(matchingProfiles))
				bundleIDToProfiles[bundleID] = matchingProfiles
			}
		}

		if len(bundleIDToProfiles) == len(bundleIDs) {
			group := SelectableCodeSignGroup{
				Certificate:         certificate,
				BundleIDProfilesMap: bundleIDToProfiles,
			}
			groups = append(groups, group)
		}
	}

	return groups
}

// ByBundleIDLength ...
type ByBundleIDLength []profileutil.ProvisioningProfileInfoModel

// Len ..
func (s ByBundleIDLength) Len() int {
	return len(s)
}

// Swap ...
func (s ByBundleIDLength) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less ...
func (s ByBundleIDLength) Less(i, j int) bool {
	return len(s[i].BundleID) > len(s[j].BundleID)
}
