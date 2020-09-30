package main

import (
	"fmt"
	"os"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-io/go-xcode/certificateutil"
	"github.com/bitrise-io/go-xcode/export"
	"github.com/bitrise-io/go-xcode/exportoptions"
	"github.com/bitrise-io/go-xcode/plistutil"
	"github.com/bitrise-io/go-xcode/profileutil"
	"github.com/bitrise-io/xcode-project/serialized"
	"github.com/bitrise-io/xcode-project/xcodeproj"
	"github.com/bitrise-io/xcode-project/xcscheme"
	"github.com/bitrise-steplib/steps-xcode-archive/utils"
)

const appClipProductType = "com.apple.product-type.application.on-demand-install-capable"

// ExportOptionsGenerator generates an exportOptions.plist file from Xcode version 7 to Xcode version 11.
type ExportOptionsGenerator struct {
	xcodeProj     *xcodeproj.XcodeProj
	scheme        *xcscheme.Scheme
	configuration string

	certificateProvider CodesignIdentityProvider
	profileProvider     ProvisioningProfileProvider
	targetInfoProvider  TargetInfoProvider
}

// NewExportOptionsGenerator constructs a new ExportOptionsGenerator.
func NewExportOptionsGenerator(xcodeProj *xcodeproj.XcodeProj, scheme *xcscheme.Scheme, configuration string) ExportOptionsGenerator {
	g := ExportOptionsGenerator{
		xcodeProj:     xcodeProj,
		scheme:        scheme,
		configuration: configuration,
	}
	g.certificateProvider = LocalCodesignIdentityProvider{}
	g.profileProvider = LocalProvisioningProfileProvider{}
	g.targetInfoProvider = XcodebuildTargetInfoProvider{xcodeProj: xcodeProj}
	return g
}

// GenerateApplicationExportOptions generates exportOptions for an application export.
func (g ExportOptionsGenerator) GenerateApplicationExportOptions(exportMethod exportoptions.Method, containerEnvironment string, teamID string, uploadBitcode bool, compileBitcode bool, xcodeManaged bool,
	xcodeMajorVersion int64) (exportoptions.ExportOptions, error) {
	mainTarget, err := archivableApplicationTarget(g.xcodeProj, g.scheme, g.configuration)
	if err != nil {
		return nil, err
	}

	dependentTargets := dependentApplicationBundleTargetsOf(exportMethod, *mainTarget)

	targets := append([]xcodeproj.Target{*mainTarget}, dependentTargets...)

	var mainTargetBundleID string
	entitlementsByBundleID := map[string]plistutil.PlistData{}
	for i, target := range targets {
		bundleID, err := g.targetInfoProvider.TargetBundleID(target.Name, g.configuration)
		if err != nil {
			return nil, fmt.Errorf("failed to get target (%s) bundle id: %s", target.Name, err)
		}

		entitlements, err := g.targetInfoProvider.TargetCodeSignEntitlements(target.Name, g.configuration)
		if err != nil && !serialized.IsKeyNotFoundError(err) {
			return nil, fmt.Errorf("failed to get target (%s) bundle id: %s", target.Name, err)
		}

		entitlementsByBundleID[bundleID] = plistutil.PlistData(entitlements)

		if i == 0 {
			mainTargetBundleID = bundleID
		}
	}

	return g.generateExportOptions(exportMethod, containerEnvironment, teamID, uploadBitcode, compileBitcode,
		xcodeManaged, entitlementsByBundleID, xcodeMajorVersion, mainTargetBundleID)
}

// TargetInfoProvider can determine a target's bundle id and codesign entitlements.
type TargetInfoProvider interface {
	TargetBundleID(target, configuration string) (string, error)
	TargetCodeSignEntitlements(target, configuration string) (serialized.Object, error)
}

// XcodebuildTargetInfoProvider implements TargetInfoProvider.
type XcodebuildTargetInfoProvider struct {
	xcodeProj *xcodeproj.XcodeProj
}

// TargetBundleID ...
func (b XcodebuildTargetInfoProvider) TargetBundleID(target, configuration string) (string, error) {
	return b.xcodeProj.TargetBundleID(target, configuration)
}

// TargetCodeSignEntitlements ...
func (b XcodebuildTargetInfoProvider) TargetCodeSignEntitlements(target, configuration string) (serialized.Object, error) {
	return b.xcodeProj.TargetCodeSignEntitlements(target, configuration)
}

func archivableApplicationTarget(xcodeProj *xcodeproj.XcodeProj, scheme *xcscheme.Scheme, configurationName string) (*xcodeproj.Target, error) {
	archiveEntry, ok := scheme.AppBuildActionEntry()
	if !ok {
		return nil, fmt.Errorf("archivable entry not found in project: %s for scheme: %s", xcodeProj.Path, scheme.Name)
	}

	mainTarget, ok := xcodeProj.Proj.Target(archiveEntry.BuildableReference.BlueprintIdentifier)
	if !ok {
		return nil, fmt.Errorf("target not found: %s", archiveEntry.BuildableReference.BlueprintIdentifier)
	}

	return &mainTarget, nil
}

func dependentApplicationBundleTargetsOf(exportMethod exportoptions.Method, applicationtarget xcodeproj.Target) (dependentTargets []xcodeproj.Target) {
	for _, target := range applicationtarget.DependentExecutableProductTargets(false) {
		// App store exports contain App Clip too. App Clip provisioning profile has to be included in export options:
		// ..
		// <key>provisioningProfiles</key>
		// <dict>
		// 	<key>io.bundle.id</key>
		// 	<string>Development Application Profile</string>
		// 	<key>io.bundle.id.AppClipID</key>
		// 	<string>Development App Clip Profile</string>
		// </dict>
		// ..,
		if exportMethod != exportoptions.MethodAppStore &&
			target.ProductType == appClipProductType {
			continue
		}

		dependentTargets = append(dependentTargets, target)
	}
	return
}

// projectUsesCloudKit determines whether the project uses any CloudKit capability or not.
func projectUsesCloudKit(bundleIDEntitlementsMap map[string]plistutil.PlistData) bool {
	for _, entitlements := range bundleIDEntitlementsMap {
		if entitlements == nil {
			continue
		}

		services, ok := entitlements.GetStringArray("com.apple.developer.icloud-services")
		if !ok {
			continue
		}

		if sliceutil.IsStringInSlice("CloudKit", services) || sliceutil.IsStringInSlice("CloudDocuments", services) {
			return true
		}
	}
	return false
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
	if xcodeMajorVersion < 9 && exportMethod == exportoptions.MethodAppStore {
		return "", nil
	}

	if exportMethod == exportoptions.MethodAppStore {
		return "Production", nil
	}

	if desiredIcloudContainerEnvironment == "" {
		return "", fmt.Errorf("Your project uses CloudKit but \"iCloud container environment\" input not specified.\n"+
			"Export method is: %s (For app-store export method Production container environment is implied.)", exportMethod)
	}

	return desiredIcloudContainerEnvironment, nil
}

// generateBaseExportOptions creates a default exportOptions introudced in Xcode 7.
func generateBaseExportOptions(exportMethod exportoptions.Method, cfgUploadBitcode, cfgCompileBitcode bool, iCloudContainerEnvironment string) exportoptions.ExportOptions {
	if exportMethod == exportoptions.MethodAppStore {
		appStoreOptions := exportoptions.NewAppStoreOptions()
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

// CodesignIdentityProvider can list certificate infos.
type CodesignIdentityProvider interface {
	ListCodesignIdentities() ([]certificateutil.CertificateInfoModel, error)
}

// LocalCodesignIdentityProvider ...
type LocalCodesignIdentityProvider struct{}

// ListCodesignIdentities ...
func (p LocalCodesignIdentityProvider) ListCodesignIdentities() ([]certificateutil.CertificateInfoModel, error) {
	certs, err := certificateutil.InstalledCodesigningCertificateInfos()
	if err != nil {
		return nil, err
	}
	certInfo := certificateutil.FilterValidCertificateInfos(certs)
	return append(certInfo.ValidCertificates, certInfo.DuplicatedCertificates...), nil
}

// ProvisioningProfileProvider can list profile infos.
type ProvisioningProfileProvider interface {
	ListProvisioningProfiles() ([]profileutil.ProvisioningProfileInfoModel, error)
}

// LocalProvisioningProfileProvider ...
type LocalProvisioningProfileProvider struct{}

// ListProvisioningProfiles ...
func (p LocalProvisioningProfileProvider) ListProvisioningProfiles() ([]profileutil.ProvisioningProfileInfoModel, error) {
	return profileutil.InstalledProvisioningProfileInfos(profileutil.ProfileTypeIos)
}

// determineCodesignGroup finds the best codesign group (certificate + profiles)
// based on the installed Provisioning Profiles and Codesign Certificates.
func (g ExportOptionsGenerator) determineCodesignGroup(bundleIDEntitlementsMap map[string]plistutil.PlistData, exportMethod exportoptions.Method, teamID string, xcodeManaged bool) (*export.IosCodeSignGroup, error) {
	log.Printf("xcode major version > 9, generating provisioningProfiles node")

	fmt.Println()
	log.Printf("Target Bundle ID - Entitlements map")
	var bundleIDs []string
	for bundleID, entitlements := range bundleIDEntitlementsMap {
		bundleIDs = append(bundleIDs, bundleID)

		entitlementKeys := []string{}
		for key := range entitlements {
			entitlementKeys = append(entitlementKeys, key)
		}
		log.Printf("%s: %s", bundleID, entitlementKeys)
	}

	fmt.Println()
	log.Printf("Resolving CodeSignGroups...")

	certs, err := g.certificateProvider.ListCodesignIdentities()
	if err != nil {
		return nil, fmt.Errorf("Failed to get installed certificates, error: %s", err)
	}

	log.Debugf("Installed certificates:")
	for _, certInfo := range certs {
		log.Debugf(certInfo.String())
	}

	profs, err := g.profileProvider.ListProvisioningProfiles()
	if err != nil {
		return nil, fmt.Errorf("Failed to get installed provisioning profiles, error: %s", err)
	}

	log.Debugf("Installed profiles:")
	for _, profileInfo := range profs {
		log.Debugf(profileInfo.String(certs...))
	}

	log.Printf("Resolving CodeSignGroups...")
	codeSignGroups := export.CreateSelectableCodeSignGroups(certs, profs, bundleIDs)
	if len(codeSignGroups) == 0 {
		log.Errorf("Failed to find code signing groups for specified export method (%s)", exportMethod)
	}

	log.Debugf("\nGroups:")
	for _, group := range codeSignGroups {
		log.Debugf(group.String())
	}

	if len(bundleIDEntitlementsMap) > 0 {
		log.Warnf("Filtering CodeSignInfo groups for target capabilities")

		codeSignGroups = export.FilterSelectableCodeSignGroups(codeSignGroups, export.CreateEntitlementsSelectableCodeSignGroupFilter(bundleIDEntitlementsMap))

		log.Debugf("\nGroups after filtering for target capabilities:")
		for _, group := range codeSignGroups {
			log.Debugf(group.String())
		}
	}

	log.Warnf("Filtering CodeSignInfo groups for export method")

	codeSignGroups = export.FilterSelectableCodeSignGroups(codeSignGroups, export.CreateExportMethodSelectableCodeSignGroupFilter(exportMethod))

	log.Debugf("\nGroups after filtering for export method:")
	for _, group := range codeSignGroups {
		log.Debugf(group.String())
	}

	if teamID != "" {
		log.Warnf("Export TeamID specified: %s, filtering CodeSignInfo groups...", teamID)

		codeSignGroups = export.FilterSelectableCodeSignGroups(codeSignGroups, export.CreateTeamSelectableCodeSignGroupFilter(teamID))

		log.Debugf("\nGroups after filtering for team ID:")
		for _, group := range codeSignGroups {
			log.Debugf(group.String())
		}
	}

	if !xcodeManaged {
		log.Warnf("App was signed with NON xcode managed profile when archiving,\n" +
			"only NOT xcode managed profiles are allowed to sign when exporting the archive.\n" +
			"Removing xcode managed CodeSignInfo groups")

		codeSignGroups = export.FilterSelectableCodeSignGroups(codeSignGroups, export.CreateNotXcodeManagedSelectableCodeSignGroupFilter())

		log.Debugf("\nGroups after filtering for NOT Xcode managed profiles:")
		for _, group := range codeSignGroups {
			log.Debugf(group.String())
		}
	}

	defaultProfileURL := os.Getenv("BITRISE_DEFAULT_PROVISION_URL")
	if teamID == "" && defaultProfileURL != "" {
		if defaultProfile, err := utils.GetDefaultProvisioningProfile(); err == nil {
			log.Debugf("\ndefault profile: %v\n", defaultProfile)
			filteredCodeSignGroups := export.FilterSelectableCodeSignGroups(codeSignGroups,
				export.CreateExcludeProfileNameSelectableCodeSignGroupFilter(defaultProfile.Name))
			if len(filteredCodeSignGroups) > 0 {
				codeSignGroups = filteredCodeSignGroups

				log.Debugf("\nGroups after removing default profile:")
				for _, group := range codeSignGroups {
					log.Debugf(group.String())
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
				log.Warnf("No profile available to sign (%s) target!", bundleID)
			}
		}

		iosCodeSignGroups = append(iosCodeSignGroups, *export.NewIOSGroup(selectable.Certificate, bundleIDProfileMap))
	}

	log.Debugf("\nFiltered groups:")
	for i, group := range iosCodeSignGroups {
		log.Debugf("Group #%d:", i)
		for bundleID, profile := range group.BundleIDProfileMap() {
			log.Debugf(" - %s: %s (%s)", bundleID, profile.Name, profile.UUID)
		}
	}

	if len(iosCodeSignGroups) < 1 {
		log.Errorf("Failed to find Codesign Groups")
		return nil, nil
	}

	if len(iosCodeSignGroups) > 1 {
		log.Warnf("Multiple code signing groups found! Using the first code signing group")
	}

	return &iosCodeSignGroups[0], nil
}

// addXcode9Properties adds new exportOption properties introduced in Xcode 9.
func addXcode9Properties(exportOpts exportoptions.ExportOptions, teamID, codesignIdentity, signingStyle string, bundleIDProfileMap map[string]string, xcodeManaged bool) exportoptions.ExportOptions {
	switch exportOpts.(type) {
	case exportoptions.AppStoreOptionsModel:
		options, ok := exportOpts.(exportoptions.AppStoreOptionsModel)
		if !ok {
			// will be ok because of the type switch
		}

		options.BundleIDProvisioningProfileMapping = bundleIDProfileMap
		options.SigningCertificate = codesignIdentity
		options.TeamID = teamID

		if xcodeManaged && signingStyle == "manual" {
			log.Warnf("App was signed with xcode managed profile when archiving,")
			log.Warnf("ipa export uses manual code signing.")
			log.Warnf(`Setting "signingStyle" to "manual"`)

			options.SigningStyle = "manual"
		}
		return options
	case exportoptions.NonAppStoreOptionsModel:
		options, ok := exportOpts.(exportoptions.NonAppStoreOptionsModel)
		if !ok {
			// will be ok because of the type switch
		}

		options.BundleIDProvisioningProfileMapping = bundleIDProfileMap
		options.SigningCertificate = codesignIdentity
		options.TeamID = teamID

		if xcodeManaged && signingStyle == "manual" {
			log.Warnf("App was signed with xcode managed profile when archiving,")
			log.Warnf("ipa export uses manual code signing.")
			log.Warnf(`Setting "signingStyle" to "manual"`)

			options.SigningStyle = "manual"
		}
		return options
	}
	return nil
}

func addXcode12Properties(exportOpts exportoptions.ExportOptions, distributionBundleIdentifier string) exportoptions.ExportOptions {
	switch exportOpts.(type) {
	case exportoptions.AppStoreOptionsModel:
		// Export option plist with App store export method (Xcode 12.0.1) do not contain distribution bundle identifier.
		// Propably due to App store IPAs containing App Clips also, which are executable targets with a seperate bundle ID.
		return exportOpts
	case exportoptions.NonAppStoreOptionsModel:
		options, ok := exportOpts.(exportoptions.NonAppStoreOptionsModel)
		if !ok {
			// will be ok because of the type switch
		}
		options.DistributionBundleIdentifier = distributionBundleIdentifier
		return options
	}
	return nil
}

// generateExportOptions generates an exportOptions based on the provided conditions.
func (g ExportOptionsGenerator) generateExportOptions(exportMethod exportoptions.Method, containerEnvironment string, teamID string, uploadBitcode bool, compileBitcode bool, xcodeManaged bool,
	bundleIDEntitlementsMap map[string]plistutil.PlistData, xcodeMajorVersion int64, distributionBundleIdentifier string) (exportoptions.ExportOptions, error) {
	iCloudContainerEnvironment, err := determineIcloudContainerEnvironment(containerEnvironment, bundleIDEntitlementsMap, exportMethod, xcodeMajorVersion)
	if err != nil {
		return nil, err
	}

	exportOpts := generateBaseExportOptions(exportMethod, uploadBitcode, compileBitcode, iCloudContainerEnvironment)

	if xcodeMajorVersion < 9 {
		return exportOpts, nil
	}

	codeSignGroup, err := g.determineCodesignGroup(bundleIDEntitlementsMap, exportMethod, teamID, xcodeManaged)
	if err != nil {
		return nil, err
	}
	if codeSignGroup == nil {
		return exportOpts, nil
	}

	exportCodeSignStyle := ""
	exportProfileMapping := map[string]string{}
	for bundleID, profileInfo := range codeSignGroup.BundleIDProfileMap() {
		exportProfileMapping[bundleID] = profileInfo.Name

		isXcodeManaged := profileutil.IsXcodeManaged(profileInfo.Name)
		if isXcodeManaged {
			if exportCodeSignStyle != "" && exportCodeSignStyle != "automatic" {
				log.Errorf("Both xcode managed and NON xcode managed profiles in code signing group")
			}
			exportCodeSignStyle = "automatic"
		} else {
			if exportCodeSignStyle != "" && exportCodeSignStyle != "manual" {
				log.Errorf("Both xcode managed and NON xcode managed profiles in code signing group")
			}
			exportCodeSignStyle = "manual"
		}
	}

	exportOpts = addXcode9Properties(exportOpts, codeSignGroup.Certificate().TeamID, codeSignGroup.Certificate().CommonName, exportCodeSignStyle, exportProfileMapping, xcodeManaged)

	if xcodeMajorVersion >= 12 {
		exportOpts = addXcode12Properties(exportOpts, distributionBundleIdentifier)
	}

	return exportOpts, nil
}
