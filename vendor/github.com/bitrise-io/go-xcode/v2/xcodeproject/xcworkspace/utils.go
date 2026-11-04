package xcworkspace

import "path/filepath"

// IsWorkspace reports whether the path has the .xcworkspace extension.
func IsWorkspace(pth string) bool {
	return filepath.Ext(pth) == XCWorkspaceExtension
}
