package xcodeproject

import (
	"fmt"

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

// Open opens the project or workspace at pth, decided by its extension.
func (f Factory) Open(pth string) (Container, error) {
	switch {
	case xcodeproj.IsXcodeProj(pth):
		project, err := f.project.Open(pth)
		if err != nil {
			return nil, err
		}
		return &container{project: project, projectFactory: f.project}, nil
	case xcworkspace.IsWorkspace(pth):
		workspace, err := f.workspace.Open(pth)
		if err != nil {
			return nil, err
		}
		return &container{workspace: workspace, projectFactory: f.project}, nil
	default:
		return nil, fmt.Errorf("%s is neither an Xcode project (%s) nor a workspace (%s)", pth, xcodeproj.XcodeProjExtension, xcworkspace.XCWorkspaceExtension)
	}
}

// Scheme returns the named scheme of the project or workspace at pth. It is Open followed by
// Container.Scheme.
func (f Factory) Scheme(pth, name string) (xcscheme.Scheme, error) {
	container, err := f.Open(pth)
	if err != nil {
		return xcscheme.Scheme{}, err
	}

	return container.Scheme(name)
}
