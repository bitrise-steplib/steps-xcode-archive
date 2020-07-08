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
	"github.com/bitrise-steplib/steps-xcode-archive/utils"
)

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

func determineCodesignGroup(bundleIDEntitlementsMap map[string]plistutil.PlistData, exportMethod exportoptions.Method, teamID string, xcodeManaged bool) (*export.IosCodeSignGroup, error) {
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

	certs, err := certificateutil.InstalledCodesigningCertificateInfos()
	if err != nil {
		return nil, fmt.Errorf("Failed to get installed certificates, error: %s", err)
	}
	certInfo := certificateutil.FilterValidCertificateInfos(certs)
	certs = append(certInfo.ValidCertificates, certInfo.DuplicatedCertificates...)

	log.Debugf("Installed certificates:")
	for _, certInfo := range certs {
		log.Debugf(certInfo.String())
	}

	profs, err := profileutil.InstalledProvisioningProfileInfos(profileutil.ProfileTypeIos)
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
	}
	return exportOpts
}

func generateExportOptions(exportMethod exportoptions.Method, containerEnvironment string, teamID string, uploadBitcode bool, compileBitcode bool, xcodeManaged bool,
	bundleIDEntitlementsMap map[string]plistutil.PlistData, xcodeMajorVersion int64, exportOptionsPath string) error {
	iCloudContainerEnvironment, err := determineIcloudContainerEnvironment(containerEnvironment, bundleIDEntitlementsMap, exportMethod, xcodeMajorVersion)
	if err != nil {
		return err
	}

	exportOpts := generateBaseExportOptions(exportMethod, uploadBitcode, compileBitcode, iCloudContainerEnvironment)

	if xcodeMajorVersion < 9 {
		fmt.Println()
		log.Printf("generated export options content:")
		fmt.Println()
		fmt.Println(exportOpts.String())

		if err = exportOpts.WriteToFile(exportOptionsPath); err != nil {
			return fmt.Errorf("Failed to write export options to file, error: %s", err)
		}
		return nil
	}

	codeSignGroup, err := determineCodesignGroup(bundleIDEntitlementsMap, exportMethod, teamID, xcodeManaged)
	if err != nil {
		return err
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

	fmt.Println()
	log.Printf("generated export options content:")
	fmt.Println()
	fmt.Println(exportOpts.String())

	if err = exportOpts.WriteToFile(exportOptionsPath); err != nil {
		return fmt.Errorf("Failed to write export options to file, error: %s", err)
	}
	return nil
}
