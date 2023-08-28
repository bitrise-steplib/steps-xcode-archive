package xcworkspace

import "path/filepath"

// IsWorkspace ...
func IsWorkspace(pth string) bool {
	return filepath.Ext(pth) == ".xcworkspace"
}
