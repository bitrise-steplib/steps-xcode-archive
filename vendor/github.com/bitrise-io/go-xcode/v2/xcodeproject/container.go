// Package xcodeproject is the common view of Xcode projects and workspaces: what a path to either
// gives a caller that does not care which of the two it holds.
package xcodeproject

import (
	"path/filepath"

	"github.com/bitrise-io/go-xcode/v2/xcodeproject/xcodeproj"
	"github.com/bitrise-io/go-xcode/v2/xcodeproject/xcscheme"
	"github.com/bitrise-io/go-xcode/v2/xcodeproject/xcworkspace"
)

// Container is an opened Xcode project or workspace. It answers for the bundle that was opened and
// for the projects that bundle references, so a caller never needs to know which of the two it has.
type Container interface {
	// Path is the absolute path of the .xcodeproj or .xcworkspace bundle that was opened.
	Path() string
	// Name is the bundle's file name without its extension.
	Name() string
	// Schemes returns the schemes Xcode considers when opening the bundle, in search order: a
	// workspace's own schemes first, then each referenced project's in file order. Each scheme
	// names the bundle holding it in ContainerPath.
	Schemes() ([]xcscheme.Scheme, error)
	// Scheme returns the named scheme, with the bundle holding it in ContainerPath.
	Scheme(name string) (xcscheme.Scheme, error)
	// Projects returns the projects the bundle references, in file order: a project is its own
	// only project, a workspace lists the projects it references, those missing from disk left
	// out. They are opened once and kept.
	Projects() ([]xcodeproj.XcodeProj, error)
	// Project returns the project at bundlePath, an absolute .xcodeproj path: one of Projects, or
	// another the bundle leads to, such as a sub-project a scheme builds, opened and kept on
	// first use.
	Project(bundlePath string) (xcodeproj.XcodeProj, error)
}

// container is the Container over either a project or a workspace, keeping the projects it opened.
type container struct {
	project        xcodeproj.XcodeProj    // the opened project, nil for a workspace
	workspace      *xcworkspace.Workspace // the opened workspace, nil for a project
	projectFactory xcodeproj.Factory

	projects       []xcodeproj.XcodeProj // Projects, once listed
	projectsListed bool
	projectsByPath map[string]xcodeproj.XcodeProj
}

func (c *container) Path() string {
	if c.workspace != nil {
		return c.workspace.Path
	}
	return c.project.Path()
}

func (c *container) Name() string {
	if c.workspace != nil {
		return c.workspace.Name
	}
	return c.project.Name()
}

func (c *container) Schemes() ([]xcscheme.Scheme, error) {
	if c.workspace != nil {
		return c.workspace.Schemes()
	}
	return c.project.Schemes()
}

func (c *container) Scheme(name string) (xcscheme.Scheme, error) {
	if c.workspace != nil {
		return c.workspace.Scheme(name)
	}
	return c.project.Scheme(name)
}

func (c *container) Projects() ([]xcodeproj.XcodeProj, error) {
	if c.projectsListed {
		return c.projects, nil
	}

	var projects []xcodeproj.XcodeProj
	if c.workspace != nil {
		var err error
		projects, err = c.workspace.Projects()
		if err != nil {
			return nil, err
		}
	} else {
		projects = []xcodeproj.XcodeProj{c.project}
	}

	for _, project := range projects {
		c.keep(project)
	}
	c.projects, c.projectsListed = projects, true

	return projects, nil
}

func (c *container) Project(bundlePath string) (xcodeproj.XcodeProj, error) {
	if _, err := c.Projects(); err != nil {
		return nil, err
	}

	if project, ok := c.projectsByPath[filepath.Clean(bundlePath)]; ok {
		return project, nil
	}

	project, err := c.projectFactory.Open(bundlePath)
	if err != nil {
		return nil, err
	}
	c.keep(project)

	return project, nil
}

func (c *container) keep(project xcodeproj.XcodeProj) {
	if c.projectsByPath == nil {
		c.projectsByPath = map[string]xcodeproj.XcodeProj{}
	}
	c.projectsByPath[filepath.Clean(project.Path())] = project
}
