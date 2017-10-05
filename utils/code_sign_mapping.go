package utils

import (
	"sort"

	"github.com/bitrise-tools/go-xcode/exportoptions"
	"github.com/davecgh/go-spew/spew"
	glob "github.com/ryanuber/go-glob"

	"github.com/bitrise-io/steps-certificate-and-profile-installer/certificateutil"
	"github.com/bitrise-io/steps-certificate-and-profile-installer/profileutil"
)

// CodeSignGroupItem ...
type CodeSignGroupItem struct {
	Certificate        certificateutil.CertificateInfosModel
	BundleIDProfileMap map[string]profileutil.ProfileModel
}

// ResolveCodeSignGroupItems ...
func ResolveCodeSignGroupItems(bundleIDs []string, exportMethod exportoptions.Method, profiles []profileutil.ProfileModel, certificates []certificateutil.CertificateInfosModel) []CodeSignGroupItem {
	certificateProfilesMapping := createCertificateProfilesMapping(profiles, certificates, exportMethod)
	spew.Dump(certificateProfilesMapping)
	return createCodeSignGroupItem(certificateProfilesMapping, bundleIDs)
}

func isCertificateInstalled(installedCertificates []certificateutil.CertificateInfosModel, certificate certificateutil.CertificateInfosModel) bool {
	installed := false
	for _, installedCertificate := range installedCertificates {
		if certificate.RawSubject == installedCertificate.RawSubject && certificate.RawEndDate == installedCertificate.RawEndDate {
			installed = true
			break
		}
	}
	return installed
}

func createCertificateProfilesMapping(profiles []profileutil.ProfileModel, certificates []certificateutil.CertificateInfosModel, exportMethod exportoptions.Method) map[string][]profileutil.ProfileModel {
	createCertificateProfilesMap := map[string][]profileutil.ProfileModel{}
	for _, profile := range profiles {
		if profile.ExportType != exportMethod {
			continue
		}
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

func createCodeSignGroupItem(profileGroups map[string][]profileutil.ProfileModel, bundleIDs []string) []CodeSignGroupItem {
	filteredCodeSignGroupItems := []CodeSignGroupItem{}
	for groupItemCertificateSubject, bundleIDProfileMap := range profileGroups {
		sort.Sort(ByBundleIDLength(bundleIDProfileMap))

		bundleIDProfileMap := map[string]profileutil.ProfileModel{}
		for _, bundleID := range bundleIDs {
			for _, profile := range bundleIDProfileMap {
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
