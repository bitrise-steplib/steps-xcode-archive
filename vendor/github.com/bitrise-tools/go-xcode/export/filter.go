package export

import (
	"github.com/bitrise-tools/go-xcode/exportoptions"
	"github.com/bitrise-tools/go-xcode/profileutil"
)

// FilterSelectableCodeSignGroupsForTeam ...
func FilterSelectableCodeSignGroupsForTeam(codeSignGroups []SelectableCodeSignGroup, teamID string) []SelectableCodeSignGroup {
	filteredGroups := []SelectableCodeSignGroup{}
	for _, group := range codeSignGroups {
		if group.Certificate.TeamID == teamID {
			filteredGroups = append(filteredGroups, group)
		}
	}
	return filteredGroups
}

// FilterSelectableCodeSignGroupsForExportMethod ...
func FilterSelectableCodeSignGroupsForExportMethod(codeSignGroups []SelectableCodeSignGroup, exportMethod exportoptions.Method) []SelectableCodeSignGroup {
	filteredGroups := []SelectableCodeSignGroup{}
	for _, group := range codeSignGroups {

		bundleIDProfilesMap := map[string][]profileutil.ProvisioningProfileInfoModel{}
		for bundleID, profiles := range group.BundleIDProfilesMap {
			matchingProfiles := []profileutil.ProvisioningProfileInfoModel{}
			for _, profile := range profiles {
				if profile.ExportType == exportMethod {
					matchingProfiles = append(matchingProfiles, profile)
				}
			}
			if len(matchingProfiles) > 0 {
				bundleIDProfilesMap[bundleID] = profiles
			}
		}

		if len(bundleIDProfilesMap) == len(group.BundleIDProfilesMap) {
			filteredGroups = append(filteredGroups, group)
		}
	}
	return filteredGroups
}

// FilterSelectableCodeSignGroupsForNotXcodeManagedProfiles ...
func FilterSelectableCodeSignGroupsForNotXcodeManagedProfiles(codeSignGroups []SelectableCodeSignGroup) []SelectableCodeSignGroup {
	filteredGroups := []SelectableCodeSignGroup{}
	for _, group := range codeSignGroups {

		bundleIDNotManagedProfilesMap := map[string][]profileutil.ProvisioningProfileInfoModel{}
		for bundleID, profiles := range group.BundleIDProfilesMap {
			notManagedProfiles := []profileutil.ProvisioningProfileInfoModel{}
			for _, profile := range profiles {
				if !profile.IsXcodeManaged() {
					notManagedProfiles = append(notManagedProfiles, profile)
				}
			}
			if len(notManagedProfiles) > 0 {
				bundleIDNotManagedProfilesMap[bundleID] = profiles
			}
		}

		if len(bundleIDNotManagedProfilesMap) == len(group.BundleIDProfilesMap) {
			filteredGroups = append(filteredGroups, group)
		}
	}
	return filteredGroups
}

// FilterCodeSignGroupsForTeam ...
func FilterCodeSignGroupsForTeam(codeSignGroups []CodeSignGroup, teamID string) []CodeSignGroup {
	filteredGroups := []CodeSignGroup{}
	for _, group := range codeSignGroups {
		if group.Certificate.TeamID == teamID {
			filteredGroups = append(filteredGroups, group)
		}
	}
	return filteredGroups
}

// FilterCodeSignGroupsForExportMethod ...
func FilterCodeSignGroupsForExportMethod(codeSignGroups []CodeSignGroup, exportMethod exportoptions.Method) []CodeSignGroup {
	filteredGroups := []CodeSignGroup{}
	for _, group := range codeSignGroups {
		matchingGroup := true
		for _, profile := range group.BundleIDProfileMap {
			if profile.ExportType != exportMethod {
				matchingGroup = false
				break
			}
		}
		if matchingGroup {
			filteredGroups = append(filteredGroups, group)
		}
	}
	return filteredGroups
}

// FilterCodeSignGroupsForNotXcodeManagedProfiles ...
func FilterCodeSignGroupsForNotXcodeManagedProfiles(codeSignGroups []CodeSignGroup) []CodeSignGroup {
	filteredGroups := []CodeSignGroup{}
	for _, group := range codeSignGroups {
		xcodeManagedGroup := false
		for _, profile := range group.BundleIDProfileMap {
			if profile.IsXcodeManaged() {
				xcodeManagedGroup = true
				break
			}
		}
		if !xcodeManagedGroup {
			filteredGroups = append(filteredGroups, group)
		}
	}
	return filteredGroups
}
