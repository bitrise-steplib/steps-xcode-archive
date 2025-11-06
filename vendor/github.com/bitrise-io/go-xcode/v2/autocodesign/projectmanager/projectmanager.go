// Package projectmanager parses and edits an Xcode project.
//
// Use cases:
//  1. Get codesigning related information, needed to fetch or recreate certificates and provisioning profiles
//  2. Apply codesigning settings in the projects
package projectmanager

import (
	"fmt"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/v2/autocodesign"
)

// Project ...
type Project struct {
	logger     log.Logger
	projHelper ProjectHelper
}

// Factory ...
type Factory struct {
	logger      log.Logger
	buildAction BuildAction
}

// InitParams ...
type InitParams struct {
	Logger                 log.Logger
	ProjectOrWorkspacePath string
	BuildAction            BuildAction
	SchemeName             string
	ConfigurationName      string
	IsDebug                bool
}

// NewFactory ...
func NewFactory(logger log.Logger, buildAction BuildAction) Factory {
	return Factory{logger: logger, buildAction: buildAction}
}

// Create ...
func (f *Factory) Create(params InitParams) (Project, error) {
	return NewProject(params)
}

// NewProject ...
func NewProject(params InitParams) (Project, error) {
	projectHelper, err := NewProjectHelper(params.ProjectOrWorkspacePath, params.Logger, params.SchemeName, params.BuildAction, params.ConfigurationName, params.IsDebug)
	if err != nil {
		return Project{}, err
	}

	return Project{
		projHelper: *projectHelper,
		logger:     params.Logger,
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

	p.logger.Debugf("Platform: %s", platform)

	return platform, nil
}

// IsMainTargetProductTypeAppClip returns true if the main target is of App Clip product type
func (p Project) IsMainTargetProductTypeAppClip() bool {
	return p.projHelper.MainTarget.IsAppClipProduct()
}

// MainTargetBundleID returns the bundle ID of the main target
func (p Project) MainTargetBundleID() (string, error) {
	bundleID, err := p.projHelper.TargetBundleID(p.projHelper.MainTarget.Name, p.projHelper.Configuration)
	if err != nil {
		return "", fmt.Errorf("failed to read bundle ID for the main target: %s", err)
	}

	return bundleID, nil
}

// ReadSchemeBuildSettingString reads a build setting (for example SDKROOT) from the workspace (if available) or project.
func (p Project) ReadSchemeBuildSettingString(key string) (string, error) {
	target := p.projHelper.MainTarget.Name
	value, err := p.projHelper.buildSettingForKey(target, p.projHelper.Configuration, key)
	if err != nil {
		return "", fmt.Errorf("failed to read build setting `%s` for the target `%s`: %s", key, target, err)
	}

	return value, nil
}

// GetAppLayout ...
func (p Project) GetAppLayout(uiTestTargets bool) (autocodesign.AppLayout, error) {
	p.logger.Printf("Configuration: %s", p.projHelper.Configuration)

	platform, err := p.projHelper.Platform(p.projHelper.Configuration)
	if err != nil {
		return autocodesign.AppLayout{}, fmt.Errorf("failed to read project platform: %s", err)
	}

	p.logger.Debugf("Platform: %s", platform)

	p.logger.Printf("Application and App Extension targets:")
	for _, target := range p.projHelper.ArchivableTargets() {
		p.logger.Printf("- %s", target.Name)
	}

	archivableTargetBundleIDToEntitlements, err := p.projHelper.ArchivableTargetBundleIDToEntitlements()
	if err != nil {
		return autocodesign.AppLayout{}, fmt.Errorf("failed to read archivable targets' entitlements: %s", err)
	}

	if ok, entitlement, bundleID := CanGenerateProfileWithEntitlements(archivableTargetBundleIDToEntitlements); !ok {
		p.logger.Errorf("Can not create profile with unsupported entitlement (%s) for the bundle ID %s, due to App Store Connect API limitations.", entitlement, bundleID)
		return autocodesign.AppLayout{}, fmt.Errorf("please generate provisioning profile manually on Apple Developer Portal and use the Certificate and profile installer Step instead")
	}

	var uiTestTargetBundleIDs []string
	if uiTestTargets {
		p.logger.Printf("UITest targets:")
		for _, target := range p.projHelper.UITestTargets {
			p.logger.Printf("- %s", target.Name)
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
	archivableTargets := p.projHelper.ArchivableTargets()
	var archivableTargetsCounter = 0

	fmt.Println()
	p.logger.TInfof("Apply Bitrise managed codesigning on the executable targets (up to: %d targets)", len(archivableTargets))

	for _, target := range archivableTargets {
		fmt.Println()
		p.logger.Infof("  Target: %s", target.Name)

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

		p.logger.Printf("  development Team: %s(%s)", codesignAssets.Certificate.TeamName, teamID)
		p.logger.Printf("  provisioning Profile: %s", profile.Attributes().Name)
		p.logger.Printf("  certificate: %s", codesignAssets.Certificate.CommonName)

		if err := p.projHelper.XcProj.ForceCodeSign(p.projHelper.Configuration, target.Name, teamID, codesignAssets.Certificate.SHA1Fingerprint, profile.Attributes().UUID); err != nil {
			return fmt.Errorf("failed to apply code sign settings for target (%s): %s", target.Name, err)
		}

		archivableTargetsCounter++
	}

	p.logger.TInfof("Applied Bitrise managed codesigning on up to %d targets", archivableTargetsCounter)

	devCodesignAssets, isDevelopmentAvailable := codesignAssetsByDistributionType[autocodesign.Development]
	if isDevelopmentAvailable && len(devCodesignAssets.UITestTargetProfilesByBundleID) != 0 {
		fmt.Println()
		p.logger.TInfof("Apply Bitrise managed codesigning on the UITest targets (%d)", len(p.projHelper.UITestTargets))

		for _, uiTestTarget := range p.projHelper.UITestTargets {
			fmt.Println()
			p.logger.Infof("  Target: %s", uiTestTarget.Name)

			teamID := devCodesignAssets.Certificate.TeamID

			targetBundleID, err := p.projHelper.TargetBundleID(uiTestTarget.Name, p.projHelper.Configuration)
			if err != nil {
				return err
			}
			profile, ok := devCodesignAssets.UITestTargetProfilesByBundleID[targetBundleID]
			if !ok {
				return fmt.Errorf("no profile ensured for the bundleID %s", targetBundleID)
			}

			p.logger.Printf("  development Team: %s(%s)", devCodesignAssets.Certificate.TeamName, teamID)
			p.logger.Printf("  provisioning Profile: %s", profile.Attributes().Name)
			p.logger.Printf("  certificate: %s", devCodesignAssets.Certificate.CommonName)

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

	p.logger.Debugf("Xcode project saved.")

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
