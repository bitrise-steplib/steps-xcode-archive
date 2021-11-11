// Package projectmanager parses and edits an Xcode project.
//
// Use cases:
//  1. Get codesigning related information, needed to fetch or recreate certificates and provisioning profiles
//  2. Apply codesigning settings in the projects
package projectmanager

import (
	"fmt"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-xcode/autocodesign"
)

// Project ...
type Project struct {
	projHelper ProjectHelper
}

// InitParams ...
type InitParams struct {
	ProjectOrWorkspacePath string
	SchemeName             string
	ConfigurationName      string
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

// MainTargetBundleID ...
func (p Project) MainTargetBundleID() (string, error) {
	bundleID, err := p.projHelper.TargetBundleID(p.projHelper.MainTarget.Name, p.projHelper.Configuration)
	if err != nil {
		return "", fmt.Errorf("failed to read bundle ID for the main target: %s", err)
	}

	return bundleID, nil
}

// IsSigningManagedAutomatically checks the "Automatically manage signing" checkbox in Xcode
// Note: it only checks the main Target based on the given Scheme and Configuration
func (p Project) IsSigningManagedAutomatically() (bool, error) {
	targetName := p.projHelper.MainTarget.Name
	settings, err := p.projHelper.targetBuildSettings(targetName, p.projHelper.Configuration)
	if err != nil {
		return false, fmt.Errorf("failed to fetch code signing info from target (%s) settings: %s", targetName, err)
	}
	codeSignStyle, err := settings.String("CODE_SIGN_STYLE")
	if err != nil {
		return false, fmt.Errorf("failed to fetch code signing info from target (%s) settings: %s", targetName, err)
	}

	return codeSignStyle != "Manual", nil
}

// GetAppLayout ...
func (p Project) GetAppLayout(uiTestTargets bool) (autocodesign.AppLayout, error) {
	log.Printf("Configuration: %s", p.projHelper.Configuration)

	teamID, err := p.projHelper.ProjectTeamID(p.projHelper.Configuration)
	if err != nil {
		return autocodesign.AppLayout{}, fmt.Errorf("failed to read project team ID: %s", err)
	}

	log.Printf("Project team ID: %s", teamID)

	platform, err := p.projHelper.Platform(p.projHelper.Configuration)
	if err != nil {
		return autocodesign.AppLayout{}, fmt.Errorf("failed to read project platform: %s", err)
	}

	log.Printf("Platform: %s", platform)

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
		TeamID:                                 teamID,
		Platform:                               platform,
		EntitlementsByArchivableTargetBundleID: archivableTargetBundleIDToEntitlements,
		UITestTargetBundleIDs:                  uiTestTargetBundleIDs,
	}, nil
}

// ForceCodesignAssets ...
func (p Project) ForceCodesignAssets(distribution autocodesign.DistributionType, codesignAssetsByDistributionType map[autocodesign.DistributionType]autocodesign.AppCodesignAssets) error {
	fmt.Println()
	log.Infof("Apply Bitrise managed codesigning on the executable targets")
	for _, target := range p.projHelper.ArchivableTargets() {
		fmt.Println()
		log.Infof("  Target: %s", target.Name)

		forceCodesignDistribution := distribution
		if _, isDevelopmentAvailable := codesignAssetsByDistributionType[autocodesign.Development]; isDevelopmentAvailable {
			forceCodesignDistribution = autocodesign.Development
		}

		codesignAssets, ok := codesignAssetsByDistributionType[forceCodesignDistribution]
		if !ok {
			return fmt.Errorf("no codesign settings ensured for distribution type %s", forceCodesignDistribution)
		}
		teamID := codesignAssets.Certificate.TeamID

		targetBundleID, err := p.projHelper.TargetBundleID(target.Name, p.projHelper.Configuration)
		if err != nil {
			return err
		}
		profile, ok := codesignAssets.ArchivableTargetProfilesByBundleID[targetBundleID]
		if !ok {
			return fmt.Errorf("no profile ensured for the bundleID %s", targetBundleID)
		}

		log.Printf("  development Team: %s(%s)", codesignAssets.Certificate.TeamName, teamID)
		log.Printf("  provisioning Profile: %s", profile.Attributes().Name)
		log.Printf("  certificate: %s", codesignAssets.Certificate.CommonName)

		if err := p.projHelper.XcProj.ForceCodeSign(p.projHelper.Configuration, target.Name, teamID, codesignAssets.Certificate.SHA1Fingerprint, profile.Attributes().UUID); err != nil {
			return fmt.Errorf("failed to apply code sign settings for target (%s): %s", target.Name, err)
		}
	}

	devCodesignAssets, isDevelopmentAvailable := codesignAssetsByDistributionType[autocodesign.Development]
	if isDevelopmentAvailable && len(devCodesignAssets.UITestTargetProfilesByBundleID) != 0 {
		fmt.Println()
		log.Infof("Apply Bitrise managed codesigning on the UITest targets")
		for _, uiTestTarget := range p.projHelper.UITestTargets {
			fmt.Println()
			log.Infof("  Target: %s", uiTestTarget.Name)

			teamID := devCodesignAssets.Certificate.TeamID

			targetBundleID, err := p.projHelper.TargetBundleID(uiTestTarget.Name, p.projHelper.Configuration)
			if err != nil {
				return err
			}
			profile, ok := devCodesignAssets.UITestTargetProfilesByBundleID[targetBundleID]
			if !ok {
				return fmt.Errorf("no profile ensured for the bundleID %s", targetBundleID)
			}

			log.Printf("  development Team: %s(%s)", devCodesignAssets.Certificate.TeamName, teamID)
			log.Printf("  provisioning Profile: %s", profile.Attributes().Name)
			log.Printf("  certificate: %s", devCodesignAssets.Certificate.CommonName)

			for _, c := range uiTestTarget.BuildConfigurationList.BuildConfigurations {
				if err := p.projHelper.XcProj.ForceCodeSign(c.Name, uiTestTarget.Name, teamID, devCodesignAssets.Certificate.SHA1Fingerprint, profile.Attributes().UUID); err != nil {
					return fmt.Errorf("failed to apply code sign settings for target (%s): %s", uiTestTarget.Name, err)
				}
			}
		}
	}

	if err := p.projHelper.XcProj.Save(); err != nil {
		return fmt.Errorf("failed to save project: %s", err)
	}

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
