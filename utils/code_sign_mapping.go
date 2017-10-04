package utils

import (
	"sort"

	"github.com/bitrise-io/steps-certificate-and-profile-installer/certificateutil"
	"github.com/bitrise-io/steps-certificate-and-profile-installer/profileutil"
	"github.com/bitrise-tools/go-xcode/xcodeproj"
	glob "github.com/ryanuber/go-glob"
)

// ResolveCodeSignMapping ...
func ResolveCodeSignMapping(codeSignInfoMap map[string]xcodeproj.CodeSignInfo, exportMethod string, profiles []profileutil.ProfileModel, certificates []certificateutil.CertificateInfosModel) (certificateutil.CertificateInfosModel, map[string]profileutil.ProfileModel) {
	sort.Sort(ByBundleIDLength(profiles))

	bundleIDTeamIDmap := map[string]xcodeproj.CodeSignInfo{}
	for _, val := range codeSignInfoMap {
		bundleIDTeamIDmap[val.BundleIdentifier] = val
	}

	filtered := map[string]profileutil.ProfileModel{}

	groupedProfiles := map[string][]profileutil.ProfileModel{}

	for _, profile := range profiles {
		for _, embeddedCert := range profile.DeveloperCertificates {
			if embeddedCert.RawSubject == "" {
				continue
			}
			isCertInstalled := false
			for _, installedCert := range certificates {
				if embeddedCert.RawSubject == installedCert.RawSubject && embeddedCert.RawEndDate == installedCert.RawEndDate {
					isCertInstalled = true
					break
				}
			}
			if !isCertInstalled {
				continue
			}
			if _, ok := groupedProfiles[embeddedCert.RawSubject]; !ok {
				groupedProfiles[embeddedCert.RawSubject] = []profileutil.ProfileModel{}
			}
			groupedProfiles[embeddedCert.RawSubject] = append(groupedProfiles[embeddedCert.RawSubject], profile)
		}
	}

	for certSubject, profiles := range groupedProfiles {
		certSubjectFound := false
		for _, profile := range profiles {
			foundProfiles := map[string]profileutil.ProfileModel{}
			skipMatching := false
			for bundleIDToCheck, codesignInfo := range bundleIDTeamIDmap {
				if codesignInfo.ProvisioningProfileSpecifier == profile.Name {
					foundProfiles[bundleIDToCheck] = profile
					skipMatching = true
					continue
				}
			}
			if !skipMatching {
				for bundleIDToCheck, codesignInfo := range bundleIDTeamIDmap {
					if codesignInfo.ProvisioningProfile == profile.UUID {
						foundProfiles[bundleIDToCheck] = profile
						skipMatching = true
						continue
					}
				}
			}
			if !skipMatching {
				for bundleIDToCheck, codesignInfo := range bundleIDTeamIDmap {
					if glob.Glob(profile.BundleIdentifier, bundleIDToCheck) && exportMethod == string(profile.ExportType) && profile.TeamIdentifier == codesignInfo.DevelopmentTeam {
						foundProfiles[bundleIDToCheck] = profile
						skipMatching = true
						continue
					}
				}
			}
			if !skipMatching {
				for bundleIDToCheck := range bundleIDTeamIDmap {
					if glob.Glob(profile.BundleIdentifier, bundleIDToCheck) && exportMethod == string(profile.ExportType) {
						foundProfiles[bundleIDToCheck] = profile
						continue
					}
				}
			}
			if len(foundProfiles) >= len(bundleIDTeamIDmap) {
				certSubjectFound = true
				filtered = foundProfiles
				break
			}
		}
		if certSubjectFound {
			for _, cert := range certificates {
				if cert.RawSubject == certSubject {
					return cert, filtered
				}
			}
			break
		}
	}

	return certificateutil.CertificateInfosModel{}, nil
}
