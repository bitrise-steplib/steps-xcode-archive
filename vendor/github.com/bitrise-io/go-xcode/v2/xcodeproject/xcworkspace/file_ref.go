package xcworkspace

import (
	"fmt"
	"strings"
)

// FileRef is a file the workspace references, such as a project.
type FileRef struct {
	Location string `xml:"location,attr"`
}

// locationType says what a file reference or group location's path is relative to.
type locationType string

const (
	// absoluteLocation is an absolute path, or a relative one that xcodebuild takes relative to
	// the workspace's directory.
	absoluteLocation locationType = "absolute"
	// groupLocation is relative to the enclosing group, or to the workspace's directory at the
	// top level.
	groupLocation locationType = "group"
	// containerLocation is relative to the workspace's directory, whichever group it is in.
	containerLocation locationType = "container"
	// developerLocation is relative to the Xcode Developer directory, where no project a build
	// targets lives.
	developerLocation locationType = "developer"
	// selfLocation is the project an embedded project.xcworkspace belongs to.
	selfLocation locationType = "self"
)

// parseLocation splits a location, such as "group:App.xcodeproj", into its type and path.
func parseLocation(location string) (locationType, string, error) {
	refType, pth, found := strings.Cut(location, ":")
	if !found {
		return "", "", fmt.Errorf("unknown file reference location (%s)", location)
	}

	switch locationType(refType) {
	case absoluteLocation, groupLocation, containerLocation, developerLocation, selfLocation:
		return locationType(refType), pth, nil
	default:
		return "", "", fmt.Errorf("unknown file reference type: %s, value: %s", refType, pth)
	}
}
