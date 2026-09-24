package codesigngroup

import (
	"github.com/bitrise-io/go-xcode/exportoptions"
	"github.com/bitrise-io/go-xcode/v2/plistutil"
	"github.com/bitrise-io/go-xcode/v2/profileutil"
)

// SelectableCodeSignGroupFilter ...
type SelectableCodeSignGroupFilter func(group *SelectableCodeSignGroup) bool

// Filter ...
func Filter(groups []SelectableCodeSignGroup, filterFunc SelectableCodeSignGroupFilter) []SelectableCodeSignGroup {
	if filterFunc == nil {
		return groups
	}

	var filteredGroups []SelectableCodeSignGroup
	for _, group := range groups {
		if filterFunc(&group) {
			filteredGroups = append(filteredGroups, group)
		}
	}

	return filteredGroups
}

// CreateEntitlementsSelectableCodeSignGroupFilter ...
func CreateEntitlementsSelectableCodeSignGroupFilter(bundleIDEntitlementsMap map[string]plistutil.PlistData) SelectableCodeSignGroupFilter {
	return func(group *SelectableCodeSignGroup) bool {
		filteredBundleIDProfilesMap := map[string][]profileutil.ProvisioningProfileInfoModel{}

		for bundleID, profiles := range group.BundleIDProfilesMap {
			var filteredProfiles []profileutil.ProvisioningProfileInfoModel

			for _, profile := range profiles {
				missingEntitlements := profileutil.MatchTargetAndProfileEntitlements(bundleIDEntitlementsMap[bundleID], profile.Entitlements, profile.Type)
				if len(missingEntitlements) == 0 {
					filteredProfiles = append(filteredProfiles, profile)
				}
			}

			if len(filteredProfiles) == 0 {
				break
			}

			filteredBundleIDProfilesMap[bundleID] = filteredProfiles
		}

		if len(filteredBundleIDProfilesMap) == len(group.BundleIDProfilesMap) {
			group.BundleIDProfilesMap = filteredBundleIDProfilesMap
			return true
		}

		return false
	}
}

// CreateExportMethodSelectableCodeSignGroupFilter ...
func CreateExportMethodSelectableCodeSignGroupFilter(exportMethod exportoptions.Method) SelectableCodeSignGroupFilter {
	return func(group *SelectableCodeSignGroup) bool {
		filteredBundleIDProfilesMap := map[string][]profileutil.ProvisioningProfileInfoModel{}

		for bundleID, profiles := range group.BundleIDProfilesMap {
			var filteredProfiles []profileutil.ProvisioningProfileInfoModel

			for _, profile := range profiles {
				if profile.ExportType == exportMethod {
					filteredProfiles = append(filteredProfiles, profile)
				}
			}

			if len(filteredProfiles) == 0 {
				break
			}

			filteredBundleIDProfilesMap[bundleID] = filteredProfiles
		}

		if len(filteredBundleIDProfilesMap) == len(group.BundleIDProfilesMap) {
			group.BundleIDProfilesMap = filteredBundleIDProfilesMap
			return true
		}

		return false
	}
}

// CreateTeamSelectableCodeSignGroupFilter ...
func CreateTeamSelectableCodeSignGroupFilter(teamID string) SelectableCodeSignGroupFilter {
	return func(group *SelectableCodeSignGroup) bool {
		return group.Certificate.TeamID == teamID
	}
}

// CreateNotXcodeManagedSelectableCodeSignGroupFilter ...
func CreateNotXcodeManagedSelectableCodeSignGroupFilter() SelectableCodeSignGroupFilter {
	return func(group *SelectableCodeSignGroup) bool {
		filteredBundleIDProfilesMap := map[string][]profileutil.ProvisioningProfileInfoModel{}

		for bundleID, profiles := range group.BundleIDProfilesMap {
			var filteredProfiles []profileutil.ProvisioningProfileInfoModel

			for _, profile := range profiles {
				if !profile.IsXcodeManaged() {
					filteredProfiles = append(filteredProfiles, profile)
				}
			}

			if len(filteredProfiles) == 0 {
				break
			}

			filteredBundleIDProfilesMap[bundleID] = filteredProfiles
		}

		if len(filteredBundleIDProfilesMap) == len(group.BundleIDProfilesMap) {
			group.BundleIDProfilesMap = filteredBundleIDProfilesMap
			return true
		}

		return false
	}
}

// CreateXcodeManagedSelectableCodeSignGroupFilter ...
func CreateXcodeManagedSelectableCodeSignGroupFilter() SelectableCodeSignGroupFilter {
	return func(group *SelectableCodeSignGroup) bool {
		filteredBundleIDProfilesMap := map[string][]profileutil.ProvisioningProfileInfoModel{}

		for bundleID, profiles := range group.BundleIDProfilesMap {
			var filteredProfiles []profileutil.ProvisioningProfileInfoModel

			for _, profile := range profiles {
				if profile.IsXcodeManaged() {
					filteredProfiles = append(filteredProfiles, profile)
				}
			}

			if len(filteredProfiles) == 0 {
				break
			}

			filteredBundleIDProfilesMap[bundleID] = filteredProfiles
		}

		if len(filteredBundleIDProfilesMap) == len(group.BundleIDProfilesMap) {
			group.BundleIDProfilesMap = filteredBundleIDProfilesMap
			return true
		}

		return false
	}
}

// CreateExcludeProfileNameSelectableCodeSignGroupFilter ...
func CreateExcludeProfileNameSelectableCodeSignGroupFilter(name string) SelectableCodeSignGroupFilter {
	return func(group *SelectableCodeSignGroup) bool {
		filteredBundleIDProfilesMap := map[string][]profileutil.ProvisioningProfileInfoModel{}

		for bundleID, profiles := range group.BundleIDProfilesMap {
			var filteredProfiles []profileutil.ProvisioningProfileInfoModel

			for _, profile := range profiles {
				if profile.Name != name {
					filteredProfiles = append(filteredProfiles, profile)
				}
			}

			if len(filteredProfiles) == 0 {
				break
			}

			filteredBundleIDProfilesMap[bundleID] = filteredProfiles
		}

		if len(filteredBundleIDProfilesMap) == len(group.BundleIDProfilesMap) {
			group.BundleIDProfilesMap = filteredBundleIDProfilesMap
			return true
		}

		return false
	}
}
