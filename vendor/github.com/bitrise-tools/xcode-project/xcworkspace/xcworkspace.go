package xcworkspace

import (
	"encoding/xml"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-tools/xcode-project/xcodeproj"
	"github.com/bitrise-tools/xcode-project/xcscheme"
)

// Workspace ...
type Workspace struct {
	FileRefs []FileRef `xml:"FileRef"`
	Groups   []Group   `xml:"Group"`

	Name string
	Path string
}

// Scheme ...
func (w Workspace) Scheme(name string) (xcscheme.Scheme, string, bool) {
	schemesByContainer, err := w.Schemes()
	if err != nil {
		return xcscheme.Scheme{}, "", false
	}

	for container, schemes := range schemesByContainer {
		for _, scheme := range schemes {
			if scheme.Name == name {
				return scheme, container, true
			}
		}
	}

	return xcscheme.Scheme{}, "", false
}

// Schemes ...
func (w Workspace) Schemes() (map[string][]xcscheme.Scheme, error) {
	schemesByContainer := map[string][]xcscheme.Scheme{}

	pattern := filepath.Join(w.Path, "xcshareddata", "xcschemes", "*.xcscheme")
	pths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	// workspace schemes
	var workspaceSchemes []xcscheme.Scheme
	for _, pth := range pths {
		scheme, err := xcscheme.Open(pth)
		if err != nil {
			return nil, err
		}
		workspaceSchemes = append(workspaceSchemes, scheme)
	}

	schemesByContainer[w.Path] = workspaceSchemes

	// project schemes
	projectLocations, err := w.ProjectFileLocations()
	if err != nil {
		return nil, err
	}

	for _, projectLocation := range projectLocations {
		project, err := xcodeproj.Open(projectLocation)
		if err != nil {
			return nil, err
		}

		projectSchemes, err := project.Schemes()
		if err != nil {
			return nil, err
		}

		schemesByContainer[project.Path] = projectSchemes
	}

	return schemesByContainer, nil
}

// FileLocations ...
func (w Workspace) FileLocations() ([]string, error) {
	var fileLocations []string

	for _, fileRef := range w.FileRefs {
		pth, err := fileRef.AbsPath(filepath.Dir(w.Path))
		if err != nil {
			return nil, err
		}

		fileLocations = append(fileLocations, pth)
	}

	for _, group := range w.Groups {
		groupFileLocations, err := group.FileLocations(filepath.Dir(w.Path))
		if err != nil {
			return nil, err
		}

		fileLocations = append(fileLocations, groupFileLocations...)
	}

	return fileLocations, nil
}

// ProjectFileLocations ...
func (w Workspace) ProjectFileLocations() ([]string, error) {
	var projectLocations []string
	fileLocations, err := w.FileLocations()
	if err != nil {
		return nil, err
	}
	for _, fileLocation := range fileLocations {
		if xcodeproj.IsXcodeProj(fileLocation) {
			projectLocations = append(projectLocations, fileLocation)
		}
	}
	return projectLocations, nil
}

// Open ...
func Open(pth string) (Workspace, error) {
	contentsPth := filepath.Join(pth, "contents.xcworkspacedata")
	b, err := fileutil.ReadBytesFromFile(contentsPth)
	if err != nil {
		return Workspace{}, err
	}

	var workspace Workspace
	if err := xml.Unmarshal(b, &workspace); err != nil {
		return Workspace{}, err
	}

	workspace.Name = strings.TrimSuffix(filepath.Base(pth), filepath.Ext(pth))
	workspace.Path = pth

	return workspace, nil
}

// IsWorkspace ...
func IsWorkspace(pth string) bool {
	return filepath.Ext(pth) == ".xcworkspace"
}
