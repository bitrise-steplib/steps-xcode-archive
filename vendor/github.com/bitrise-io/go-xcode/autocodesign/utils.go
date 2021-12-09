package autocodesign

import (
	"fmt"
	"time"

	"github.com/bitrise-io/go-utils/log"
)

func mergeCodeSignAssets(base, additional *AppCodesignAssets) *AppCodesignAssets {
	if additional == nil {
		return base
	}

	if base == nil {
		return additional
	}

	if additional.ArchivableTargetProfilesByBundleID == nil {
		additional.ArchivableTargetProfilesByBundleID = base.ArchivableTargetProfilesByBundleID
	} else {
		for bundleID, profile := range base.ArchivableTargetProfilesByBundleID {
			additional.ArchivableTargetProfilesByBundleID[bundleID] = profile
		}
	}

	if additional.UITestTargetProfilesByBundleID == nil {
		additional.UITestTargetProfilesByBundleID = base.UITestTargetProfilesByBundleID
	} else {
		for bundleID, profile := range base.UITestTargetProfilesByBundleID {
			additional.UITestTargetProfilesByBundleID[bundleID] = profile
		}
	}

	base = additional

	return base
}

func printMissingCodeSignAssets(missingCodesignAssets *AppLayout) {
	fmt.Println()
	log.Infof("Local code signing assets not found for:")
	log.Printf("Archivable targets (%d)", len(missingCodesignAssets.EntitlementsByArchivableTargetBundleID))
	for bundleID := range missingCodesignAssets.EntitlementsByArchivableTargetBundleID {
		log.Printf("- %s", bundleID)
	}
	log.Printf("UITest targets (%d)", len(missingCodesignAssets.UITestTargetBundleIDs))
	for _, bundleID := range missingCodesignAssets.UITestTargetBundleIDs {
		log.Printf("- %s", bundleID)
	}
}

func printExistingCodesignAssets(assets *AppCodesignAssets, distrType DistributionType) {
	if assets == nil {
		return
	}

	fmt.Println()
	log.Infof("Local code signing assets for %s distribution:", distrType)
	log.Printf("Certificate: %s (team name: %s, serial: %s)", assets.Certificate.CommonName, assets.Certificate.TeamName, assets.Certificate.Serial)
	log.Printf("Archivable targets (%d)", len(assets.ArchivableTargetProfilesByBundleID))
	for bundleID, profile := range assets.ArchivableTargetProfilesByBundleID {
		log.Printf("- %s: %s (ID: %s UUID: %s Expiry: %s)", bundleID, profile.Attributes().Name, profile.ID(), profile.Attributes().UUID, time.Time(profile.Attributes().ExpirationDate))
	}

	log.Printf("UITest targets (%d)", len(assets.UITestTargetProfilesByBundleID))
	for bundleID, profile := range assets.UITestTargetProfilesByBundleID {
		log.Printf("- %s: %s (ID: %s UUID: %s Expiry: %s)", bundleID, profile.Attributes().Name, profile.ID(), profile.Attributes().UUID, time.Time(profile.Attributes().ExpirationDate))
	}
}
