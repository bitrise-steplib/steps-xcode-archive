package xcodeproj

import (
	"github.com/bitrise-io/go-plist"
	"github.com/bitrise-io/go-xcode/v2/xcodeproject/serialized"
)

const (
	customAnnotationKey = plist.CustomAnnotationKey
	startKey            = plist.CustomAnnotationStartKey
	endKey              = plist.CustomAnnotationEndKey
)

// PBXProj is a project whose project file is project.pbxproj, the property list format.
type PBXProj struct {
	baseProj

	// rawProj is the source of truth Save writes out; every edit lands here.
	rawProj serialized.Object
	format  int
	// originalContents is what Save diffs against to rewrite only the changed objects.
	originalContents []byte

	projectID        string
	targetAttributes serialized.Object
	// buildPhaseIDs maps target IDs to the IDs of their build phases.
	buildPhaseIDs map[string][]string
}

func (f Factory) newPBXProj(content []byte, projectPath string) (*PBXProj, error) {
	return parsePBXProj(content, f.newBaseProj(projectPath))
}
