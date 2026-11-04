// Package xcodeproject is the common view of Xcode projects and workspaces: what a path to either
// gives a caller that does not care which of the two it holds.
package xcodeproject

import (
	"github.com/bitrise-io/go-xcode/v2/xcodeproject/xcscheme"
)

// Container is an opened Xcode project or workspace, so a caller never needs to know which of the
// two it has.
type Container interface {
	// Scheme returns the named scheme and the path of the bundle holding it: for a workspace, either
	// the workspace itself or one of the projects it references.
	Scheme(name string) (xcscheme.Scheme, string, error)
}

// schemeSource looks up a scheme by name. Both xcodeproj.XcodeProj and *xcworkspace.Workspace
// implement it.
type schemeSource interface {
	Scheme(name string) (xcscheme.Scheme, string, error)
}

// container is the Container over either a project or a workspace: it keeps what both offer, so no
// method needs to know which of the two it holds.
type container struct {
	schemes schemeSource
}

// Scheme implements Container.
func (c *container) Scheme(name string) (xcscheme.Scheme, string, error) {
	return c.schemes.Scheme(name)
}
