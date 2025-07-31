// Package projectmanager parses and edits an Xcode project.
//
// Use cases:
//  1. Get codesigning related information, needed to fetch or recreate certificates and provisioning profiles
//  2. Apply codesigning settings in the projects
package projectmanager

import (
	"fmt"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-xcode/v2/autocodesign"
)

// Project ...
type Project struct {
	projHelper ProjectHelper
}

// Factory ...
type Factory struct {
	params InitParams
}

// InitParams ...
type InitParams struct {
	ProjectOrWorkspacePath string
	SchemeName             string
	ConfigurationName      string
}

// NewFactory ...
func NewFactory(params InitParams) Factory {
	return Factory{params: params}
}

// Create ...
func (f *Factory) Create() (Project, error) {
	return NewProject(f.params)
}

// NewProject ...
func NewProject(params InitParams) (Project, error) {
	projectHelper, err := NewProjectHelper(params.ProjectOrWorkspacePath, params.SchemeName, params.ConfigurationName)
	if err != nil {
		return Project{}, err
	}

	return Project{
		projHelper: *projectHelper,
	}, nil
}

// IsSigningManagedAutomatically checks the "Automatically manage signing" checkbox in Xcode
// Note: it only checks the main Target based on the given Scheme and Configuration
func (p Project) IsSigningManagedAutomatically() (bool, error) {
	return p.projHelper.IsSigningManagedAutomatically()
}

// Platform get the platform (PLATFORM_DISPLAY_NAME) - iOS, tvOS, macOS
func (p Project) Platform() (autocodesign.Platform, error) {
	platform, err := p.projHelper.Platform(p.projHelper.Configuration)
	if err != nil {
		return "", fmt.Errorf("failed to read project platform: %s", err)
	}

	log.Debugf("Platform: %s", platform)

	return platform, nil
}

// MainTargetBundleID ...
func (p Project) MainTargetBundleID() (string, error) {
	bundleID, err := p.projHelper.TargetBundleID(p.projHelper.MainTarget.Name, p.projHelper.Configuration)
	if err != nil {
		return "", fmt.Errorf("failed to read bundle ID for the main target: %s", err)
	}

	return bundleID, nil
}

// GetAppLayout ...
func (p Project) GetAppLayout(uiTestTargets bool) (autocodesign.AppLayout, error) {
	log.Printf("Configuration: %s", p.projHelper.Configuration)

	platform, err := p.projHelper.Platform(p.projHelper.Configuration)
	if err != nil {
		return autocodesign.AppLayout{}, fmt.Errorf("failed to read project platform: %s", err)
	}

	log.Debugf("Platform: %s", platform)

	log.Printf("Application and App Extension targets:")
	for _, target := range p.projHelper.ArchivableTargets() {
		log.Printf("- %s", target.Name)
	}

	archivableTargetBundleIDToEntitlements, err := p.projHelper.ArchivableTargetBundleIDToEntitlements()
	if err != nil {
		return autocodesign.AppLayout{}, fmt.Errorf("failed to read archivable targets' entitlements: %s", err)
	}

	if ok, entitlement, bundleID := CanGenerateProfileWithEntitlements(archivableTargetBundleIDToEntitlements); !ok {
		log.Errorf("Can not create profile with unsupported entitlement (%s) for the bundle ID %s, due to App Store Connect API limitations.", entitlement, bundleID)
		return autocodesign.AppLayout{}, fmt.Errorf("please generate provisioning profile manually on Apple Developer Portal and use the Certificate and profile installer Step instead")
	}

	var uiTestTargetBundleIDs []string
	if uiTestTargets {
		log.Printf("UITest targets:")
		for _, target := range p.projHelper.UITestTargets {
			log.Printf("- %s", target.Name)
		}

		uiTestTargetBundleIDs, err = p.projHelper.UITestTargetBundleIDs()
		if err != nil {
			return autocodesign.AppLayout{}, fmt.Errorf("failed to read UITest targets' entitlements: %s", err)
		}
	}

	return autocodesign.AppLayout{
		Platform:                               platform,
		EntitlementsByArchivableTargetBundleID: archivableTargetBundleIDToEntitlements,
		UITestTargetBundleIDs:                  uiTestTargetBundleIDs,
	}, nil
}

// ForceCodesignAssets ...
func (p Project) ForceCodesignAssets(distribution autocodesign.DistributionType, codesignAssetsByDistributionType map[autocodesign.DistributionType]autocodesign.AppCodesignAssets) error {
	log.Debugf("🔍 [FORCE CODESIGN] Starting ForceCodesignAssets with distribution: %s", distribution)
	log.Debugf("🔍 [FORCE CODESIGN] Available distribution types in codesignAssetsByDistributionType:")
	for distType := range codesignAssetsByDistributionType {
		log.Debugf("🔍 [FORCE CODESIGN]   - %s", distType)
	}

	archivableTargets := p.projHelper.ArchivableTargets()
	var archivableTargetsCounter = 0

	log.Debugf("🔍 [FORCE CODESIGN] Found %d archivable targets:", len(archivableTargets))
	for i, target := range archivableTargets {
		log.Debugf("🔍 [FORCE CODESIGN]   %d. %s", i+1, target.Name)
	}

	fmt.Println()
	log.TInfof("Apply Bitrise managed codesigning on the executable targets (up to: %d targets)", len(archivableTargets))

	for _, target := range archivableTargets {
		fmt.Println()
		log.Infof("  Target: %s", target.Name)
		log.Debugf("🔍 [FORCE CODESIGN] Processing target: %s", target.Name)

		forceCodesignDistribution := distribution
		log.Debugf("🔍 [FORCE CODESIGN] Initial distribution for target %s: %s", target.Name, forceCodesignDistribution)

		if _, isDevelopmentAvailable := codesignAssetsByDistributionType[autocodesign.Development]; isDevelopmentAvailable {
			forceCodesignDistribution = autocodesign.Development
			log.Debugf("🔍 [FORCE CODESIGN] Development assets available, switching to Development distribution for target %s", target.Name)
		} else {
			log.Debugf("🔍 [FORCE CODESIGN] No Development assets available for target %s", target.Name)
		}

		log.Debugf("🔍 [FORCE CODESIGN] Final distribution for target %s: %s", target.Name, forceCodesignDistribution)

		codesignAssets, ok := codesignAssetsByDistributionType[forceCodesignDistribution]
		if !ok {
			log.Debugf("❌ [FORCE CODESIGN] No codesign settings found for distribution type %s for target %s", forceCodesignDistribution, target.Name)
			return fmt.Errorf("no codesign settings ensured for distribution type %s", forceCodesignDistribution)
		}
		log.Debugf("🔍 [FORCE CODESIGN] Found codesign assets for distribution %s for target %s", forceCodesignDistribution, target.Name)

		teamID := codesignAssets.Certificate.TeamID
		log.Debugf("🔍 [FORCE CODESIGN] Team ID for target %s: %s", target.Name, teamID)

		targetBundleID, err := p.projHelper.TargetBundleID(target.Name, p.projHelper.Configuration)
		if err != nil {
			log.Debugf("❌ [FORCE CODESIGN] Failed to get bundle ID for target %s: %v", target.Name, err)
			return err
		}
		log.Debugf("🔍 [FORCE CODESIGN] Bundle ID for target %s: %s", target.Name, targetBundleID)

		log.Debugf("🔍 [FORCE CODESIGN] Available profiles in ArchivableTargetProfilesByBundleID for target %s:", target.Name)
		for bundleID, profile := range codesignAssets.ArchivableTargetProfilesByBundleID {
			log.Debugf("🔍 [FORCE CODESIGN]   Bundle ID: %s -> Profile: %s", bundleID, profile.Attributes().Name)
		}

		profile, ok := codesignAssets.ArchivableTargetProfilesByBundleID[targetBundleID]
		if !ok {
			log.Debugf("❌ [FORCE CODESIGN] No profile found for bundle ID %s for target %s", targetBundleID, target.Name)
			log.Debugf("❌ [FORCE CODESIGN] Available bundle IDs in profiles:")
			for availableBundleID := range codesignAssets.ArchivableTargetProfilesByBundleID {
				log.Debugf("❌ [FORCE CODESIGN]   - %s", availableBundleID)
			}
			return fmt.Errorf("no profile ensured for the bundleID %s", targetBundleID)
		}
		log.Debugf("✅ [FORCE CODESIGN] Found profile for target %s (bundle ID %s): %s", target.Name, targetBundleID, profile.Attributes().Name)

		log.Printf("  development Team: %s(%s)", codesignAssets.Certificate.TeamName, teamID)
		log.Printf("  provisioning Profile: %s", profile.Attributes().Name)
		log.Printf("  certificate: %s", codesignAssets.Certificate.CommonName)

		log.Debugf("🔍 [FORCE CODESIGN] About to apply code sign settings for target %s:", target.Name)
		log.Debugf("🔍 [FORCE CODESIGN]   Configuration: %s", p.projHelper.Configuration)
		log.Debugf("🔍 [FORCE CODESIGN]   Target: %s", target.Name)
		log.Debugf("🔍 [FORCE CODESIGN]   Team ID: %s", teamID)
		log.Debugf("🔍 [FORCE CODESIGN]   Certificate SHA1: %s", codesignAssets.Certificate.SHA1Fingerprint)
		log.Debugf("🔍 [FORCE CODESIGN]   Profile UUID: %s", profile.Attributes().UUID)

		if err := p.projHelper.XcProj.ForceCodeSign(p.projHelper.Configuration, target.Name, teamID, codesignAssets.Certificate.SHA1Fingerprint, profile.Attributes().UUID); err != nil {
			log.Debugf("❌ [FORCE CODESIGN] Failed to apply code sign settings for target %s: %v", target.Name, err)
			return fmt.Errorf("failed to apply code sign settings for target (%s): %s", target.Name, err)
		}
		log.Debugf("✅ [FORCE CODESIGN] Successfully applied code sign settings for target %s", target.Name)

		archivableTargetsCounter++
	}

	log.TInfof("Applied Bitrise managed codesigning on up to %s targets", archivableTargetsCounter)
	log.Debugf("🔍 [FORCE CODESIGN] Completed processing %d archivable targets", archivableTargetsCounter)

	devCodesignAssets, isDevelopmentAvailable := codesignAssetsByDistributionType[autocodesign.Development]
	log.Debugf("🔍 [FORCE CODESIGN] Checking for UITest targets. Development available: %t", isDevelopmentAvailable)

	if isDevelopmentAvailable {
		log.Debugf("🔍 [FORCE CODESIGN] UITest target profiles count: %d", len(devCodesignAssets.UITestTargetProfilesByBundleID))
		log.Debugf("🔍 [FORCE CODESIGN] UITest targets count: %d", len(p.projHelper.UITestTargets))
	}

	if isDevelopmentAvailable && len(devCodesignAssets.UITestTargetProfilesByBundleID) != 0 {
		fmt.Println()
		log.TInfof("Apply Bitrise managed codesigning on the UITest targets (%d)", len(p.projHelper.UITestTargets))
		log.Debugf("🔍 [FORCE CODESIGN] Starting UITest targets processing")

		for _, uiTestTarget := range p.projHelper.UITestTargets {
			fmt.Println()
			log.Infof("  Target: %s", uiTestTarget.Name)
			log.Debugf("🔍 [FORCE CODESIGN] Processing UITest target: %s", uiTestTarget.Name)

			teamID := devCodesignAssets.Certificate.TeamID
			log.Debugf("🔍 [FORCE CODESIGN] Team ID for UITest target %s: %s", uiTestTarget.Name, teamID)

			targetBundleID, err := p.projHelper.TargetBundleID(uiTestTarget.Name, p.projHelper.Configuration)
			if err != nil {
				log.Debugf("❌ [FORCE CODESIGN] Failed to get bundle ID for UITest target %s: %v", uiTestTarget.Name, err)
				return err
			}
			log.Debugf("🔍 [FORCE CODESIGN] Bundle ID for UITest target %s: %s", uiTestTarget.Name, targetBundleID)

			log.Debugf("🔍 [FORCE CODESIGN] Available UITest profiles:")
			for bundleID, profile := range devCodesignAssets.UITestTargetProfilesByBundleID {
				log.Debugf("🔍 [FORCE CODESIGN]   Bundle ID: %s -> Profile: %s", bundleID, profile.Attributes().Name)
			}

			profile, ok := devCodesignAssets.UITestTargetProfilesByBundleID[targetBundleID]
			if !ok {
				log.Debugf("❌ [FORCE CODESIGN] No UITest profile found for bundle ID %s for target %s", targetBundleID, uiTestTarget.Name)
				return fmt.Errorf("no profile ensured for the bundleID %s", targetBundleID)
			}
			log.Debugf("✅ [FORCE CODESIGN] Found UITest profile for target %s (bundle ID %s): %s", uiTestTarget.Name, targetBundleID, profile.Attributes().Name)

			log.Printf("  development Team: %s(%s)", devCodesignAssets.Certificate.TeamName, teamID)
			log.Printf("  provisioning Profile: %s", profile.Attributes().Name)
			log.Printf("  certificate: %s", devCodesignAssets.Certificate.CommonName)

			log.Debugf("🔍 [FORCE CODESIGN] Processing %d build configurations for UITest target %s", len(uiTestTarget.BuildConfigurationList.BuildConfigurations), uiTestTarget.Name)
			for _, c := range uiTestTarget.BuildConfigurationList.BuildConfigurations {
				log.Debugf("🔍 [FORCE CODESIGN] Applying code sign for UITest target %s, configuration %s", uiTestTarget.Name, c.Name)
				if err := p.projHelper.XcProj.ForceCodeSign(c.Name, uiTestTarget.Name, teamID, devCodesignAssets.Certificate.SHA1Fingerprint, profile.Attributes().UUID); err != nil {
					log.Debugf("❌ [FORCE CODESIGN] Failed to apply code sign settings for UITest target %s, config %s: %v", uiTestTarget.Name, c.Name, err)
					return fmt.Errorf("failed to apply code sign settings for target (%s): %s", uiTestTarget.Name, err)
				}
				log.Debugf("✅ [FORCE CODESIGN] Successfully applied code sign for UITest target %s, configuration %s", uiTestTarget.Name, c.Name)
			}
		}
		log.Debugf("🔍 [FORCE CODESIGN] Completed UITest targets processing")
	} else {
		if !isDevelopmentAvailable {
			log.Debugf("🔍 [FORCE CODESIGN] Skipping UITest targets: Development assets not available")
		} else {
			log.Debugf("🔍 [FORCE CODESIGN] Skipping UITest targets: No UITest profiles available")
		}
	}

	log.Debugf("🔍 [FORCE CODESIGN] About to save Xcode project...")
	if err := p.projHelper.XcProj.Save(); err != nil {
		log.Debugf("❌ [FORCE CODESIGN] Failed to save project: %v", err)
		return fmt.Errorf("failed to save project: %s", err)
	}
	log.Debugf("✅ [FORCE CODESIGN] Successfully saved Xcode project")

	log.Debugf("Xcode project saved.")
	log.Debugf("🔍 [FORCE CODESIGN] ForceCodesignAssets completed successfully")

	return nil
}

// CanGenerateProfileWithEntitlements checks all entitlements, whether they can be generated
func CanGenerateProfileWithEntitlements(entitlementsByBundleID map[string]autocodesign.Entitlements) (ok bool, badEntitlement string, badBundleID string) {
	for bundleID, entitlements := range entitlementsByBundleID {
		for entitlementKey, value := range entitlements {
			if (autocodesign.Entitlement{entitlementKey: value}).IsProfileAttached() {
				return false, entitlementKey, bundleID
			}
		}
	}

	return true, "", ""
}
