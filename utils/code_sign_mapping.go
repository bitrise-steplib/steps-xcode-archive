package utils

import (
	"sort"

	"github.com/bitrise-tools/go-xcode/exportoptions"
	glob "github.com/ryanuber/go-glob"

	"github.com/bitrise-io/steps-certificate-and-profile-installer/certificateutil"
	"github.com/bitrise-io/steps-certificate-and-profile-installer/profileutil"
)

// ProfileGroup ...
type ProfileGroup struct {
	Certificate certificateutil.CertificateInfosModel
	Profiles    map[string]profileutil.ProfileModel
}

// ResolveCodeSignMapping ...
func ResolveCodeSignMapping(bundleIDs []string, exportMethod exportoptions.Method, profiles []profileutil.ProfileModel, certificates []certificateutil.CertificateInfosModel) []ProfileGroup {
	profileGroups := groupInstalledProfilesByInstalledEmbeddedCertificateSubjects(profiles, certificates)
	return findProfilesByBundleIDsAndExportMethod(profileGroups, bundleIDs, exportMethod)
}

func isCertificateInstalled(installedCertificates []certificateutil.CertificateInfosModel, certificate certificateutil.CertificateInfosModel) bool {
	isCertInstalled := false
	for _, installedCert := range installedCertificates {
		if certificate.RawSubject == installedCert.RawSubject && certificate.RawEndDate == installedCert.RawEndDate {
			isCertInstalled = true
			break
		}
	}
	return isCertInstalled
}

func groupInstalledProfilesByInstalledEmbeddedCertificateSubjects(profiles []profileutil.ProfileModel, certificates []certificateutil.CertificateInfosModel) map[string][]profileutil.ProfileModel {
	groupedProfiles := map[string][]profileutil.ProfileModel{}
	for _, profile := range profiles {
		for _, embeddedCert := range profile.DeveloperCertificates {
			if embeddedCert.RawSubject == "" {
				continue
			}
			if !isCertificateInstalled(certificates, embeddedCert) {
				continue
			}

			if _, ok := groupedProfiles[embeddedCert.RawSubject]; !ok {
				groupedProfiles[embeddedCert.RawSubject] = []profileutil.ProfileModel{}
			}
			groupedProfiles[embeddedCert.RawSubject] = append(groupedProfiles[embeddedCert.RawSubject], profile)
		}
	}

	return groupedProfiles
}

func findProfilesByBundleIDsAndExportMethod(profileGroups map[string][]profileutil.ProfileModel, bundleIDs []string, exportMethod exportoptions.Method) []ProfileGroup {
	filteredProfileGroups := []ProfileGroup{}
	for certSubject, profiles := range profileGroups {
		sort.Sort(ByBundleIDLength(profiles))

		bundleIDProfilePairs := map[string]profileutil.ProfileModel{}
		profileFound := 0
		for _, profile := range profiles {
			for _, bundleID := range bundleIDs {
				if glob.Glob(profile.BundleIdentifier, bundleID) && exportMethod == profile.ExportType {
					profileFound++
					bundleIDProfilePairs[bundleID] = profile
				}
			}
		}

		if profileFound == len(bundleIDs) {
			cert := certificateutil.CertificateInfosModel{}
			for _, profile := range profiles {
				for _, embeddedCert := range profile.DeveloperCertificates {
					if certSubject == embeddedCert.RawSubject {
						cert = embeddedCert
					}
				}
			}

			filteredProfileGroups = append(filteredProfileGroups, ProfileGroup{Certificate: cert, Profiles: bundleIDProfilePairs})
		}
	}

	return filteredProfileGroups
}
