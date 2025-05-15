package exportoptionsgenerator

import (
	"fmt"
	"os"

	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/export"
	"github.com/bitrise-io/go-xcode/exportoptions"
	"github.com/bitrise-io/go-xcode/plistutil"
	"github.com/bitrise-io/go-xcode/profileutil"
	"github.com/bitrise-io/go-xcode/v2/xcodeversion"
)

const (
	// AppClipProductType ...
	AppClipProductType = "com.apple.product-type.application.on-demand-install-capable"
)

// Opts contains options for the exportOptions generator.
type Opts struct {
	ContainerEnvironment             string
	TeamID                           string
	UploadBitcode                    bool
	CompileBitcode                   bool
	ArchivedWithXcodeManagedProfiles bool
	TestFlightInternalTestingOnly    bool
	ManageVersionAndBuildNumber      bool
}

// ExportOptionsGenerator generates an exportOptions.plist file.
type ExportOptionsGenerator struct {
	xcodeVersionReader  xcodeversion.Reader
	certificateProvider CodesignIdentityProvider
	profileProvider     ProvisioningProfileProvider
	logger              log.Logger
}

// New constructs a new ExportOptionsGenerator.
func New(xcodeVersionReader xcodeversion.Reader, logger log.Logger) ExportOptionsGenerator {
	return ExportOptionsGenerator{
		xcodeVersionReader:  xcodeVersionReader,
		certificateProvider: LocalCodesignIdentityProvider{},
		profileProvider:     LocalProvisioningProfileProvider{},
		logger:              logger,
	}
}

// GenerateApplicationExportOptions generates exportOptions for an application export.
func (g ExportOptionsGenerator) GenerateApplicationExportOptions(
	exportedProduct ExportProduct,
	archiveInfo ArchiveInfo,
	exportMethod exportoptions.Method,
	codeSigningStyle exportoptions.SigningStyle,
	opts Opts,
) (exportoptions.ExportOptions, error) {
	xcodeVersion, err := g.xcodeVersionReader.GetVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get Xcode version: %w", err)
	}

	// BundleIDs appear in the export options plist in:
	// - distributionBundleIdentifier: can be the main app or the app Clip bundle ID.
	//   It is only valid for NON app-store-connect distribution. App Store export includes both app and app-clip in one go, others do not.
	// - provisioningProfiles dictionary:
	//  When distributing an app-clip, its bundle ID needs to be in the provisioningProfiles dictionary, otherwise it needs to be removed.
	productToDistributeBundleID := archiveInfo.AppBundleID
	if exportedProduct == ExportProductAppClip {
		if archiveInfo.AppClipBundleID == "" {
			return nil, fmt.Errorf("xcarchive does not contain an App Clip, cannot export an App Clip")
		}

		if exportMethod.IsAppStore() {
			g.logger.Warnf("Selected app-clip for distribution, but distribution method is the App Store.\n" +
				"Exported .app will contain both the app and the app-clip for App Store exports.\n")
		}
		productToDistributeBundleID = archiveInfo.AppClipBundleID
	}

	if exportedProduct != ExportProductAppClip {
		for bundleID := range archiveInfo.EntitlementsByBundleID {
			if bundleID == archiveInfo.AppClipBundleID && !exportMethod.IsAppStore() {
				g.logger.Debugf("Filtering out App Clip target, as non App Store distribution is used: %s", bundleID)
				delete(archiveInfo.EntitlementsByBundleID, bundleID)
			}
		}
	}

	iCloudContainerEnvironment, err := determineIcloudContainerEnvironment(opts.ContainerEnvironment, archiveInfo.EntitlementsByBundleID, exportMethod, xcodeVersion.Major)
	if err != nil {
		return nil, err
	}

	exportOpts := generateBaseExportOptions(exportMethod, xcodeVersion, opts.UploadBitcode, opts.CompileBitcode, iCloudContainerEnvironment)

	if xcodeVersion.Major >= 12 {
		exportOpts = addDistributionBundleIdentifierFromXcode12(exportOpts, productToDistributeBundleID)
	}

	if xcodeVersion.Major >= 13 {
		exportOpts = addManagedBuildNumberFromXcode13(exportOpts, opts.ManageVersionAndBuildNumber)
	}

	if codeSigningStyle == exportoptions.SigningStyleAutomatic {
		exportOpts = addTeamID(exportOpts, opts.TeamID)
	} else {
		codeSignGroup, err := g.determineCodesignGroup(archiveInfo.EntitlementsByBundleID, exportMethod, opts.TeamID, opts.ArchivedWithXcodeManagedProfiles)
		if err != nil {
			return nil, err
		}
		if codeSignGroup == nil {
			return exportOpts, nil
		}

		exportOpts = addManualSigningFields(exportOpts, codeSignGroup, opts.ArchivedWithXcodeManagedProfiles, g.logger)
	}

	if xcodeVersion.Major >= 15 {
		if opts.TestFlightInternalTestingOnly {
			exportOpts = addTestFlightInternalTestingOnly(exportOpts, opts.TestFlightInternalTestingOnly)
		}
	}

	return exportOpts, nil
}

// determineCodesignGroup finds the best codesign group (certificate + profiles)
// based on the installed Provisioning Profiles and Codesign Certificates.
func (g ExportOptionsGenerator) determineCodesignGroup(bundleIDEntitlementsMap map[string]plistutil.PlistData, exportMethod exportoptions.Method, teamID string, xcodeManaged bool) (*export.IosCodeSignGroup, error) {
	fmt.Println()
	g.logger.Printf("Target Bundle ID - Entitlements map")
	var bundleIDs []string
	for bundleID, entitlements := range bundleIDEntitlementsMap {
		bundleIDs = append(bundleIDs, bundleID)

		var entitlementKeys []string
		for key := range entitlements {
			entitlementKeys = append(entitlementKeys, key)
		}
		g.logger.Printf("%s: %s", bundleID, entitlementKeys)
	}

	fmt.Println()
	g.logger.Printf("Resolving CodeSignGroups...")

	certs, err := g.certificateProvider.ListCodesignIdentities()
	if err != nil {
		return nil, fmt.Errorf("failed to get installed certificates: %w", err)
	}

	g.logger.Debugf("Installed certificates:")
	for _, certInfo := range certs {
		g.logger.Debugf(certInfo.String())
	}

	profs, err := g.profileProvider.ListProvisioningProfiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get installed provisioning profiles: %w", err)
	}

	g.logger.Debugf("Installed profiles:")
	for _, profileInfo := range profs {
		g.logger.Debugf(profileInfo.String(certs...))
	}

	g.logger.Printf("Resolving CodeSignGroups...")
	codeSignGroups := export.CreateSelectableCodeSignGroups(certs, profs, bundleIDs)
	if len(codeSignGroups) == 0 {
		g.logger.Errorf("Failed to find code signing groups for specified export method (%s)", exportMethod)
	}

	g.logger.Debugf("\nGroups:")
	for _, group := range codeSignGroups {
		g.logger.Debugf(group.String())
	}

	if len(bundleIDEntitlementsMap) > 0 {
		g.logger.Warnf("Filtering CodeSignInfo groups for target capabilities")

		codeSignGroups = export.FilterSelectableCodeSignGroups(codeSignGroups, export.CreateEntitlementsSelectableCodeSignGroupFilter(bundleIDEntitlementsMap))

		g.logger.Debugf("\nGroups after filtering for target capabilities:")
		for _, group := range codeSignGroups {
			g.logger.Debugf(group.String())
		}
	}

	g.logger.Warnf("Filtering CodeSignInfo groups for export method")

	codeSignGroups = export.FilterSelectableCodeSignGroups(codeSignGroups, export.CreateExportMethodSelectableCodeSignGroupFilter(exportMethod))

	g.logger.Debugf("\nGroups after filtering for export method:")
	for _, group := range codeSignGroups {
		g.logger.Debugf(group.String())
	}

	if teamID != "" {
		g.logger.Warnf("ExportDevelopmentTeam specified: %s, filtering CodeSignInfo groups...", teamID)

		codeSignGroups = export.FilterSelectableCodeSignGroups(codeSignGroups, export.CreateTeamSelectableCodeSignGroupFilter(teamID))

		g.logger.Debugf("\nGroups after filtering for team ID:")
		for _, group := range codeSignGroups {
			g.logger.Debugf(group.String())
		}
	}

	if !xcodeManaged {
		g.logger.Warnf("App was signed with NON Xcode managed profile when archiving,\n" +
			"only NOT Xcode managed profiles are allowed to sign when exporting the archive.\n" +
			"Removing Xcode managed CodeSignInfo groups")

		codeSignGroups = export.FilterSelectableCodeSignGroups(codeSignGroups, export.CreateNotXcodeManagedSelectableCodeSignGroupFilter())

		g.logger.Debugf("\nGroups after filtering for NOT Xcode managed profiles:")
		for _, group := range codeSignGroups {
			g.logger.Debugf(group.String())
		}
	}

	defaultProfileURL := os.Getenv("BITRISE_DEFAULT_PROVISION_URL")
	if teamID == "" && defaultProfileURL != "" {
		if defaultProfile, err := g.profileProvider.GetDefaultProvisioningProfile(); err == nil {
			g.logger.Debugf("\ndefault profile: %v\n", defaultProfile)
			filteredCodeSignGroups := export.FilterSelectableCodeSignGroups(codeSignGroups,
				export.CreateExcludeProfileNameSelectableCodeSignGroupFilter(defaultProfile.Name))
			if len(filteredCodeSignGroups) > 0 {
				codeSignGroups = filteredCodeSignGroups

				g.logger.Debugf("\nGroups after removing default profile:")
				for _, group := range codeSignGroups {
					g.logger.Debugf(group.String())
				}
			}
		}
	}

	var iosCodeSignGroups []export.IosCodeSignGroup

	for _, selectable := range codeSignGroups {
		bundleIDProfileMap := map[string]profileutil.ProvisioningProfileInfoModel{}
		for bundleID, profiles := range selectable.BundleIDProfilesMap {
			if len(profiles) > 0 {
				bundleIDProfileMap[bundleID] = profiles[0]
			} else {
				g.logger.Warnf("No profile available to sign (%s) target!", bundleID)
			}
		}

		iosCodeSignGroups = append(iosCodeSignGroups, *export.NewIOSGroup(selectable.Certificate, bundleIDProfileMap))
	}

	g.logger.Debugf("\nFiltered groups:")
	for i, group := range iosCodeSignGroups {
		g.logger.Debugf("Group #%d:", i)
		for bundleID, profile := range group.BundleIDProfileMap() {
			g.logger.Debugf(" - %s: %s (%s)", bundleID, profile.Name, profile.UUID)
		}
	}

	if len(iosCodeSignGroups) < 1 {
		g.logger.Errorf("Failed to find Codesign Groups")
		return nil, nil
	}

	if len(iosCodeSignGroups) > 1 {
		g.logger.Warnf("Multiple code signing groups found! Using the first code signing group")
	}

	return &iosCodeSignGroups[0], nil
}

// determineIcloudContainerEnvironment calculates the value of iCloudContainerEnvironment.
func determineIcloudContainerEnvironment(desiredIcloudContainerEnvironment string, bundleIDEntitlementsMap map[string]plistutil.PlistData, exportMethod exportoptions.Method, xcodeMajorVersion int64) (string, error) {
	// iCloudContainerEnvironment: If the app is using CloudKit, this configures the "com.apple.developer.icloud-container-environment" entitlement.
	// Available options vary depending on the type of provisioning profile used, but may include: Development and Production.
	usesCloudKit := projectUsesCloudKit(bundleIDEntitlementsMap)
	if !usesCloudKit {
		return "", nil
	}

	// From Xcode 9 iCloudContainerEnvironment is required for every export method, before that version only for non app-store exports.
	if xcodeMajorVersion < 9 && exportMethod.IsAppStore() {
		return "", nil
	}

	if exportMethod.IsAppStore() {
		return "Production", nil
	}

	if desiredIcloudContainerEnvironment == "" {
		return "", fmt.Errorf("Your project uses CloudKit but \"iCloud container environment\" input not specified.\n"+
			"Export method is: %s (For app-store export method Production container environment is implied.)", exportMethod)
	}

	return desiredIcloudContainerEnvironment, nil
}

// projectUsesCloudKit determines whether the project uses any CloudKit capability or not.
func projectUsesCloudKit(bundleIDEntitlementsMap map[string]plistutil.PlistData) bool {
	fmt.Printf("Checking if project uses CloudKit")

	for _, entitlements := range bundleIDEntitlementsMap {
		if entitlements == nil {
			continue
		}

		services, ok := entitlements.GetStringArray("com.apple.developer.icloud-services")
		if !ok {
			continue
		}

		if sliceutil.IsStringInSlice("CloudKit", services) || sliceutil.IsStringInSlice("CloudDocuments", services) {
			fmt.Printf("Project uses CloudKit")

			return true
		}
	}
	return false
}

// generateBaseExportOptions creates a default exportOptions introduced in Xcode 7.
func generateBaseExportOptions(exportMethod exportoptions.Method, xcodeVersion xcodeversion.Version, cfgUploadBitcode, cfgCompileBitcode bool, iCloudContainerEnvironment string) exportoptions.ExportOptions {
	if xcodeVersion.IsGreaterThanOrEqualTo(15, 3) {
		exportMethod = exportoptions.UpgradeToXcode15_3MethodName(exportMethod)
	}

	if exportMethod.IsAppStore() {
		appStoreOptions := exportoptions.NewAppStoreConnectOptions(exportMethod)
		appStoreOptions.UploadBitcode = cfgUploadBitcode
		if iCloudContainerEnvironment != "" {
			appStoreOptions.ICloudContainerEnvironment = exportoptions.ICloudContainerEnvironment(iCloudContainerEnvironment)
		}
		return appStoreOptions
	}

	nonAppStoreOptions := exportoptions.NewNonAppStoreOptions(exportMethod)
	nonAppStoreOptions.CompileBitcode = cfgCompileBitcode

	if iCloudContainerEnvironment != "" {
		nonAppStoreOptions.ICloudContainerEnvironment = exportoptions.ICloudContainerEnvironment(iCloudContainerEnvironment)
	}

	return nonAppStoreOptions
}

func addDistributionBundleIdentifierFromXcode12(exportOpts exportoptions.ExportOptions, distributionBundleIdentifier string) exportoptions.ExportOptions {
	switch options := exportOpts.(type) {
	case exportoptions.AppStoreOptionsModel:
		// Export option plist with App store export method (Xcode 12.0.1) do not contain distribution bundle identifier.
		// Probably due to App store IPAs containing App Clips also, which are executable targets with a separate bundle ID.
		return exportOpts
	case exportoptions.NonAppStoreOptionsModel:
		options.DistributionBundleIdentifier = distributionBundleIdentifier
		return options
	}
	return nil
}

func addManagedBuildNumberFromXcode13(exportOpts exportoptions.ExportOptions, isManageAppVersion bool) exportoptions.ExportOptions {
	switch options := exportOpts.(type) {
	case exportoptions.AppStoreOptionsModel:
		options.ManageAppVersion = isManageAppVersion // Only available for app-store exports

		return options
	}

	return exportOpts
}

func addTeamID(exportOpts exportoptions.ExportOptions, teamID string) exportoptions.ExportOptions {
	switch options := exportOpts.(type) {
	case exportoptions.AppStoreOptionsModel:
		options.TeamID = teamID
		return options
	case exportoptions.NonAppStoreOptionsModel:
		options.TeamID = teamID
		return options
	}
	return exportOpts
}

func addTestFlightInternalTestingOnly(exportOpts exportoptions.ExportOptions, testFlightInternalTestingOnly bool) exportoptions.ExportOptions {
	switch options := exportOpts.(type) {
	case exportoptions.AppStoreOptionsModel:
		options.TestFlightInternalTestingOnly = testFlightInternalTestingOnly // Only available for app-store exports
		return options
	}

	return exportOpts
}

func addManualSigningFields(exportOpts exportoptions.ExportOptions, codeSignGroup *export.IosCodeSignGroup, archivedWithXcodeManagedProfiles bool, logger log.Logger) exportoptions.ExportOptions {
	exportCodeSignStyle := ""
	exportProfileMapping := map[string]string{}

	for bundleID, profileInfo := range codeSignGroup.BundleIDProfileMap() {
		exportProfileMapping[bundleID] = profileInfo.Name

		isXcodeManaged := profileutil.IsXcodeManaged(profileInfo.Name)
		if isXcodeManaged {
			if exportCodeSignStyle != "" && exportCodeSignStyle != "automatic" {
				logger.Errorf("Both Xcode managed and NON Xcode managed profiles in code signing group")
			}
			exportCodeSignStyle = "automatic"
		} else {
			if exportCodeSignStyle != "" && exportCodeSignStyle != string(exportoptions.SigningStyleManual) {
				logger.Errorf("Both Xcode managed and NON Xcode managed profiles in code signing group")
			}
			exportCodeSignStyle = string(exportoptions.SigningStyleManual)
		}
	}

	shouldSetManualSigning := archivedWithXcodeManagedProfiles && exportCodeSignStyle == string(exportoptions.SigningStyleManual)
	if shouldSetManualSigning {
		logger.Warnf("App was signed with Xcode managed profile when archiving,")
		logger.Warnf("ipa export uses manual code signing.")
		logger.Warnf(`Setting "signingStyle" to "manual".`)
	}

	logger.TDebugf("Determined code signing style")

	switch options := exportOpts.(type) {
	case exportoptions.AppStoreOptionsModel:
		options.BundleIDProvisioningProfileMapping = exportProfileMapping
		options.SigningCertificate = codeSignGroup.Certificate().CommonName
		options.TeamID = codeSignGroup.Certificate().TeamID

		if shouldSetManualSigning {
			options.SigningStyle = exportoptions.SigningStyleManual
		}
		exportOpts = options
	case exportoptions.NonAppStoreOptionsModel:
		options.BundleIDProvisioningProfileMapping = exportProfileMapping
		options.SigningCertificate = codeSignGroup.Certificate().CommonName
		options.TeamID = codeSignGroup.Certificate().TeamID

		if shouldSetManualSigning {
			options.SigningStyle = exportoptions.SigningStyleManual
		}
		exportOpts = options
	}

	return exportOpts
}
