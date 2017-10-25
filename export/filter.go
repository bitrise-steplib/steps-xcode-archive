package export

import (
	"fmt"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-tools/go-xcode/profileutil"
)

// FilterCodeSignGroupsForTeam ...
func FilterCodeSignGroupsForTeam(codeSignGroups []CodeSignGroupItem, teamID string) []CodeSignGroupItem {
	filteredGroups := []CodeSignGroupItem{}
	for _, group := range codeSignGroups {
		if group.Certificate.TeamID == teamID {
			filteredGroups = append(filteredGroups, group)
		} else {
			log.Warnf("removing CodeSignGroup: %s", group.Certificate.CommonName)
			fmt.Println()
		}
	}
	return filteredGroups
}

// FilterCodeSignGroupsForNotXcodeManagedProfiles ...
func FilterCodeSignGroupsForNotXcodeManagedProfiles(codeSignGroups []CodeSignGroupItem) []CodeSignGroupItem {
	filteredGroups := []CodeSignGroupItem{}
	for _, group := range codeSignGroups {
		xcodeManagedGroup := false
		for _, profile := range group.BundleIDProfileMap {
			isXcodeManaged := profileutil.IsXcodeManaged(profile.Name)
			if isXcodeManaged {
				xcodeManagedGroup = true
				break
			}
		}
		if !xcodeManagedGroup {
			filteredGroups = append(filteredGroups, group)
		} else {
			log.Warnf("removing CodeSignGroup: %s", group.Certificate.CommonName)
		}
	}
	return filteredGroups
}
