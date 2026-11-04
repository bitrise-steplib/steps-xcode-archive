package xcodeproject

import (
	"fmt"
	"path/filepath"

	"github.com/bitrise-io/go-xcode/v2/xcodeproject/xcodeproj"
	"github.com/bitrise-io/go-xcode/v2/xcodeproject/xcscheme"
	"github.com/bitrise-io/go-xcode/v2/xcodeproject/xcworkspace"
)

// Factory opens projects and workspaces with the factories it was given.
type Factory struct {
	project   xcodeproj.Factory
	workspace xcworkspace.Factory
}

// NewFactory returns a Factory that opens projects with project and workspaces with workspace.
func NewFactory(project xcodeproj.Factory, workspace xcworkspace.Factory) Factory {
	return Factory{
		project:   project,
		workspace: workspace,
	}
}

// Open opens the project or workspace at pth, decided by its extension. A trailing path separator
// is ignored, as xcodebuild ignores it.
func (f Factory) Open(pth string) (Container, error) {
	switch cleanPth := filepath.Clean(pth); {
	case xcodeproj.IsXcodeProj(cleanPth):
		project, err := f.project.Open(pth)
		if err != nil {
			return nil, err
		}
		return &container{schemes: project}, nil
	case xcworkspace.IsWorkspace(cleanPth):
		workspace, err := f.workspace.Open(pth)
		if err != nil {
			return nil, err
		}
		return &container{schemes: workspace}, nil
	default:
		return nil, fmt.Errorf("%s is neither an Xcode project nor a workspace: use the path of the %s or %s bundle", pth, xcodeproj.XcodeProjExtension, xcworkspace.XCWorkspaceExtension)
	}
}

// Scheme returns the project or workspace scheme by name, with the path of the bundle holding it.
// It is Open followed by Container.Scheme.
func (f Factory) Scheme(pth string, name string) (xcscheme.Scheme, string, error) {
	c, err := f.Open(pth)
	if err != nil {
		return xcscheme.Scheme{}, "", err
	}

	return c.Scheme(name)
}
