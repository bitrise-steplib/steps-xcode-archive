package xcworkspace

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-io/go-xcode/v2/plistutil"
	"github.com/bitrise-io/go-xcode/v2/xcodeproject/serialized"
	"github.com/bitrise-io/go-xcode/v2/xcodeproject/xcodeproj"
	"github.com/bitrise-io/go-xcode/v2/xcodeproject/xcscheme"
)

const (
	// XCWorkspaceExtension ...
	XCWorkspaceExtension = ".xcworkspace"
)

const contentsFileName = "contents.xcworkspacedata"

// BuildSettingsProvider resolves the effective build settings of a scheme.
type BuildSettingsProvider interface {
	SchemeBuildSettings(workspacePath, scheme, configuration string, extraArgs ...string) (serialized.Object, error)
}

// Factory opens Xcode workspaces with a fixed set of dependencies.
type Factory struct {
	logger           log.Logger
	buildSettings    BuildSettingsProvider
	fileManager      fileutil.FileManager
	plistFileHandler plistutil.FileHandler
	pathModifier     pathutil.PathModifier
	pathChecker      pathutil.PathChecker
	userProvider     xcodeproj.UserProvider
	projectFactory   xcodeproj.Factory
	schemeFactory    xcscheme.Factory
}

// NewFactory returns a Factory that injects the given collaborators into every workspace it opens.
// projectFactory opens the projects a workspace references.
func NewFactory(
	logger log.Logger,
	buildSettings BuildSettingsProvider,
	fileManager fileutil.FileManager,
	plistFileHandler plistutil.FileHandler,
	pathModifier pathutil.PathModifier,
	pathChecker pathutil.PathChecker,
	userProvider xcodeproj.UserProvider,
	projectFactory xcodeproj.Factory,
) Factory {
	return Factory{
		logger:           logger,
		buildSettings:    buildSettings,
		fileManager:      fileManager,
		plistFileHandler: plistFileHandler,
		pathModifier:     pathModifier,
		pathChecker:      pathChecker,
		userProvider:     userProvider,
		projectFactory:   projectFactory,
		schemeFactory:    xcscheme.NewFactory(fileManager),
	}
}

// Workspace represents an Xcode workspace
type Workspace struct {
	FileRefs []FileRef `xml:"FileRef"`
	Groups   []Group   `xml:"Group"`

	Name string
	Path string

	logger           log.Logger
	buildSettings    BuildSettingsProvider
	fileManager      fileutil.FileManager
	plistFileHandler plistutil.FileHandler
	pathModifier     pathutil.PathModifier
	pathChecker      pathutil.PathChecker
	userProvider     xcodeproj.UserProvider
	projectFactory   xcodeproj.Factory
	schemeFactory    xcscheme.Factory
}

// Open reads the workspace at pth.
func (f Factory) Open(pth string) (*Workspace, error) {
	absPth, err := f.pathModifier.AbsPath(pth)
	if err != nil {
		return nil, fmt.Errorf("failed to expand path (%s): %w", pth, err)
	}

	contentsPth := filepath.Join(absPth, contentsFileName)

	file, err := f.fileManager.Open(contentsPth)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", contentsPth, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			f.logger.Warnf("Failed to close %s: %s", contentsPth, closeErr)
		}
	}()

	var workspace Workspace
	if err := xml.NewDecoder(file).Decode(&workspace); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s: %w", contentsPth, err)
	}

	workspace.Name = strings.TrimSuffix(filepath.Base(absPth), filepath.Ext(absPth))
	workspace.Path = absPth
	workspace.logger = f.logger
	workspace.buildSettings = f.buildSettings
	workspace.fileManager = f.fileManager
	workspace.plistFileHandler = f.plistFileHandler
	workspace.pathModifier = f.pathModifier
	workspace.pathChecker = f.pathChecker
	workspace.userProvider = f.userProvider
	workspace.projectFactory = f.projectFactory
	workspace.schemeFactory = f.schemeFactory

	return &workspace, nil
}

// SchemeBuildSettings returns the scheme's effective build settings, resolved by xcodebuild within
// the workspace.
func (w *Workspace) SchemeBuildSettings(scheme, configuration string, customOptions ...string) (serialized.Object, error) {
	return w.buildSettings.SchemeBuildSettings(w.Path, scheme, configuration, customOptions...)
}

// fileLocations returns the absolute paths of the files the workspace references: the top-level
// ones first, then each group's, where a group's own files precede those of its subgroups. A
// developer: location is left out, as it points into the Xcode Developer directory; a self: one
// resolves to the project an embedded project.xcworkspace belongs to and is left out elsewhere.
func (w *Workspace) fileLocations() ([]string, error) {
	return w.collectFileLocations(w.FileRefs, w.Groups, filepath.Dir(w.Path))
}

func (w *Workspace) collectFileLocations(fileRefs []FileRef, groups []Group, groupDir string) ([]string, error) {
	var locations []string

	for _, fileRef := range fileRefs {
		pth, ok, err := w.absPath(fileRef.Location, groupDir)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}

		locations = append(locations, pth)
	}

	for _, group := range groups {
		subgroupDir, ok, err := w.absPath(group.Location, groupDir)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}

		groupLocations, err := w.collectFileLocations(group.FileRefs, group.Groups, subgroupDir)
		if err != nil {
			return nil, err
		}

		locations = append(locations, groupLocations...)
	}

	return locations, nil
}

// absPath resolves a file reference or group location found in the group at groupDir. ok is false
// for a location the file list leaves out: a developer: one, which points into the Xcode Developer
// directory, where no project a build targets lives, or a self: one in a workspace that is not
// embedded in a project, where xcodebuild ignores it too.
func (w *Workspace) absPath(location, groupDir string) (pth string, ok bool, err error) {
	refType, pth, err := parseLocation(location)
	if err != nil {
		return "", false, err
	}

	switch refType {
	case groupLocation:
		pth = filepath.Join(groupDir, pth)
	case containerLocation, absoluteLocation:
		// An absolute: location can hold a relative path, which xcodebuild resolves against the
		// workspace directory, like a container: one.
		if !filepath.IsAbs(pth) {
			pth = filepath.Join(filepath.Dir(w.Path), pth)
		}
	case selfLocation:
		project, embedded := w.embeddingProject()
		if !embedded {
			w.logger.TDebugf("Skipping self: location of a workspace not embedded in a project: %s", location)
			return "", false, nil
		}
		pth = project
	case developerLocation:
		w.logger.TDebugf("Skipping location in the Developer directory: %s", location)
		return "", false, nil
	}

	pth, err = w.pathModifier.AbsPath(pth)
	if err != nil {
		return "", false, err
	}

	return pth, true, nil
}

// embeddingProject returns the project the workspace belongs to when it is the project's own
// <name>.xcodeproj/project.xcworkspace, which Xcode writes with a single self: reference.
func (w *Workspace) embeddingProject() (string, bool) {
	parent := filepath.Dir(w.Path)
	if !xcodeproj.IsXcodeProj(parent) {
		return "", false
	}

	return parent, true
}

// ProjectFileLocations returns the absolute paths of the projects the workspace references, in the
// order they appear in its file tree: top-level references first, then each group's.
func (w *Workspace) ProjectFileLocations() ([]string, error) {
	var projectLocations []string
	fileLocations, err := w.fileLocations()
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

// Projects opens the projects the workspace references, in ProjectFileLocations order. A referenced
// project missing from disk is skipped, as xcodebuild -list skips it and its schemes.
func (w *Workspace) Projects() ([]*xcodeproj.XcodeProj, error) {
	locations, err := w.ProjectFileLocations()
	if err != nil {
		return nil, fmt.Errorf("failed to get project locations from workspace: %w", err)
	}

	var projects []*xcodeproj.XcodeProj
	for _, location := range locations {
		project, ok, err := w.openProject(location)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}

		projects = append(projects, project)
	}

	return projects, nil
}

// openProject opens the referenced project at location. ok is false when the project is missing from
// disk, which xcodebuild -list skips too, but which usually means a generated project was not
// generated, so it is worth a warning.
func (w *Workspace) openProject(location string) (project *xcodeproj.XcodeProj, ok bool, err error) {
	exists, err := w.pathChecker.IsPathExists(location)
	if err != nil {
		return nil, false, fmt.Errorf("failed to check if project (%s) exists: %w", location, err)
	}
	if !exists {
		w.logger.Warnf("The workspace references a project that is missing, so its schemes are not available: %s. Generate it before this step (for CocoaPods, run pod install) or fix the reference in contents.xcworkspacedata.", location)
		return nil, false, nil
	}

	project, err = w.projectFactory.Open(location)
	if err != nil {
		return nil, false, fmt.Errorf("failed to open project (%s): %w", location, err)
	}

	return project, true, nil
}
