package xcworkspace

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-xcode/xcodebuild"
	"github.com/bitrise-io/go-xcode/xcodeproject/serialized"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcodeproj"
)

const (
	// XCWorkspaceExtension ...
	XCWorkspaceExtension = ".xcworkspace"
)

// Workspace represents an Xcode workspace
type Workspace struct {
	FileRefs []FileRef `xml:"FileRef"`
	Groups   []Group   `xml:"Group"`

	Name string
	Path string
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
		return Workspace{}, fmt.Errorf("failed to unmarshal workspace file: %s, error: %s", pth, err)
	}

	workspace.Name = strings.TrimSuffix(filepath.Base(pth), filepath.Ext(pth))
	workspace.Path = pth

	return workspace, nil
}

// SchemeBuildSettings ...
func (w Workspace) SchemeBuildSettings(scheme, configuration string, customOptions ...string) (serialized.Object, error) {
	log.TDebugf("Fetching %s scheme build settings", scheme)

	commandModel := xcodebuild.NewShowBuildSettingsCommand(w.Path)
	commandModel.SetScheme(scheme)
	commandModel.SetConfiguration(configuration)
	commandModel.SetCustomOptions(customOptions)

	object, err := commandModel.RunAndReturnSettings(false)

	log.TDebugf("Fetched %s scheme build settings", scheme)

	return object, err
}

// SchemeCodeSignEntitlements returns the code sign entitlements for a scheme and configuration
func (w Workspace) SchemeCodeSignEntitlements(scheme, configuration string) (serialized.Object, error) {
	// Get build settings to find the entitlements file path
	buildSettings, err := w.SchemeBuildSettings(scheme, configuration)
	if err != nil {
		return nil, err
	}

	// Get the CODE_SIGN_ENTITLEMENTS path
	entitlementsPath, err := buildSettings.String("CODE_SIGN_ENTITLEMENTS")
	if err != nil {
		return nil, err
	}

	// Resolve the absolute path relative to workspace directory
	absolutePath := filepath.Join(filepath.Dir(w.Path), entitlementsPath)

	// Read and parse the entitlements file
	entitlements, _, err := xcodeproj.ReadPlistFile(absolutePath)
	if err != nil {
		return nil, err
	}

	log.TDebugf("Fetched %s scheme code sign entitlements", scheme)
	return entitlements, nil
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
