package codesigngroup

import (
	"slices"

	"github.com/bitrise-io/go-xcode/certificateutil"
	"github.com/bitrise-io/go-xcode/v2/profileutil"
	"github.com/ryanuber/go-glob"
)

// SelectableCodeSignGroup ...
type SelectableCodeSignGroup struct {
	Certificate         certificateutil.CertificateInfoModel
	BundleIDProfilesMap map[string][]profileutil.ProvisioningProfileInfoModel
}

// BuildFilterableList ...
func BuildFilterableList(installedCertificates []certificateutil.CertificateInfoModel, profiles []profileutil.ProvisioningProfileInfoModel, bundleIDs []string) []SelectableCodeSignGroup {
	var groups []SelectableCodeSignGroup

	serialToProfiles := map[string][]profileutil.ProvisioningProfileInfoModel{}
	serialToCertificate := map[string]certificateutil.CertificateInfoModel{}
	for _, profile := range profiles {
		for _, certificate := range profile.DeveloperCertificates {
			if !containsCertificate(installedCertificates, certificate) {
				continue
			}

			serialToProfiles[certificate.Serial] = append(serialToProfiles[certificate.Serial], profile)
			serialToCertificate[certificate.Serial] = certificate
		}
	}

	for serial, profiles := range serialToProfiles {
		certificate := serialToCertificate[serial]

		bundleIDToProfiles := map[string][]profileutil.ProvisioningProfileInfoModel{}
		for _, bundleID := range bundleIDs {
			var matchingProfiles []profileutil.ProvisioningProfileInfoModel
			for _, profile := range profiles {
				if !glob.Glob(profile.BundleID, bundleID) {
					continue
				}

				matchingProfiles = append(matchingProfiles, profile)
			}

			if len(matchingProfiles) > 0 {
				slices.SortFunc(matchingProfiles, func(a, b profileutil.ProvisioningProfileInfoModel) int {
					return len(b.BundleID) - len(a.BundleID)
				})
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

func containsCertificate(list []certificateutil.CertificateInfoModel, item certificateutil.CertificateInfoModel) bool {
	return slices.ContainsFunc(list, func(cert certificateutil.CertificateInfoModel) bool {
		return cert.Serial == item.Serial
	})
}
