package utils

import (
	"sort"

	"github.com/bitrise-tools/go-xcode/exportoptions"
	glob "github.com/ryanuber/go-glob"

	"github.com/bitrise-io/steps-certificate-and-profile-installer/certificateutil"
	"github.com/bitrise-io/steps-certificate-and-profile-installer/profileutil"
	"github.com/bitrise-tools/go-xcode/xcodeproj"
)

// ResolveCodeSignMapping ...
func ResolveCodeSignMapping(codeSignInfoMap map[string]xcodeproj.CodeSignInfo, exportMethod exportoptions.Method, profiles []profileutil.ProfileModel, certificates []certificateutil.CertificateInfosModel) (certificateutil.CertificateInfosModel, map[string]profileutil.ProfileModel) {
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
		profileFound := true
		filtered = map[string]profileutil.ProfileModel{}
		for _, profile := range profiles {
			bundleIDFound := false
			for bundleID, codesignInfo := range bundleIDTeamIDmap {
				if codesignInfo.ProvisioningProfileSpecifier != "" && profile.Name != "" {
					if codesignInfo.ProvisioningProfileSpecifier == profile.Name {
						bundleIDFound = true
						filtered[bundleID] = profile
						continue
					}
				}
				if codesignInfo.ProvisioningProfile != "" && profile.UUID != "" {
					if codesignInfo.ProvisioningProfile == profile.UUID {
						bundleIDFound = true
						filtered[bundleID] = profile
						continue
					}
				}
				if glob.Glob(profile.BundleIdentifier, bundleID) && exportMethod == profile.ExportType && profile.TeamIdentifier == codesignInfo.DevelopmentTeam {
					bundleIDFound = true
					filtered[bundleID] = profile
					continue
				}
				if glob.Glob(profile.BundleIdentifier, bundleID) && exportMethod == profile.ExportType {
					bundleIDFound = true
					filtered[bundleID] = profile
					continue
				}
			}
			if !bundleIDFound {
				profileFound = false
			}
		}

		if profileFound {
			for _, cert := range certificates {
				if cert.RawSubject == certSubject {
					return cert, filtered
				}
			}
		}
	}

	return certificateutil.CertificateInfosModel{}, nil
}
