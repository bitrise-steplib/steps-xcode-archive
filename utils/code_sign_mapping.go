package utils

import (
	"fmt"
	"sort"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-xcode/certificateutil"
	"github.com/bitrise-io/go-xcode/exportoptions"
	"github.com/bitrise-io/go-xcode/profileutil"
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

func createCertificateProfilesMapping(profiles []profileutil.ProvisioningProfileInfoModel, certificates []certificateutil.CertificateInfoModel) map[string][]profileutil.ProvisioningProfileInfoModel {
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

// ResolveCodeSignGroupItems ...
func ResolveCodeSignGroupItems(bundleIDs []string, exportMethod exportoptions.Method, profiles []profileutil.ProvisioningProfileInfoModel, certificates []certificateutil.CertificateInfoModel) []CodeSignGroupItem {
	log.Printf("Creating certificate profiles mapping...")
	certificateProfilesMapping := createCertificateProfilesMapping(profiles, certificates)

	log.Printf("Creating CodeSignGroups...")
	groups := createCodeSignGroups(certificateProfilesMapping, bundleIDs, exportMethod)

	return groups
}
