package exportoptionsgenerator

import (
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/certificateutil"
	"github.com/bitrise-io/go-xcode/export"
	"github.com/bitrise-io/go-xcode/exportoptions"
	"github.com/bitrise-io/go-xcode/profileutil"
	"github.com/bitrise-io/go-xcode/v2/plistutil"
)

// CodeSignGroupProvider ...
type CodeSignGroupProvider interface {
	DetermineCodesignGroup(certificates []certificateutil.CertificateInfoModel, profiles []profileutil.ProvisioningProfileInfoModel, defaultProfile *profileutil.ProvisioningProfileInfoModel, bundleIDEntitlementsMap map[string]plistutil.PlistData, exportMethod exportoptions.Method, teamID string, xcodeManaged bool) (*export.IosCodeSignGroup, error)
}

type codeSignGroupProvider struct {
	logger log.Logger
}

// NewCodeSignGroupProvider ...
func NewCodeSignGroupProvider(logger log.Logger) CodeSignGroupProvider {
	return &codeSignGroupProvider{logger: logger}
}

// DetermineCodesignGroup ....
func (g codeSignGroupProvider) DetermineCodesignGroup(certificates []certificateutil.CertificateInfoModel, profiles []profileutil.ProvisioningProfileInfoModel, defaultProfile *profileutil.ProvisioningProfileInfoModel, bundleIDEntitlementsMap map[string]plistutil.PlistData, exportMethod exportoptions.Method, teamID string, xcodeManaged bool) (*export.IosCodeSignGroup, error) {
	g.logger.Println()
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

	g.logger.Println()
	g.logger.Printf("Resolving CodeSignGroups...")

	g.logger.Debugf("Installed certificates:")
	for _, certInfo := range certificates {
		g.logger.Debugf(certInfo.String())
	}

	g.logger.Debugf("Installed profiles:")
	for _, profileInfo := range profiles {
		g.logger.Debugf(profileInfo.String(certificates...))
	}

	g.logger.Printf("Resolving CodeSignGroups...")
	codeSignGroups := export.CreateSelectableCodeSignGroups(certificates, profiles, bundleIDs)
	if len(codeSignGroups) == 0 {
		g.logger.Errorf("Failed to find code signing groups for specified export method (%s)", exportMethod)
	}

	g.logger.Debugf("\nGroups:")
	for _, group := range codeSignGroups {
		g.logger.Debugf(group.String())
	}

	if len(bundleIDEntitlementsMap) > 0 {
		g.logger.Warnf("Filtering CodeSignInfo groups for target capabilities")

		codeSignGroups = export.FilterSelectableCodeSignGroups(codeSignGroups, export.CreateEntitlementsSelectableCodeSignGroupFilter(convertToV1PlistData(bundleIDEntitlementsMap)))

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

	if teamID == "" && defaultProfile != nil {
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
