package export

import (
	"fmt"
	"sort"

	"github.com/bitrise-tools/go-xcode/certificateutil"
	"github.com/bitrise-tools/go-xcode/plistutil"
	"github.com/bitrise-tools/go-xcode/profileutil"
	"github.com/ryanuber/go-glob"
)

func isCertificateInstalled(installedCertificates []certificateutil.CertificateInfoModel, certificate certificateutil.CertificateInfoModel) bool {
	installedMap := map[string]bool{}
	for _, certificate := range installedCertificates {
		installedMap[certificate.Serial] = true
	}
	return installedMap[certificate.Serial]
}

// CertificateProfilesGroup ...
type CertificateProfilesGroup struct {
	Certificate certificateutil.CertificateInfoModel
	Profiles    []profileutil.ProvisioningProfileInfoModel
}

func createCertificateProfilesGroups(certificates []certificateutil.CertificateInfoModel, profiles []profileutil.ProvisioningProfileInfoModel) []CertificateProfilesGroup {
	serialProfilesMap := map[string][]profileutil.ProvisioningProfileInfoModel{}
	serialCertificateMap := map[string]certificateutil.CertificateInfoModel{}
	for _, profile := range profiles {
		for _, certificate := range profile.DeveloperCertificates {
			if !isCertificateInstalled(certificates, certificate) {
				continue
			}

			certificateProfiles := serialProfilesMap[certificate.Serial]
			if certificateProfiles == nil {
				certificateProfiles = []profileutil.ProvisioningProfileInfoModel{}
			}
			certificateProfiles = append(certificateProfiles, profile)
			serialProfilesMap[certificate.Serial] = certificateProfiles
			serialCertificateMap[certificate.Serial] = certificate
		}
	}

	groups := []CertificateProfilesGroup{}
	for serial, profiles := range serialProfilesMap {
		certificate := serialCertificateMap[serial]
		group := CertificateProfilesGroup{
			Certificate: certificate,
			Profiles:    profiles,
		}
		groups = append(groups, group)
	}

	return groups
}

// SelectableCodeSignGroup ..
type SelectableCodeSignGroup struct {
	Certificate         certificateutil.CertificateInfoModel
	BundleIDProfilesMap map[string][]profileutil.ProvisioningProfileInfoModel
}

func createSelectableCodeSignGroups(certificateProfilesGroups []CertificateProfilesGroup, bundleIDCapabilitiesMap map[string]plistutil.PlistData) []SelectableCodeSignGroup {
	groups := []SelectableCodeSignGroup{}

	for _, certificateProfilesGroup := range certificateProfilesGroups {
		certificate := certificateProfilesGroup.Certificate
		profiles := certificateProfilesGroup.Profiles

		bundleIDProfilesMap := map[string][]profileutil.ProvisioningProfileInfoModel{}
		for bundleID, capabilities := range bundleIDCapabilitiesMap {

			matchingProfiles := []profileutil.ProvisioningProfileInfoModel{}
			for _, profile := range profiles {
				if !glob.Glob(profile.BundleID, bundleID) {
					continue
				}

				if missingCapabilities := profileutil.MatchTargetAndProfileEntitlements(capabilities, profile.Entitlements); len(missingCapabilities) > 0 {
					continue
				}

				matchingProfiles = append(matchingProfiles, profile)
			}

			if len(matchingProfiles) > 0 {
				sort.Sort(ByBundleIDLength(matchingProfiles))
				bundleIDProfilesMap[bundleID] = matchingProfiles
			}
		}

		if len(bundleIDProfilesMap) == len(bundleIDCapabilitiesMap) {
			group := SelectableCodeSignGroup{
				Certificate:         certificate,
				BundleIDProfilesMap: bundleIDProfilesMap,
			}
			groups = append(groups, group)
		}
	}

	return groups
}

// ResolveSelectableCodeSignGroups ...
func ResolveSelectableCodeSignGroups(certificates []certificateutil.CertificateInfoModel, profiles []profileutil.ProvisioningProfileInfoModel, bundleIDCapabilities map[string]plistutil.PlistData) []SelectableCodeSignGroup {
	certificateProfilesGroups := createCertificateProfilesGroups(certificates, profiles)
	return createSelectableCodeSignGroups(certificateProfilesGroups, bundleIDCapabilities)
}

// CodeSignGroup ...
type CodeSignGroup struct {
	Certificate        certificateutil.CertificateInfoModel
	BundleIDProfileMap map[string]profileutil.ProvisioningProfileInfoModel
}

func createCodeSignGroups(selectableGroups []SelectableCodeSignGroup) []CodeSignGroup {
	alreadyUsedProfileUUIDMap := map[string]bool{}

	singleWildcardGroups := []CodeSignGroup{}
	xcodeManagedGroups := []CodeSignGroup{}
	notXcodeManagedGroups := []CodeSignGroup{}
	remainingGroups := []CodeSignGroup{}

	for _, selectableGroup := range selectableGroups {
		certificate := selectableGroup.Certificate
		bundleIDProfilesMap := selectableGroup.BundleIDProfilesMap

		bundleIDs := []string{}
		profiles := []profileutil.ProvisioningProfileInfoModel{}
		for bundleID, matchingProfiles := range bundleIDProfilesMap {
			bundleIDs = append(bundleIDs, bundleID)
			profiles = append(profiles, matchingProfiles...)
		}

		//
		// create groups with single wildcard profiles
		{
			for _, profile := range profiles {
				if alreadyUsedProfileUUIDMap[profile.UUID] {
					continue
				}

				matchesForAllBundleID := true
				for _, bundleID := range bundleIDs {
					if !glob.Glob(profile.BundleID, bundleID) {
						matchesForAllBundleID = false
						break
					}
				}
				if matchesForAllBundleID {
					bundleIDProfileMap := map[string]profileutil.ProvisioningProfileInfoModel{}
					for _, bundleID := range bundleIDs {
						bundleIDProfileMap[bundleID] = profile
					}

					group := CodeSignGroup{
						Certificate:        certificate,
						BundleIDProfileMap: bundleIDProfileMap,
					}
					singleWildcardGroups = append(singleWildcardGroups, group)

					alreadyUsedProfileUUIDMap[profile.UUID] = true
				}
			}
		}

		//
		// create groups with xcode managed profiles
		{
			// collect xcode managed profiles
			xcodeManagedProfiles := []profileutil.ProvisioningProfileInfoModel{}
			for _, profile := range profiles {
				if !alreadyUsedProfileUUIDMap[profile.UUID] && profile.IsXcodeManaged() {
					xcodeManagedProfiles = append(xcodeManagedProfiles, profile)
				}
			}
			sort.Sort(ByBundleIDLength(xcodeManagedProfiles))

			// map profiles to bundle ids + remove the already used profiles
			bundleIDMannagedProfilesMap := map[string][]profileutil.ProvisioningProfileInfoModel{}
			for _, bundleID := range bundleIDs {
				for _, profile := range xcodeManagedProfiles {
					if !glob.Glob(profile.BundleID, bundleID) {
						continue
					}

					matchingProfiles := bundleIDMannagedProfilesMap[bundleID]
					if matchingProfiles == nil {
						matchingProfiles = []profileutil.ProvisioningProfileInfoModel{}
					}
					matchingProfiles = append(matchingProfiles, profile)
					bundleIDMannagedProfilesMap[bundleID] = matchingProfiles
				}
			}

			if len(bundleIDMannagedProfilesMap) == len(bundleIDs) {
				// if only one profile can sign a bundle id, remove it from bundleIDMannagedProfilesMap
				alreadyUsedManagedProfileMap := map[string]bool{}
				for _, profiles := range bundleIDMannagedProfilesMap {
					if len(profiles) == 1 {
						profile := profiles[0]
						alreadyUsedManagedProfileMap[profile.UUID] = true
					}
				}

				bundleIDMannagedProfileMap := map[string]profileutil.ProvisioningProfileInfoModel{}
				for bundleID, profiles := range bundleIDMannagedProfilesMap {
					if len(profiles) == 1 {
						bundleIDMannagedProfileMap[bundleID] = profiles[0]
					} else {
						remainingProfiles := []profileutil.ProvisioningProfileInfoModel{}
						for _, profile := range profiles {
							if !alreadyUsedManagedProfileMap[profile.UUID] {
								remainingProfiles = append(remainingProfiles, profile)
							}
						}
						if len(remainingProfiles) == 1 {
							bundleIDMannagedProfileMap[bundleID] = remainingProfiles[0]
						}
					}
				}

				// create code sign group
				if len(bundleIDMannagedProfileMap) == len(bundleIDs) {
					for _, profile := range bundleIDMannagedProfileMap {
						alreadyUsedProfileUUIDMap[profile.UUID] = true
					}

					group := CodeSignGroup{
						Certificate:        certificate,
						BundleIDProfileMap: bundleIDMannagedProfileMap,
					}
					xcodeManagedGroups = append(xcodeManagedGroups, group)
				}
			}
		}

		//
		// create groups with NOT xcode managed profiles
		{
			// collect xcode managed profiles
			notXcodeManagedProfiles := []profileutil.ProvisioningProfileInfoModel{}
			for _, profile := range profiles {
				if !alreadyUsedProfileUUIDMap[profile.UUID] && !profile.IsXcodeManaged() {
					notXcodeManagedProfiles = append(notXcodeManagedProfiles, profile)
				}
			}
			sort.Sort(ByBundleIDLength(notXcodeManagedProfiles))

			// map profiles to bundle ids + remove the already used profiles
			bundleIDNotMannagedProfilesMap := map[string][]profileutil.ProvisioningProfileInfoModel{}
			for _, bundleID := range bundleIDs {
				for _, profile := range notXcodeManagedProfiles {
					if !glob.Glob(profile.BundleID, bundleID) {
						continue
					}

					matchingProfiles := bundleIDNotMannagedProfilesMap[bundleID]
					if matchingProfiles == nil {
						matchingProfiles = []profileutil.ProvisioningProfileInfoModel{}
					}
					matchingProfiles = append(matchingProfiles, profile)
					bundleIDNotMannagedProfilesMap[bundleID] = matchingProfiles
				}
			}

			if len(bundleIDNotMannagedProfilesMap) == len(bundleIDs) {
				// if only one profile can sign a bundle id, remove it from bundleIDNotMannagedProfilesMap
				alreadyUsedNotManagedProfileMap := map[string]bool{}
				for _, profiles := range bundleIDNotMannagedProfilesMap {
					if len(profiles) == 1 {
						profile := profiles[0]
						alreadyUsedNotManagedProfileMap[profile.UUID] = true
					}
				}

				bundleIDNotMannagedProfileMap := map[string]profileutil.ProvisioningProfileInfoModel{}
				for bundleID, profiles := range bundleIDNotMannagedProfilesMap {
					if len(profiles) == 1 {
						bundleIDNotMannagedProfileMap[bundleID] = profiles[0]
					} else {
						remainingProfiles := []profileutil.ProvisioningProfileInfoModel{}
						for _, profile := range profiles {
							if !alreadyUsedNotManagedProfileMap[profile.UUID] {
								remainingProfiles = append(remainingProfiles, profile)
							}
						}
						if len(remainingProfiles) == 1 {
							bundleIDNotMannagedProfileMap[bundleID] = remainingProfiles[0]
						}
					}
				}

				// create code sign group
				if len(bundleIDNotMannagedProfileMap) == len(bundleIDs) {
					for _, profile := range bundleIDNotMannagedProfileMap {
						alreadyUsedProfileUUIDMap[profile.UUID] = true
					}

					codeSignGroup := CodeSignGroup{
						Certificate:        certificate,
						BundleIDProfileMap: bundleIDNotMannagedProfileMap,
					}
					notXcodeManagedGroups = append(notXcodeManagedGroups, codeSignGroup)
				}
			}
		}

		//
		// if there are remaining profiles we create a not exact group by using the first matching profile for every bundle id
		{
			if len(alreadyUsedProfileUUIDMap) != len(profiles) {
				bundleIDProfileMap := map[string]profileutil.ProvisioningProfileInfoModel{}
				for _, bundleID := range bundleIDs {
					for _, profile := range profiles {
						if alreadyUsedProfileUUIDMap[profile.UUID] {
							continue
						}

						if !glob.Glob(profile.BundleID, bundleID) {
							continue
						}

						bundleIDProfileMap[bundleID] = profile
						break
					}
				}

				if len(bundleIDProfileMap) == len(bundleIDs) {
					group := CodeSignGroup{
						Certificate:        certificate,
						BundleIDProfileMap: bundleIDProfileMap,
					}
					remainingGroups = append(remainingGroups, group)
				}
			}
		}

		fmt.Println()
	}

	codeSignGroups := []CodeSignGroup{}
	codeSignGroups = append(codeSignGroups, notXcodeManagedGroups...)
	codeSignGroups = append(codeSignGroups, xcodeManagedGroups...)
	codeSignGroups = append(codeSignGroups, singleWildcardGroups...)
	codeSignGroups = append(codeSignGroups, remainingGroups...)

	return codeSignGroups
}

// ResolveCodeSignGroups ...
func ResolveCodeSignGroups(certificates []certificateutil.CertificateInfoModel, profiles []profileutil.ProvisioningProfileInfoModel, bundleIDCapabilities map[string]plistutil.PlistData) []CodeSignGroup {
	selectableCodeSignGroups := ResolveSelectableCodeSignGroups(certificates, profiles, bundleIDCapabilities)
	return createCodeSignGroups(selectableCodeSignGroups)
}
