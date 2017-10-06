package utils

import (
	"sort"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/steps-certificate-and-profile-installer/certificateutil"
	"github.com/bitrise-io/steps-certificate-and-profile-installer/profileutil"
	"github.com/bitrise-tools/go-xcode/exportoptions"
	"github.com/ryanuber/go-glob"
)

// CodeSignGroupItem ...
type CodeSignGroupItem struct {
	Certificate        certificateutil.CertificateInfoModel
	BundleIDProfileMap map[string]profileutil.ProfileInfoModel
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
		log.Printf("certificate: %s installed", certificate.CommonName)
	}

	return installed
}

func createCertificateProfilesMapping(profiles []profileutil.ProfileInfoModel, certificates []certificateutil.CertificateInfoModel) map[string][]profileutil.ProfileInfoModel {
	createCertificateProfilesMap := map[string][]profileutil.ProfileInfoModel{}
	for _, profile := range profiles {
		for _, embeddedCert := range profile.DeveloperCertificates {
			if !isCertificateInstalled(certificates, embeddedCert) {
				continue
			}

			if _, ok := createCertificateProfilesMap[embeddedCert.Serial]; !ok {
				createCertificateProfilesMap[embeddedCert.Serial] = []profileutil.ProfileInfoModel{}
			}
			createCertificateProfilesMap[embeddedCert.Serial] = append(createCertificateProfilesMap[embeddedCert.Serial], profile)
		}
	}

	for subject, profiles := range createCertificateProfilesMap {
		log.Printf("certificate: %s profiles:", subject)
		for _, profile := range profiles {
			log.Printf("- %s", profile.Name)
		}
	}

	return createCertificateProfilesMap
}

func createCodeSignGroups(profileGroups map[string][]profileutil.ProfileInfoModel, bundleIDs []string, exportMethod exportoptions.Method) []CodeSignGroupItem {
	filteredCodeSignGroupItems := []CodeSignGroupItem{}
	for groupItemCertificateSerial, profiles := range profileGroups {
		sort.Sort(ByBundleIDLength(profiles))

		bundleIDProfileMap := map[string]profileutil.ProfileInfoModel{}
		for _, bundleID := range bundleIDs {
			for _, profile := range profiles {
				if profile.ExportType != exportMethod {
					log.Printf("profile: %s is not for export method: %s", profile.Name, exportMethod)
					continue
				}

				if !glob.Glob(profile.BundleIdentifier, bundleID) {
					log.Printf("profile: %s is not for bundle id: %s", profile.Name, profile.BundleIdentifier)
					continue
				}

				log.Printf("profile: %s MATCHES for: %s", profile.Name, bundleID)

				bundleIDProfileMap[bundleID] = profile
				break
			}
		}

		log.Printf("len(bundleIDProfileMap): %d <-> len(bundleIDs): %d", len(bundleIDProfileMap), len(bundleIDs))

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

// ResolveCodeSignGroupItems ...
func ResolveCodeSignGroupItems(bundleIDs []string, exportMethod exportoptions.Method, profiles []profileutil.ProfileInfoModel, certificates []certificateutil.CertificateInfoModel) []CodeSignGroupItem {
	certificateProfilesMapping := createCertificateProfilesMapping(profiles, certificates)
	return createCodeSignGroups(certificateProfilesMapping, bundleIDs, exportMethod)
}
