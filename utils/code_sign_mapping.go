package utils

import (
	"sort"

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
		if certificate.RawSubject == installedCertificate.RawSubject && certificate.RawEndDate == installedCertificate.RawEndDate {
			installed = true
			break
		}
	}
	return installed
}

func createCertificateProfilesMapping(profiles []profileutil.ProfileInfoModel, certificates []certificateutil.CertificateInfoModel) map[string][]profileutil.ProfileInfoModel {
	createCertificateProfilesMap := map[string][]profileutil.ProfileInfoModel{}
	for _, profile := range profiles {
		for _, embeddedCert := range profile.DeveloperCertificates {
			if embeddedCert.RawSubject == "" {
				continue
			}
			if !isCertificateInstalled(certificates, embeddedCert) {
				continue
			}

			if _, ok := createCertificateProfilesMap[embeddedCert.RawSubject]; !ok {
				createCertificateProfilesMap[embeddedCert.RawSubject] = []profileutil.ProfileModel{}
			}
			createCertificateProfilesMap[embeddedCert.RawSubject] = append(createCertificateProfilesMap[embeddedCert.RawSubject], profile)
		}
	}

	return createCertificateProfilesMap
}

func createCodeSignGroups(profileGroups map[string][]profileutil.ProfileInfoModel, bundleIDs []string, exportMethod exportoptions.Method) []CodeSignGroupItem {
	filteredCodeSignGroupItems := []CodeSignGroupItem{}
	for groupItemCertificateSubject, bundleIDProfileMap := range profileGroups {
		sort.Sort(ByBundleIDLength(bundleIDProfileMap))

		bundleIDProfileMap := map[string]profileutil.ProfileModel{}
		for _, bundleID := range bundleIDs {
			for _, profile := range bundleIDProfileMap {
				if profile.ExportType != exportMethod {
					continue
				}

				if glob.Glob(profile.BundleIdentifier, bundleID) {
					bundleIDProfileMap[bundleID] = profile
					break
				}
			}
		}

		if len(bundleIDProfileMap) == len(bundleIDs) {
			groupItemCertificate := certificateutil.CertificateInfosModel{}
			for _, profile := range bundleIDProfileMap {
				for _, certificate := range profile.DeveloperCertificates {
					if groupItemCertificateSubject == certificate.RawSubject {
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
