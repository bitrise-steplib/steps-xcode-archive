package xcworkspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/text/unicode/norm"

	"github.com/bitrise-io/go-xcode/v2/xcodeproject/serialized"
	"github.com/bitrise-io/go-xcode/v2/xcodeproject/xcscheme"
)

const (
	xcschemeExtension           = ".xcscheme"
	autocreateSchemesSettingKey = "IDEWorkspaceSharedSettings_AutocreateContextsIfNeeded"
)

// containerSchemes are the schemes one container, the workspace or a project, holds.
type containerSchemes struct {
	path    string
	schemes []xcscheme.Scheme
}

// Schemes returns the schemes Xcode considers when opening the workspace, keyed by the path of the
// container that holds them: the workspace's own shared and user schemes under the workspace path,
// and each referenced project's schemes under that project's path. A project without schemes gets
// Xcode's default ones when the workspace's "Autocreate schemes" option is on.
func (w *Workspace) Schemes() (map[string][]xcscheme.Scheme, error) {
	schemesByContainer := map[string][]xcscheme.Scheme{}
	err := w.visitSchemeContainers(func(container containerSchemes) bool {
		schemesByContainer[container.path] = container.schemes
		return false
	})
	if err != nil {
		return nil, err
	}

	return schemesByContainer, nil
}

// Scheme returns the scheme with the given name and the path of the container that holds it. The
// workspace's own schemes are searched first, then each project's in ProjectFileLocations order,
// and the search stops at the first container holding the name, so the projects after it are not
// opened. Names are compared after Unicode normalisation.
func (w *Workspace) Scheme(name string) (xcscheme.Scheme, string, error) {
	normalisedName := norm.NFC.String(name)

	var found xcscheme.Scheme
	var containerPath string
	err := w.visitSchemeContainers(func(container containerSchemes) bool {
		for _, scheme := range container.schemes {
			if norm.NFC.String(scheme.Name) == normalisedName {
				found, containerPath = scheme, container.path
				return true
			}
		}
		return false
	})
	if err != nil {
		return xcscheme.Scheme{}, "", err
	}
	if containerPath == "" {
		return xcscheme.Scheme{}, "", xcscheme.NotFoundError{Scheme: name, Container: w.Name}
	}

	return found, containerPath, nil
}

// visitSchemeContainers calls visit with each container holding at least one scheme, the workspace
// first and then each referenced project in ProjectFileLocations order, until visit returns true.
// Projects are opened one at a time, so a visit that stops early leaves the rest unread.
func (w *Workspace) visitSchemeContainers(visit func(containerSchemes) bool) error {
	w.logger.TDebugf("Searching schemes in workspace: %s", w.Path)

	sharedSchemes, err := w.sharedSchemes()
	if err != nil {
		return fmt.Errorf("failed to read shared schemes: %w", err)
	}

	userSchemes, err := w.userSchemes()
	if err != nil {
		return fmt.Errorf("failed to read user schemes: %w", err)
	}

	workspaceSchemes := append(sharedSchemes, userSchemes...)

	w.logger.TDebugf("%d scheme(s) found", len(workspaceSchemes))
	if len(workspaceSchemes) > 0 && visit(containerSchemes{path: w.Path, schemes: workspaceSchemes}) {
		return nil
	}

	locations, err := w.ProjectFileLocations()
	if err != nil {
		return fmt.Errorf("failed to get project locations from workspace: %w", err)
	}

	// Read lazily: only a project without schemes of its own depends on this file.
	var autocreate *bool
	for _, location := range locations {
		project, ok, err := w.openProject(location)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}

		if autocreate == nil {
			enabled := w.isAutocreateSchemesEnabled()
			autocreate = &enabled
		}

		schemes, err := project.SchemesWithAutocreateOverride(*autocreate)
		if err != nil {
			return fmt.Errorf("failed to read project (%s) schemes: %w", project.Path(), err)
		}

		if len(schemes) > 0 && visit(containerSchemes{path: project.Path(), schemes: schemes}) {
			return nil
		}
	}

	return nil
}

func (w *Workspace) sharedSchemes() ([]xcscheme.Scheme, error) {
	// <workspace_name>.xcworkspace/xcshareddata/xcschemes/<scheme_name>.xcscheme
	schemes, err := w.readSchemes(filepath.Join(w.Path, "xcshareddata", "xcschemes"))
	if err != nil {
		return nil, err
	}

	for i := range schemes {
		schemes[i].IsShared = true
	}

	return schemes, nil
}

func (w *Workspace) userSchemes() ([]xcscheme.Scheme, error) {
	// <workspace_name>.xcworkspace/xcuserdata/<current_user>.xcuserdatad/xcschemes/<scheme_name>.xcscheme
	username, err := w.userProvider.CurrentUserName()
	if err != nil {
		return nil, fmt.Errorf("failed to get the current user: %w", err)
	}

	return w.readSchemes(filepath.Join(w.Path, "xcuserdata", username+".xcuserdatad", "xcschemes"))
}

// readSchemes returns the schemes in dir, in directory order. A missing dir holds none.
func (w *Workspace) readSchemes(dir string) ([]xcscheme.Scheme, error) {
	names, err := w.fileManager.ReadDirEntryNames(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var schemes []xcscheme.Scheme
	for _, name := range names {
		if filepath.Ext(name) != xcschemeExtension {
			continue
		}

		scheme, err := w.schemeFactory.Open(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}

		schemes = append(schemes, scheme)
	}

	return schemes, nil
}

// isAutocreateSchemesEnabled defaults to true when the settings file or the key is missing. Like
// xcodebuild, which warns about a settings file it cannot read and lists the schemes anyway, it
// also falls back to the default when the file cannot be read or the key is not a boolean.
func (w *Workspace) isAutocreateSchemesEnabled() bool {
	// <workspace_name>.xcworkspace/xcshareddata/WorkspaceSettings.xcsettings
	pth := filepath.Join(w.Path, "xcshareddata", "WorkspaceSettings.xcsettings")

	data, _, err := w.plistFileHandler.Read(pth)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			w.logger.Warnf("Failed to read the workspace's 'Autocreate schemes' setting, using Xcode's default (on): %s. Fix or remove %s.", err, pth)
		}
		// By default 'Autocreate Schemes' is enabled
		return true
	}
	settings := serialized.Object(data)

	if !settings.Has(autocreateSchemesSettingKey) {
		// By default 'Autocreate Schemes' is enabled
		return true
	}

	autocreate, ok := settings.Bool(autocreateSchemesSettingKey)
	if !ok {
		w.logger.Warnf("The workspace's 'Autocreate schemes' setting (%s) is not a boolean, using Xcode's default (on). Set it to true or false in %s.", autocreateSchemesSettingKey, pth)
		return true
	}

	return autocreate
}
