package export

import (
	"fmt"
	"sort"

	"github.com/bitrise-tools/go-xcode/plistutil"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-tools/go-xcode/certificateutil"
	"github.com/bitrise-tools/go-xcode/exportoptions"
	"github.com/bitrise-tools/go-xcode/profileutil"
	"github.com/ryanuber/go-glob"
)

// CodeSignGroupItem ...
type CodeSignGroupItem struct {
	Certificate        certificateutil.CertificateInfoModel
	BundleIDProfileMap map[string]profileutil.ProvisioningProfileInfoModel
}

func isCertificateInstalled(installedCertificates []certificateutil.CertificateInfoModel, certificate certificateutil.CertificateInfoModel) bool {
	installed := false
	for _, installedCertificate := range installedCertificates {
		if certificate.Serial == installedCertificate.Serial {
			installed = true
			break
		}
	}

	if installed {
		log.Printf("certificate: %s [%s] is installed", certificate.CommonName, certificate.Serial)
	}

	return installed
}

func createCertificateProfilesMapping(certificates []certificateutil.CertificateInfoModel, profiles []profileutil.ProvisioningProfileInfoModel) map[string][]profileutil.ProvisioningProfileInfoModel {
	createCertificateProfilesMap := map[string][]profileutil.ProvisioningProfileInfoModel{}
	for _, profile := range profiles {
		for _, embeddedCert := range profile.DeveloperCertificates {
			if !isCertificateInstalled(certificates, embeddedCert) {
				continue
			}

			if _, ok := createCertificateProfilesMap[embeddedCert.Serial]; !ok {
				createCertificateProfilesMap[embeddedCert.Serial] = []profileutil.ProvisioningProfileInfoModel{}
			}
			createCertificateProfilesMap[embeddedCert.Serial] = append(createCertificateProfilesMap[embeddedCert.Serial], profile)
		}
	}

	fmt.Println()

	for subject, profiles := range createCertificateProfilesMap {
		log.Printf("certificate: %s included in profiles:", subject)
		for _, profile := range profiles {
			log.Printf("- %s", profile.Name)
		}
		fmt.Println()
	}

	return createCertificateProfilesMap
}

func printCertificateProfilesGroup(serial string, profiles []profileutil.ProvisioningProfileInfoModel) {
	log.Printf("%s:", serial)
	for _, profile := range profiles {
		log.Printf("- %s", profile.Name)
	}
}

func filterCertificateProfilesMapping(mapping map[string][]profileutil.ProvisioningProfileInfoModel, bundleIDCapabilitiesMap map[string]plistutil.PlistData, exportMethod exportoptions.Method) map[string][]profileutil.ProvisioningProfileInfoModel {
	createCertificateProfilesMap := map[string][]profileutil.ProvisioningProfileInfoModel{}

	for serial, profiles := range mapping {
		log.Printf("Checking certificate - profiles group: %s", serial)

		bundleIDProfilesMap := map[string][]profileutil.ProvisioningProfileInfoModel{}
		for bundleID, capabilities := range bundleIDCapabilitiesMap {
			for _, profile := range profiles {
				if profile.ExportType != exportMethod {
					log.Printf("Profile (%s) export type (%s) does not match: %s", profile.Name, profile.ExportType, exportMethod)
					continue
				}

				if !glob.Glob(profile.BundleID, bundleID) {
					log.Printf("Profile (%s) bundle id (%s) does not match: %s", profile.Name, profile.BundleID, bundleID)
					continue
				}

				if missingCapabilities := profileutil.MatchTargetAndProfileEntitlements(capabilities, profile.Entitlements); len(missingCapabilities) > 0 {
					log.Printf("Profile (%s) does not have capabilities: %v", profile.Name, missingCapabilities)
					continue
				}

				log.Printf("Profile (%s) matches", profile.Name)

				matchingProfiles := bundleIDProfilesMap[bundleID]
				if matchingProfiles == nil {
					matchingProfiles = []profileutil.ProvisioningProfileInfoModel{}
				}
				matchingProfiles = append(matchingProfiles, profile)
				bundleIDProfilesMap[bundleID] = matchingProfiles
			}
		}
		if len(bundleIDProfilesMap) == len(bundleIDCapabilitiesMap) {
			matchingProfiles := []profileutil.ProvisioningProfileInfoModel{}
			for _, profiles := range bundleIDProfilesMap {
				matchingProfiles = append(matchingProfiles, profiles...)
			}
			createCertificateProfilesMap[serial] = matchingProfiles

			log.Printf("Valid certificate - profiles group:")
			printCertificateProfilesGroup(serial, matchingProfiles)
		} else {
			log.Printf("Removing certificate - profiles group: %s", serial)
		}

		fmt.Println()
	}

	fmt.Println()
	return createCertificateProfilesMap
}

func createCodeSignGroups(profileGroups map[string][]profileutil.ProvisioningProfileInfoModel, bundleIDs []string, exportMethod exportoptions.Method) []CodeSignGroupItem {
	filteredCodeSignGroupItems := []CodeSignGroupItem{}
	for groupItemCertificateSerial, profiles := range profileGroups {
		log.Printf("checking certificate (%s) group:", groupItemCertificateSerial)
		sort.Sort(ByBundleIDLength(profiles))

		bundleIDProfileMap := map[string]profileutil.ProvisioningProfileInfoModel{}
		for _, bundleID := range bundleIDs {
			for _, profile := range profiles {
				if profile.ExportType != exportMethod {
					log.Printf("profile (%s) export method (%s) is not the desired (%s)", profile.Name, profile.ExportType, exportMethod)
					continue
				}

				if !glob.Glob(profile.BundleID, bundleID) {
					log.Printf("profile (%s) does not provision bundle id: %s", profile.Name, profile.BundleID)
					continue
				}

				log.Printf("profile (%s) MATCHES for bundle id (%s) and export method (%s)", profile.Name, bundleID, exportMethod)

				bundleIDProfileMap[bundleID] = profile
				break
			}
		}

		log.Printf("matching profiles: %d should be: %d", len(bundleIDProfileMap), len(bundleIDs))
		fmt.Println()

		if len(bundleIDProfileMap) == len(bundleIDs) {
			groupItemCertificate := certificateutil.CertificateInfoModel{}
			for _, profile := range bundleIDProfileMap {
				for _, certificate := range profile.DeveloperCertificates {
					if groupItemCertificateSerial == certificate.Serial {
						groupItemCertificate = certificate
					}
				}
			}

			filteredCodeSignGroupItems = append(filteredCodeSignGroupItems, CodeSignGroupItem{Certificate: groupItemCertificate, BundleIDProfileMap: bundleIDProfileMap})
		}
	}
	return filteredCodeSignGroupItems
}

func findCertificate(certificates []certificateutil.CertificateInfoModel, serial string) *certificateutil.CertificateInfoModel {
	for _, certificate := range certificates {
		if certificate.Serial == serial {
			return &certificate
		}
	}
	return nil
}

func printGroup(group CodeSignGroupItem) {
	log.Printf("Signing with certificate: %s", group.Certificate.CommonName)
	for bundleID, profile := range group.BundleIDProfileMap {
		log.Printf("signing %s with: %s", bundleID, profile.Name)
	}
}

func createGroups(mapping map[string][]profileutil.ProvisioningProfileInfoModel, bundleIDs []string) []CodeSignGroupItem {
	alreadyUsedProfileUUIDMap := map[string]bool{}

	singleWildcardGroups := []CodeSignGroupItem{}
	xcodeManagedGroups := []CodeSignGroupItem{}
	notXcodeManagedGroups := []CodeSignGroupItem{}
	remainingGroups := []CodeSignGroupItem{}

	for serial, profiles := range mapping {
		log.Printf("Checking certificate - profiles group: %s", serial)

		//
		// create groups with single wildcard profiles
		{
			log.Printf("Checking for group with single wildcard profile")
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
					certificate := findCertificate(profile.DeveloperCertificates, serial)
					if certificate != nil {
						bundleIDProfileMap := map[string]profileutil.ProvisioningProfileInfoModel{}
						for _, bundleID := range bundleIDs {
							bundleIDProfileMap[bundleID] = profile
						}
						codeSignGroup := CodeSignGroupItem{
							Certificate:        *certificate,
							BundleIDProfileMap: bundleIDProfileMap,
						}
						singleWildcardGroups = append(singleWildcardGroups, codeSignGroup)
						alreadyUsedProfileUUIDMap[profile.UUID] = true
						log.Printf("Group with single wildcard profile found:")
						printGroup(codeSignGroup)
					}
				}
			}
		}

		//
		// create groups with xcode managed profiles
		{
			log.Printf("Checking for group with xcode managed profiles")

			// collect xcode managed profiles
			xcodeManagedProfiles := []profileutil.ProvisioningProfileInfoModel{}
			for _, profile := range profiles {
				if alreadyUsedProfileUUIDMap[profile.UUID] {
					continue
				}

				if profile.IsXcodeManaged() {
					xcodeManagedProfiles = append(xcodeManagedProfiles, profile)
				}
			}

			// map profiles to bundle ids
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
				// if only one profile can sign a bundle id, remove it from other bundle id - profiles map
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
					lastProfile := profileutil.ProvisioningProfileInfoModel{}
					for _, profile := range bundleIDMannagedProfileMap {
						lastProfile = profile
						alreadyUsedProfileUUIDMap[profile.UUID] = true
					}

					certificate := findCertificate(lastProfile.DeveloperCertificates, serial)
					if certificate != nil {
						codeSignGroup := CodeSignGroupItem{
							Certificate:        *certificate,
							BundleIDProfileMap: bundleIDMannagedProfileMap,
						}
						xcodeManagedGroups = append(xcodeManagedGroups, codeSignGroup)
						log.Printf("Group with xcode managed profiles found:")
						printGroup(codeSignGroup)
					}
				}
			}
		}

		//
		// create groups with NOT xcode managed profiles
		{
			log.Printf("Checking for group with NOT xcode managed profiles")

			// collect xcode managed profiles
			notXcodeManagedProfiles := []profileutil.ProvisioningProfileInfoModel{}
			for _, profile := range profiles {
				if alreadyUsedProfileUUIDMap[profile.UUID] {
					continue
				}

				if !profile.IsXcodeManaged() {
					notXcodeManagedProfiles = append(notXcodeManagedProfiles, profile)
				}
			}

			// map profiles to bundle ids
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
				// if only one profile can sign a bundle id, remove it from other bundle id - profiles map
				alreadyUsedManagedProfileMap := map[string]bool{}
				for _, profiles := range bundleIDNotMannagedProfilesMap {
					if len(profiles) == 1 {
						profile := profiles[0]
						alreadyUsedManagedProfileMap[profile.UUID] = true
					}
				}

				bundleIDMannagedProfileMap := map[string]profileutil.ProvisioningProfileInfoModel{}
				for bundleID, profiles := range bundleIDNotMannagedProfilesMap {
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
					lastProfile := profileutil.ProvisioningProfileInfoModel{}
					for _, profile := range bundleIDMannagedProfileMap {
						lastProfile = profile
						alreadyUsedProfileUUIDMap[profile.UUID] = true
					}

					certificate := findCertificate(lastProfile.DeveloperCertificates, serial)
					if certificate != nil {
						codeSignGroup := CodeSignGroupItem{
							Certificate:        *certificate,
							BundleIDProfileMap: bundleIDMannagedProfileMap,
						}
						notXcodeManagedGroups = append(notXcodeManagedGroups, codeSignGroup)
						log.Printf("Group with NOT xcode managed profiles found:")
						printGroup(codeSignGroup)
					}
				}
			}
		}

		//
		// if there are remaining profiles we create a not exact group by using the first matching profile for every bundle id
		{
			if len(alreadyUsedProfileUUIDMap) != len(profiles) {
				log.Printf("There are remaining profile create group by using the first matching profile for every bundle id")

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
					firstProfile := profileutil.ProvisioningProfileInfoModel{}
					for _, profile := range bundleIDProfileMap {
						firstProfile = profile
						break
					}

					certificate := findCertificate(firstProfile.DeveloperCertificates, serial)
					if certificate != nil {
						codeSignGroup := CodeSignGroupItem{
							Certificate:        *certificate,
							BundleIDProfileMap: bundleIDProfileMap,
						}
						remainingGroups = append(remainingGroups, codeSignGroup)
						log.Printf("Group with first matching profiles:")
						printGroup(codeSignGroup)
					}
				}
			}
		}

		fmt.Println()
	}

	codeSignGroups := []CodeSignGroupItem{}
	codeSignGroups = append(codeSignGroups, notXcodeManagedGroups...)
	codeSignGroups = append(codeSignGroups, xcodeManagedGroups...)
	codeSignGroups = append(codeSignGroups, singleWildcardGroups...)
	codeSignGroups = append(codeSignGroups, remainingGroups...)

	return codeSignGroups
}

// Resolve ...
func Resolve(certificates []certificateutil.CertificateInfoModel, profiles []profileutil.ProvisioningProfileInfoModel, bundleIDCapabilities map[string]plistutil.PlistData, exportMethod exportoptions.Method) []CodeSignGroupItem {
	log.Printf("Creating certificate profiles mapping...")
	certificateProfilesMapping := createCertificateProfilesMapping(certificates, profiles)

	log.Printf("Filtering certificate profiles mapping...")
	certificateProfilesMapping = filterCertificateProfilesMapping(certificateProfilesMapping, bundleIDCapabilities, exportMethod)

	bundleIDs := []string{}
	for bundleID := range bundleIDCapabilities {
		bundleIDs = append(bundleIDs, bundleID)
	}

	log.Printf("Creating code sign groups")
	return createGroups(certificateProfilesMapping, bundleIDs)
}

// ResolveCodeSignGroupItems ...
func ResolveCodeSignGroupItems(bundleIDs []string, exportMethod exportoptions.Method, profiles []profileutil.ProvisioningProfileInfoModel, certificates []certificateutil.CertificateInfoModel) []CodeSignGroupItem {
	log.Printf("Creating certificate profiles mapping...")
	certificateProfilesMapping := createCertificateProfilesMapping(certificates, profiles)

	log.Printf("Creating CodeSignGroups...")
	groups := createCodeSignGroups(certificateProfilesMapping, bundleIDs, exportMethod)

	return groups
}
