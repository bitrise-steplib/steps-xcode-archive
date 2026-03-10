package codesigngroup

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/v2/log"
)

// Printer ...
type Printer struct {
	logger log.Logger
}

// NewPrinter ...
func NewPrinter(logger log.Logger) *Printer {
	return &Printer{
		logger: logger,
	}
}

// ListToDebugString ...
func (printer *Printer) ListToDebugString(groups []SelectableCodeSignGroup) string {
	var builder strings.Builder
	for _, group := range groups {
		builder.WriteString(printer.ToDebugString(group) + "\n")
	}

	return builder.String()
}

// ToDebugString ...
func (printer *Printer) ToDebugString(group SelectableCodeSignGroup) string {
	printable := map[string]any{}
	printable["team"] = fmt.Sprintf("%s (%s)", group.Certificate.TeamName, group.Certificate.TeamID)
	printable["certificate"] = fmt.Sprintf("%s (%s)", group.Certificate.CommonName, group.Certificate.Serial)

	bundleIDProfiles := map[string][]string{}
	for bundleID, profileInfos := range group.BundleIDProfilesMap {
		printableProfiles := []string{}
		for _, profileInfo := range profileInfos {
			printableProfiles = append(printableProfiles, fmt.Sprintf("%s (%s)", profileInfo.Name, profileInfo.UUID))
		}
		bundleIDProfiles[bundleID] = printableProfiles
	}
	printable["bundle_id_profiles"] = bundleIDProfiles

	data, err := json.MarshalIndent(printable, "", "\t")
	if err != nil {
		printer.logger.Errorf("Failed to marshal (%v): %s", printable, err)
		return ""
	}

	return string(data)
}
