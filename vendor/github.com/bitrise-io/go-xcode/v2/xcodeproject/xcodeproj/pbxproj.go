package xcodeproj

import (
	"bytes"

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

// Parse builds a project from project.pbxproj content. projectPath is where the project is
// considered to live: relative build setting paths resolve against it and Save writes to it.
// content is copied, so the caller may reuse it.
func (f Factory) Parse(content []byte, projectPath string) (*PBXProj, error) {
	// Why clone: Save() rebuilds the file from these bytes, so a buffer the caller changes later would corrupt it.
	return f.newPBXProj(bytes.Clone(content), projectPath)
}

func (f Factory) newPBXProj(content []byte, projectPath string) (*PBXProj, error) {
	return parsePBXProj(content, f.newBaseProj(projectPath))
}

// TargetDevelopmentTeam implements XcodeProj. It reads the target's DevelopmentTeam from
// TargetAttributes.
func (p *PBXProj) TargetDevelopmentTeam(targetID string) (string, bool) {
	if p.targetAttributes == nil {
		return "", false
	}

	attributes, ok := p.targetAttributes.Object(targetID)
	if !ok {
		return "", false
	}

	developmentTeam, ok := attributes.String("DevelopmentTeam")
	if !ok || developmentTeam == "" {
		return "", false
	}

	return developmentTeam, true
}
