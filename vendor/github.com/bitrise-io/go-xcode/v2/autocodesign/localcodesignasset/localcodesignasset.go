package localcodesignasset

import (
	"fmt"

	"github.com/bitrise-io/go-xcode/v2/autocodesign"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/devportalclient/appstoreconnect"
)

// Manager ...
type Manager struct {
	profileProvider  ProvisioningProfileProvider
	profileConverter ProvisioningProfileConverter
}

// NewManager ...
func NewManager(provisioningProfileProvider ProvisioningProfileProvider, provisioningProfileConverter ProvisioningProfileConverter) Manager {
	return Manager{
		profileProvider:  provisioningProfileProvider,
		profileConverter: provisioningProfileConverter,
	}
}

// FindCodesignAssets ...
func (m Manager) FindCodesignAssets(appLayout autocodesign.AppLayout, distrType autocodesign.DistributionType, certsByType map[appstoreconnect.CertificateType][]autocodesign.Certificate, deviceIDs []string, minProfileDaysValid int) (*autocodesign.AppCodesignAssets, *autocodesign.AppLayout, error) {
	profiles, err := m.profileProvider.ListProvisioningProfiles()
	if err != nil {
		return nil, nil, err
	}

	certSerials := certificateSerials(certsByType, distrType)

	var asset *autocodesign.AppCodesignAssets
	for bundleID, entitlements := range appLayout.EntitlementsByArchivableTargetBundleID {
		profileInfo := findProfile(profiles, appLayout.Platform, distrType, bundleID, entitlements, minProfileDaysValid, certSerials, deviceIDs)
		if profileInfo == nil {
			continue
		}

		profile, err := m.profileConverter.ProfileInfoToProfile(*profileInfo)
		if err != nil {
			return nil, nil, err
		}

		if asset == nil {
			asset = &autocodesign.AppCodesignAssets{
				ArchivableTargetProfilesByBundleID: map[string]autocodesign.Profile{
					bundleID: profile,
				},
			}
		} else {
			profileByArchivableTargetBundleID := asset.ArchivableTargetProfilesByBundleID
			if profileByArchivableTargetBundleID == nil {
				profileByArchivableTargetBundleID = map[string]autocodesign.Profile{}
			}

			profileByArchivableTargetBundleID[bundleID] = profile
			asset.ArchivableTargetProfilesByBundleID = profileByArchivableTargetBundleID
		}

		delete(appLayout.EntitlementsByArchivableTargetBundleID, bundleID)
	}

	if distrType == autocodesign.Development {
		bundleIDs := map[string]bool{}
		for _, bundleID := range appLayout.UITestTargetBundleIDs {
			bundleIDs[bundleID] = true // profile missing?
		}

		for bundleID := range bundleIDs {
			wildcardBundleID, err := autocodesign.CreateWildcardBundleID(bundleID)
			if err != nil {
				return nil, nil, fmt.Errorf("could not create wildcard bundle id: %s", err)
			}

			// Capabilities are not supported for UITest targets.
			profileInfo := findProfile(profiles, appLayout.Platform, distrType, wildcardBundleID, nil, minProfileDaysValid, certSerials, deviceIDs)
			if profileInfo == nil {
				continue
			}

			profile, err := m.profileConverter.ProfileInfoToProfile(*profileInfo)
			if err != nil {
				return nil, nil, err
			}

			if asset == nil {
				asset = &autocodesign.AppCodesignAssets{
					UITestTargetProfilesByBundleID: map[string]autocodesign.Profile{
						bundleID: profile,
					},
				}
			} else {
				profileByUITestTargetBundleID := asset.UITestTargetProfilesByBundleID
				if profileByUITestTargetBundleID == nil {
					profileByUITestTargetBundleID = map[string]autocodesign.Profile{}
				}

				profileByUITestTargetBundleID[bundleID] = profile
				asset.UITestTargetProfilesByBundleID = profileByUITestTargetBundleID
			}

			bundleIDs[bundleID] = false
		}

		var uiTestTargetBundleIDs []string
		for bundleID, missing := range bundleIDs {
			if missing {
				uiTestTargetBundleIDs = append(uiTestTargetBundleIDs, bundleID)
			}
		}

		appLayout.UITestTargetBundleIDs = uiTestTargetBundleIDs
	}

	if asset != nil {
		// We will always have a certificate at this point because if we do not have any then we also could not have
		// found a profile as all of them requires at least one certificate.
		certificate, err := autocodesign.SelectCertificate(certsByType, distrType)
		if err != nil {
			return nil, nil, err
		}

		asset.Certificate = certificate.CertificateInfo
	}

	if len(appLayout.EntitlementsByArchivableTargetBundleID) == 0 && len(appLayout.UITestTargetBundleIDs) == 0 {
		return asset, nil, nil
	}

	return asset, &appLayout, nil
}
