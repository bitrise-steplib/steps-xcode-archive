package xcodeproj

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
	xcschemeExtension             = ".xcscheme"
	schemeManagementFileName      = "xcschememanagement.plist"
	autocreateSchemesSettingKey   = "IDEWorkspaceSharedSettings_AutocreateContextsIfNeeded"
	errNoSchemesAutocreateOffText = "no schemes found and the Xcode project's 'Autocreate schemes' option is disabled"
)

// Schemes returns the project's shared and user schemes. If there are none, it generates Xcode's
// default schemes, unless the "Autocreate schemes" option is off, in which case it returns an error.
func (p *XcodeProj) Schemes() ([]xcscheme.Scheme, error) {
	p.logger.TDebugf("Searching schemes in project: %s", p.Path)

	schemes, found, err := p.existingOrManagedSchemes()
	if err != nil || found {
		return schemes, err
	}

	// Read lazily: a project that has schemes never depends on this file.
	autocreate, err := p.isAutocreateSchemesEnabled()
	if err != nil {
		return nil, fmt.Errorf("failed to read the project autocreate scheme option: %w", err)
	}

	if autocreate {
		p.logger.TDebugf("Autocreating the default schemes")
		return p.defaultSchemes(), nil
	}

	return nil, errors.New(errNoSchemesAutocreateOffText)
}

// SchemesWithAutocreateOverride is Schemes with the "Autocreate schemes" option set by the caller,
// for use within a workspace. With no schemes and autocreate off it returns no schemes and no error.
func (p *XcodeProj) SchemesWithAutocreateOverride(autocreateEnabled bool) ([]xcscheme.Scheme, error) {
	p.logger.TDebugf("Searching schemes in project: %s", p.Path)

	schemes, found, err := p.existingOrManagedSchemes()
	if err != nil || found {
		return schemes, err
	}

	if autocreateEnabled {
		p.logger.TDebugf("Autocreating the default schemes")
		return p.defaultSchemes(), nil
	}

	p.logger.TDebugf("No schemes found")

	return nil, nil
}

// Scheme returns the scheme with the given name and the path of its project. Names are compared
// after Unicode normalisation.
func (p *XcodeProj) Scheme(name string) (xcscheme.Scheme, string, error) {
	schemes, err := p.Schemes()
	if err != nil {
		return xcscheme.Scheme{}, "", err
	}

	normalisedName := norm.NFC.String(name)
	for _, scheme := range schemes {
		if norm.NFC.String(scheme.Name) == normalisedName {
			return scheme, p.Path, nil
		}
	}

	return xcscheme.Scheme{}, "", xcscheme.NotFoundError{Scheme: name, Container: p.Name}
}

// existingOrManagedSchemes returns found=false when the caller must decide by the autocreate option.
func (p *XcodeProj) existingOrManagedSchemes() (schemes []xcscheme.Scheme, found bool, err error) {
	schemes, err = p.visibleSchemes()
	if err != nil {
		return nil, false, err
	}

	if len(schemes) > 0 {
		p.logger.TDebugf("%d scheme(s) found", len(schemes))
		return schemes, true, nil
	}

	schemeManagementExists, err := p.isUserSchemeManagementFileExist()
	if err != nil {
		return nil, false, err
	}

	if schemeManagementExists {
		p.logger.TDebugf("Default scheme found")
		return p.defaultSchemes(), true, nil
	}

	return nil, false, nil
}

func (p *XcodeProj) defaultSchemes() []xcscheme.Scheme {
	schemes := p.generateSchemes()
	for i := range schemes {
		schemes[i].IsShared = true
	}
	return schemes
}

func (p *XcodeProj) visibleSchemes() ([]xcscheme.Scheme, error) {
	shared, err := p.readSchemes(p.sharedSchemesDir())
	if err != nil {
		return nil, err
	}
	for i := range shared {
		shared[i].IsShared = true
	}

	userDir, err := p.userSchemesDir()
	if err != nil {
		return nil, err
	}

	user, err := p.readSchemes(userDir)
	if err != nil {
		return nil, err
	}

	return append(shared, user...), nil
}

func (p *XcodeProj) sharedSchemesDir() string {
	return filepath.Join(p.Path, "xcshareddata", "xcschemes")
}

func (p *XcodeProj) userSchemesDir() (string, error) {
	username, err := p.userProvider.CurrentUserName()
	if err != nil {
		return "", fmt.Errorf("failed to get the current user: %w", err)
	}

	return filepath.Join(p.Path, "xcuserdata", username+".xcuserdatad", "xcschemes"), nil
}

func (p *XcodeProj) readSchemes(dir string) ([]xcscheme.Scheme, error) {
	names, err := p.fileManager.ReadDirEntryNames(dir)
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

		scheme, err := p.schemeFactory.Open(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}

		schemes = append(schemes, scheme)
	}

	return schemes, nil
}

func (p *XcodeProj) isUserSchemeManagementFileExist() (bool, error) {
	userDir, err := p.userSchemesDir()
	if err != nil {
		return false, err
	}

	file, err := p.fileManager.Open(filepath.Join(userDir, schemeManagementFileName))
	if err != nil {
		return false, nil
	}
	_ = file.Close()

	return true, nil
}

// isAutocreateSchemesEnabled defaults to true when the settings file or the key is missing.
func (p *XcodeProj) isAutocreateSchemesEnabled() (bool, error) {
	pth := filepath.Join(p.Path, "project.xcworkspace", "xcshareddata", "WorkspaceSettings.xcsettings")

	data, _, err := p.plistFileHandler.Read(pth)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		return false, err
	}
	settings := serialized.Object(data)

	if !settings.Has(autocreateSchemesSettingKey) {
		return true, nil
	}

	autocreate, ok := settings.Bool(autocreateSchemesSettingKey)
	if !ok {
		return false, fmt.Errorf("%s is not a boolean", autocreateSchemesSettingKey)
	}

	return autocreate, nil
}
