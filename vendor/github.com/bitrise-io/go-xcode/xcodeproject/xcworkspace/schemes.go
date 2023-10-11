package xcworkspace

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-xcode/xcodeproject/serialized"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcodeproj"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcscheme"
	"golang.org/x/text/unicode/norm"
	"howett.net/plist"
)

// Schemes returns the schemes considered by Xcode, when opening the given workspace.
// The considered schemes are the workspace shared schemes, the workspace user schemes (for the current user)
// and the embedded project's schemes (XcodeProj.SchemesWithAutocreateEnabled).
func (w Workspace) Schemes() (map[string][]xcscheme.Scheme, error) {
	log.TDebugf("Searching schemes in workspace: %s", w.Path)

	schemesByContainer := map[string][]xcscheme.Scheme{}

	sharedSchemes, err := w.sharedSchemes()
	if err != nil {
		return nil, fmt.Errorf("failed to read shared schemes: %w", err)
	}

	userSchemes, err := w.userSchemes()
	if err != nil {
		return nil, fmt.Errorf("failed to read user schemes: %w", err)
	}

	workspaceSchemes := append(sharedSchemes, userSchemes...)

	log.TDebugf("%d scheme(s) found", len(workspaceSchemes))
	if len(workspaceSchemes) > 0 {
		schemesByContainer[w.Path] = workspaceSchemes
	}

	// project schemes
	projectLocations, err := w.ProjectFileLocations()
	if err != nil {
		return nil, fmt.Errorf("failed to get project locations from workspace: %w", err)
	}

	isAutocreateSchemesEnabled, err := w.isAutocreateSchemesEnabled()
	if err != nil {
		return nil, fmt.Errorf("failed to read the workspace autocreate scheme option: %w", err)
	}

	for _, projectLocation := range projectLocations {
		if exist, err := pathutil.IsPathExists(projectLocation); err != nil {
			return nil, fmt.Errorf("failed to check if project (%s) exists: %w", projectLocation, err)
		} else if !exist {
			// at this point we are interested the schemes visible for the workspace
			continue
		}

		project, err := xcodeproj.Open(projectLocation)
		if err != nil {
			return nil, fmt.Errorf("failed to open project (%s): %w", projectLocation, err)
		}

		projectSchemes, err := project.SchemesWithAutocreateEnabled(isAutocreateSchemesEnabled)
		if err != nil {
			return nil, fmt.Errorf("failed to read project (%s) schemes: %w", projectLocation, err)
		}

		if len(projectSchemes) > 0 {
			schemesByContainer[project.Path] = projectSchemes
		}
	}

	return schemesByContainer, nil
}

// Scheme returns the scheme by name, and it's container's absolute path.
func (w Workspace) Scheme(name string) (*xcscheme.Scheme, string, error) {
	schemesByContainer, err := w.Schemes()
	if err != nil {
		return nil, "", err
	}

	normName := norm.NFC.String(name)
	for container, schemes := range schemesByContainer {
		for _, scheme := range schemes {
			if norm.NFC.String(scheme.Name) == normName {
				return &scheme, container, nil
			}
		}
	}

	return nil, "", xcscheme.NotFoundError{Scheme: name, Container: w.Name}
}

func (w Workspace) sharedSchemes() ([]xcscheme.Scheme, error) {
	sharedSchemeFilePaths, err := w.sharedSchemeFilePaths()
	if err != nil {
		return nil, err
	}

	var sharedSchemes []xcscheme.Scheme
	for _, pth := range sharedSchemeFilePaths {
		scheme, err := xcscheme.Open(pth)
		if err != nil {
			return nil, err
		}

		sharedSchemes = append(sharedSchemes, scheme)
	}

	return sharedSchemes, nil
}

func (w Workspace) sharedSchemeFilePaths() ([]string, error) {
	// <workspace_name>.xcworkspace/xcshareddata/xcschemes/<scheme_name>.xcscheme
	sharedSchemesDir := filepath.Join(w.Path, "xcshareddata", "xcschemes")
	return listSchemeFilePaths(sharedSchemesDir)
}

func (w Workspace) userSchemes() ([]xcscheme.Scheme, error) {
	userSchemeFilePaths, err := w.userSchemeFilePaths()
	if err != nil {
		return nil, err
	}

	var userSchemes []xcscheme.Scheme
	for _, pth := range userSchemeFilePaths {
		scheme, err := xcscheme.Open(pth)
		if err != nil {
			return nil, err
		}

		userSchemes = append(userSchemes, scheme)
	}

	return userSchemes, nil
}

func (w Workspace) userSchemeFilePaths() ([]string, error) {
	// <workspace_name>.xcworkspace/xcuserdata/<current_user>.xcuserdatad/xcschemes/<scheme_name>.xcscheme
	userSchemesDir, err := w.userSchemesDir()
	if err != nil {
		return nil, err
	}
	return listSchemeFilePaths(userSchemesDir)
}

func (w Workspace) userSchemesDir() (string, error) {
	// <workspace_name>.xcworkspace/xcuserdata/<current_user>.xcuserdatad/xcschemes/
	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}

	username := currentUser.Username

	return filepath.Join(w.Path, "xcuserdata", username+".xcuserdatad", "xcschemes"), nil
}

func (w Workspace) isAutocreateSchemesEnabled() (bool, error) {
	// <workspace_name>.xcworkspace/xcshareddata/WorkspaceSettings.xcsettings
	shareddataDir := filepath.Join(w.Path, "xcshareddata")
	workspaceSettingsPth := filepath.Join(shareddataDir, "WorkspaceSettings.xcsettings")

	workspaceSettingsContent, err := os.ReadFile(workspaceSettingsPth)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// By default 'Autocreate Schemes' is enabled
			return true, nil
		}

		return false, err
	}

	var settings serialized.Object
	if _, err := plist.Unmarshal(workspaceSettingsContent, &settings); err != nil {
		return false, fmt.Errorf("failed to unmarshall settings: %w", err)
	}

	autoCreate, err := settings.Bool("IDEWorkspaceSharedSettings_AutocreateContextsIfNeeded")
	if err != nil {
		if serialized.IsKeyNotFoundError(err) {
			// By default 'Autocreate Schemes' is enabled
			return true, nil
		}
		return false, err
	}

	return autoCreate, nil
}

func listSchemeFilePaths(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var schemeFilePaths []string
	for _, entry := range entries {
		baseName := entry.Name()
		if filepath.Ext(baseName) == ".xcscheme" {
			schemeFilePaths = append(schemeFilePaths, filepath.Join(dir, baseName))
		}
	}

	return schemeFilePaths, nil
}
