package projectmanager

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/v2/autocodesign"
	"github.com/bitrise-io/go-xcode/xcodeproject/schemeint"
	"github.com/bitrise-io/go-xcode/xcodeproject/serialized"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcodeproj"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcscheme"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcworkspace"
	"howett.net/plist"
)

type buildSettings struct {
	settings serialized.Object
	basePath string
}

// ProjectHelper ...
type ProjectHelper struct {
	Logger                    log.Logger
	MainTarget                xcodeproj.Target
	DependentTargets          []xcodeproj.Target
	UITestTargets             []xcodeproj.Target
	XcWorkspace               *xcworkspace.Workspace // nil if working with standalone project
	XcProj                    xcodeproj.XcodeProj
	Configuration             string
	IsDebugProjectBasedLookup bool

	buildSettingsCache map[string]map[string][]buildSettings // target/config/buildSettings
}

// NewProjectHelper checks the provided project or workspace and generate a ProjectHelper with the provided scheme and configuration
// Previously in the ruby version the initialize method did the same
// It returns a new ProjectHelper, whose Configuration field contains is the selected configuration (even when configurationName parameter is empty)
func NewProjectHelper(projOrWSPath string, logger log.Logger, schemeName, configurationName string, isDebug bool) (*ProjectHelper, error) {
	if exits, err := pathutil.IsPathExists(projOrWSPath); err != nil {
		return nil, err
	} else if !exits {
		return nil, fmt.Errorf("provided path does not exists: %s", projOrWSPath)
	}

	// Get the project and scheme of the provided .xcodeproj or .xcworkspace
	// It is important to keep the returned scheme, as it can be located under the .xcworkspace and not the .xcodeproj.
	// Fetching the scheme from the project based on name is not possible later.
	xcproj, scheme, err := findBuiltProject(logger, projOrWSPath, schemeName)
	if err != nil {
		return nil, fmt.Errorf("failed to find build project: %s", err)
	}

	mainTarget, err := mainTargetOfScheme(logger, xcproj, scheme)
	if err != nil {
		return nil, fmt.Errorf("failed to find the main target of the scheme (%s): %s", schemeName, err)
	}

	var dependentTargets []xcodeproj.Target
	for _, target := range xcproj.DependentTargetsOfTarget(mainTarget) {
		if target.IsExecutableProduct() {
			dependentTargets = append(dependentTargets, target)
		}
	}

	var uiTestTargets []xcodeproj.Target
	for _, target := range xcproj.Proj.Targets {
		if target.IsUITestProduct() && target.DependsOn(mainTarget.ID) {
			uiTestTargets = append(uiTestTargets, target)
		}
	}

	conf, err := configuration(logger, configurationName, scheme, xcproj)
	if err != nil {
		return nil, err
	}
	if conf == "" {
		return nil, fmt.Errorf("no configuration provided nor default defined for the scheme's (%s) archive action", schemeName)
	}

	var workspace *xcworkspace.Workspace
	if filepath.Ext(projOrWSPath) == ".xcworkspace" {
		ws, err := xcworkspace.Open(projOrWSPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open workspace: %s", err)
		}
		workspace = &ws
	}

	return &ProjectHelper{
		Logger:                    logger,
		MainTarget:                mainTarget,
		DependentTargets:          dependentTargets,
		UITestTargets:             uiTestTargets,
		XcWorkspace:               workspace,
		XcProj:                    xcproj,
		Configuration:             conf,
		IsDebugProjectBasedLookup: isDebug,
	}, nil
}

// ArchivableTargets ...
func (p *ProjectHelper) ArchivableTargets() []xcodeproj.Target {
	return append([]xcodeproj.Target{p.MainTarget}, p.DependentTargets...)
}

// ArchivableTargetBundleIDToEntitlements ...
func (p *ProjectHelper) ArchivableTargetBundleIDToEntitlements() (map[string]autocodesign.Entitlements, error) {
	entitlementsByBundleID := map[string]autocodesign.Entitlements{}

	for _, target := range p.ArchivableTargets() {
		bundleID, err := p.TargetBundleID(target.Name, p.Configuration)
		if err != nil {
			return nil, fmt.Errorf("failed to get target (%s) bundle id: %s", target.Name, err)
		}

		entitlements, err := p.targetEntitlements(target.Name, p.Configuration, bundleID)
		if err != nil && !serialized.IsKeyNotFoundError(err) {
			return nil, fmt.Errorf("failed to get target (%s) bundle id: %s", target.Name, err)
		}

		entitlementsByBundleID[bundleID] = entitlements
	}

	return entitlementsByBundleID, nil
}

// UITestTargetBundleIDs ...
func (p *ProjectHelper) UITestTargetBundleIDs() ([]string, error) {
	var bundleIDs []string

	for _, target := range p.UITestTargets {
		bundleID, err := p.TargetBundleID(target.Name, p.Configuration)
		if err != nil {
			return nil, fmt.Errorf("failed to get target (%s) bundle id: %s", target.Name, err)
		}

		bundleIDs = append(bundleIDs, bundleID)
	}

	return bundleIDs, nil
}

// Platform get the platform (PLATFORM_DISPLAY_NAME) - iOS, tvOS, macOS
func (p *ProjectHelper) Platform(configurationName string) (autocodesign.Platform, error) {
	platformDisplayName, err := p.buildSettingForKey(p.MainTarget.Name, configurationName, "PLATFORM_DISPLAY_NAME")
	if err != nil {
		return "", fmt.Errorf("no PLATFORM_DISPLAY_NAME config found for (%s) target", p.MainTarget.Name)
	}

	if platformDisplayName != string(autocodesign.IOS) && platformDisplayName != string(autocodesign.MacOS) && platformDisplayName != string(autocodesign.TVOS) {
		return "", fmt.Errorf("not supported platform. Platform (PLATFORM_DISPLAY_NAME) = %s, supported: %s, %s", platformDisplayName, autocodesign.IOS, autocodesign.TVOS)
	}
	return autocodesign.Platform(platformDisplayName), nil
}

// ProjectTeamID returns the development team's ID
// If there is multiple development team in the project (different team for targets) it will return an error
// It returns the development team's ID
func (p *ProjectHelper) ProjectTeamID(config string) (string, error) {
	var teamID string

	for _, target := range p.XcProj.Proj.Targets {
		currentTeamID, err := p.targetTeamID(target.Name, config)
		if err != nil {
			p.Logger.Debugf("buildSettings: %", err)
		} else {
			p.Logger.Debugf("buildSettings: Target (%s) build settings/DEVELOPMENT_TEAM Team ID: %s", target.Name, currentTeamID)
		}

		if currentTeamID == "" {
			targetAttributes, err := p.XcProj.Proj.Attributes.TargetAttributes.Object(target.ID)
			if err != nil {
				// Skip projects not using target attributes
				if serialized.IsKeyNotFoundError(err) {
					p.Logger.Debugf("buildSettings: Target (%s) does not have TargetAttributes: No Team ID found.", target.Name)
					continue
				}

				return "", fmt.Errorf("failed to parse target (%s) attributes: %s", target.ID, err)
			}

			targetAttributesTeamID, err := targetAttributes.String("DevelopmentTeam")
			if err != nil && !serialized.IsKeyNotFoundError(err) {
				return "", fmt.Errorf("failed to parse development team for target (%s): %s", target.ID, err)
			}

			p.Logger.Debugf("buildSettings: Target (%s) DevelopmentTeam attribute: %s", target.Name, targetAttributesTeamID)

			if targetAttributesTeamID == "" {
				p.Logger.Debugf("buildSettings: Target (%s): No Team ID found.", target.Name)
				continue
			}

			currentTeamID = targetAttributesTeamID
		}

		if teamID == "" {
			teamID = currentTeamID
			continue
		}

		if teamID != currentTeamID {
			p.Logger.Warnf("buildSettings: Target (%s) Team ID (%s) does not match to the already registered team ID: %s\nThis causes build issue like: `Embedded binary is not signed with the same certificate as the parent app. Verify the embedded binary target's code sign settings match the parent app's.`", target.Name, currentTeamID, teamID)
			teamID = ""
			break
		}
	}

	return teamID, nil
}

func (p *ProjectHelper) targetTeamID(targetName, config string) (string, error) {
	devTeam, err := p.buildSettingForKey(targetName, config, "DEVELOPMENT_TEAM")
	if serialized.IsKeyNotFoundError(err) {
		p.Logger.Debugf("buildSettings: Target (%s) does not have DEVELOPMENT_TEAM in build settings", targetName)
		return "", nil
	}
	return devTeam, err
}

func (p *ProjectHelper) fetchBuildSettings(targetName, conf string, customOptions ...string) ([]buildSettings, error) {
	if p.XcWorkspace == nil {
		settings, err := p.XcProj.TargetBuildSettings(targetName, conf, customOptions...)
		return []buildSettings{{settings: settings, basePath: p.XcProj.Path}}, err
	}

	settings, err := p.XcWorkspace.SchemeBuildSettings(targetName, conf, customOptions...)
	if err == nil {
		wsSettings := buildSettings{settings: settings, basePath: p.XcWorkspace.Path}
		if !p.IsDebugProjectBasedLookup {
			return []buildSettings{wsSettings}, nil
		}

		// In debug mode, also fetch project build settings to compare values
		projectSettings, err := p.XcProj.TargetBuildSettings(targetName, conf, customOptions...)
		if err != nil {
			p.Logger.Errorf("buildSettings: Failed to fetch build settings from project for target (%s): %s", targetName, err)
		}

		return []buildSettings{wsSettings, {settings: projectSettings, basePath: p.XcProj.Path}}, nil
	}

	p.Logger.Warnf("buildSettings: Failed to fetch build settings from workspace for target (%s): %s", targetName, err)
	p.Logger.Printf("buildSettings: Falling back to project build settings")
	settings, err = p.XcProj.TargetBuildSettings(targetName, conf, customOptions...)
	return []buildSettings{{settings: settings, basePath: p.XcProj.Path}}, err
}

func (p *ProjectHelper) cachedBuildSettings(targetName, conf string, customOptions ...string) ([]buildSettings, error) {
	targetCache, ok := p.buildSettingsCache[targetName]
	if ok {
		confCache, ok := targetCache[conf]
		if ok {
			p.Logger.Debugf("buildSettings: Using cached settings for target='%s'", targetName)
			return confCache, nil
		}
	}

	settingsList, err := p.fetchBuildSettings(targetName, conf, customOptions...)
	if err != nil {
		return settingsList, err
	}

	if targetCache == nil {
		targetCache = map[string][]buildSettings{}
	}
	targetCache[conf] = settingsList

	if p.buildSettingsCache == nil {
		p.buildSettingsCache = map[string]map[string][]buildSettings{}
	}
	p.buildSettingsCache[targetName] = targetCache

	return settingsList, nil
}

func (p *ProjectHelper) targetBuildSettings(targetName, conf string) (serialized.Object, error) {
	settingsList, err := p.cachedBuildSettings(targetName, conf)
	if err != nil {
		var basePath string
		if len(settingsList) > 0 {
			basePath = settingsList[0].basePath
		}
		return nil, fmt.Errorf("failed to fetch target (%s) build settings for project (%s): %s", targetName, basePath, err)
	}

	if len(settingsList) == 1 {
		return settingsList[0].settings, nil
	}

	p.Logger.Debugf("buildSettings: Workspace target build settings: %+v", settingsList[0])
	p.Logger.Debugf("buildSettings: Project target build settings: %+v", settingsList[1])
	p.Logger.Debugf("buildSettings: Multiple build settings found for target (%s), returning the project one", targetName)
	return settingsList[1].settings, nil
}

func (p *ProjectHelper) buildSettingForKey(targetName, conf string, key string, customOptions ...string) (string, error) {
	settingsList, err := p.cachedBuildSettings(targetName, conf, customOptions...)
	if err != nil {
		var basePath string
		if len(settingsList) > 0 {
			basePath = settingsList[0].basePath
		}
		return "", fmt.Errorf("failed to fetch target (%s) build settings for project (%s): %s", targetName, basePath, err)
	}

	wsSettings := settingsList[0].settings
	wsValue, err := wsSettings.String(key)
	if err != nil {
		return wsValue, err
	}

	if len(settingsList) == 1 {
		return wsValue, nil
	}

	projectSettings := settingsList[1].settings
	projectValue, err := projectSettings.String(key)
	if err != nil {
		p.Logger.Errorf("buildSettings: Failed to fetch project build setting for key (%s): %s", key, err)
		p.Logger.Printf("buildSettings: Returning workspace value for key (%s): %s", key, wsValue)
		return wsValue, nil
	}

	if projectValue != wsValue {
		p.Logger.Errorf("buildSettings: Conflicting values for build setting %s: '%s' (workspace) vs '%s' (project)", key, wsValue, projectValue)
		// Return alternate value to be consistent with old project based target build setting fetch
		p.Logger.Printf("buildSettings: Returning project value for key (%s): %s", key, projectValue)
		return projectValue, nil
	}
	p.Logger.Debugf("buildSettings: Matching values for workspace and project build setting %s: '%s'", key, wsValue)

	return wsValue, err
}

func (p *ProjectHelper) buildSettingPathForKey(targetName, conf string, key string, customOptions ...string) (string, error) {
	settingsList, err := p.cachedBuildSettings(targetName, conf, customOptions...)
	if err != nil {
		var basePath string
		if len(settingsList) > 0 {
			basePath = settingsList[0].basePath
		}
		return "", fmt.Errorf("failed to fetch target (%s) build settings for project (%s): %s", targetName, basePath, err)
	}

	wsSettings := settingsList[0]
	wsValue, err := wsSettings.settings.String(key)
	if err != nil {
		return wsValue, err
	}

	if pathutil.IsRelativePath(wsValue) {
		wsValue = filepath.Join(path.Dir(wsSettings.basePath), wsValue)
	}

	if len(settingsList) == 1 {
		return wsValue, nil
	}

	projectSettings := settingsList[1]
	projectValue, err := projectSettings.settings.String(key)
	if err != nil {
		return projectValue, err
	}

	if pathutil.IsRelativePath(projectValue) {
		projectValue = filepath.Join(path.Dir(projectSettings.basePath), projectValue)
	}

	if projectValue != wsValue {
		p.Logger.Errorf("buildSettings: Conflicting paths for build setting %s: '%s' (workspace) vs '%s' (project)", key, wsValue, projectValue)
		p.Logger.Printf("buildSettings: Returning project path for key (%s): %s", key, projectValue)
		return projectValue, nil
	}

	p.Logger.Debugf("buildSettings: Matching paths for workspace and project build setting %s: '%s'", key, wsValue)
	return wsValue, nil
}

// TargetBundleID returns the target bundle ID
// First it tries to fetch the bundle ID from the `PRODUCT_BUNDLE_IDENTIFIER` build settings
// If it's no available it will fetch the target's Info.plist and search for the `CFBundleIdentifier` key.
// The CFBundleIdentifier's value is not resolved in the Info.plist, so it will try to resolve it by the resolveBundleID()
// It returns  the target bundle ID
func (p *ProjectHelper) TargetBundleID(name, conf string) (string, error) {
	bundleID, err := p.buildSettingForKey(name, conf, "PRODUCT_BUNDLE_IDENTIFIER")
	if err != nil && !serialized.IsKeyNotFoundError(err) {
		return "", fmt.Errorf("failed to parse target (%s) build settings attribute PRODUCT_BUNDLE_IDENTIFIER: %s", name, err)
	}
	if bundleID != "" {
		return bundleID, nil
	}

	p.Logger.Debugf("buildSettings: PRODUCT_BUNDLE_IDENTIFIER env not found in 'xcodebuild -showBuildSettings -project %s -target %s -configuration %s command's output, checking the Info.plist file's CFBundleIdentifier property...", p.XcProj.Path, name, conf)

	infoPlistAbsPath, err := p.buildSettingPathForKey(name, conf, "INFOPLIST_FILE")
	if err != nil {
		return "", fmt.Errorf("failed to fetch Info.plist path from target (%s) build settings: %s", name, err)
	}

	if infoPlistAbsPath == "" {
		return "", fmt.Errorf("failed to to determine bundle id: xcodebuild -showBuildSettings does not contains PRODUCT_BUNDLE_IDENTIFIER nor INFOPLIST_FILE' unless info_plist_path")
	}

	b, err := fileutil.ReadBytesFromFile(infoPlistAbsPath)
	if err != nil {
		return "", fmt.Errorf("failed to read Info.plist: %s", err)
	}

	var options map[string]interface{}
	if _, err := plist.Unmarshal(b, &options); err != nil {
		return "", fmt.Errorf("failed to unmarshal Info.plist: %s ", err)
	}

	bundleID, ok := options["CFBundleIdentifier"].(string)
	if !ok || bundleID == "" {
		return "", fmt.Errorf("failed to parse CFBundleIdentifier from the Info.plist")
	}

	if !strings.Contains(bundleID, "$") {
		return bundleID, nil
	}

	p.Logger.Debugf("buildSettings: CFBundleIdentifier defined with variable: %s, trying to resolve it...", bundleID)

	settings, err := p.targetBuildSettings(name, conf)
	if err != nil {
		return "", err
	}
	resolved, err := expandTargetSetting(bundleID, settings)
	if err != nil {
		return "", fmt.Errorf("failed to resolve bundle ID: %s", err)
	}

	p.Logger.Debugf("buildSettings: resolved CFBundleIdentifier: %s", resolved)

	return resolved, nil
}

func (p *ProjectHelper) targetEntitlements(name, config, bundleID string) (autocodesign.Entitlements, error) {
	codeSignEntitlementsPth, err := p.buildSettingPathForKey(name, config, "CODE_SIGN_ENTITLEMENTS")
	if err != nil {
		if serialized.IsKeyNotFoundError(err) {
			p.Logger.Debugf("buildSettings: Target (%s) does not have CODE_SIGN_ENTITLEMENTS in build settings", name)
			return autocodesign.Entitlements{}, nil
		}
		return nil, err
	}

	entitlements, _, err := xcodeproj.ReadPlistFile(codeSignEntitlementsPth)
	if err != nil {
		return nil, err
	}

	return resolveEntitlementVariables(p.Logger, autocodesign.Entitlements(entitlements), bundleID)
}

// IsSigningManagedAutomatically checks the "Automatically manage signing" checkbox in Xcode
// Note: it only checks the main Target based on the given Scheme and Configuration
func (p *ProjectHelper) IsSigningManagedAutomatically() (bool, error) {
	targetName := p.MainTarget.Name
	codeSignStyle, err := p.buildSettingForKey(targetName, p.Configuration, "CODE_SIGN_STYLE")
	if err != nil {
		if errors.As(err, &serialized.KeyNotFoundError{}) {
			p.Logger.Debugf("setting CODE_SIGN_STYLE unspecified for target (%s), defaulting to `Manual`", targetName)

			return false, nil
		}

		return false, fmt.Errorf("failed to fetch code signing info from target (%s) settings: %s", targetName, err)
	}

	return codeSignStyle != "Manual", nil
}

// resolveEntitlementVariables expands variables in the project entitlements.
// Entitlement values can contain variables, for example: `iCloud.$(CFBundleIdentifier)`.
// Expanding iCloud Container values only, as they are compared to the profile values later.
// Expand CFBundleIdentifier variable only, other variables are not yet supported.
func resolveEntitlementVariables(logger log.Logger, entitlements autocodesign.Entitlements, bundleID string) (autocodesign.Entitlements, error) {
	containers, err := entitlements.ICloudContainers()
	if err != nil {
		return nil, err
	}

	if len(containers) == 0 {
		return entitlements, nil
	}

	var expandedContainers []interface{}
	for _, container := range containers {
		if strings.ContainsRune(container, '$') {
			expanded, err := expandTargetSetting(container, serialized.Object{"CFBundleIdentifier": bundleID})
			if err != nil {
				logger.Warnf("buildSettings: Ignoring iCloud container ID (%s) as can not expand variable: %v", container, err)
				continue
			}

			expandedContainers = append(expandedContainers, expanded)
			continue
		}

		expandedContainers = append(expandedContainers, container)
	}

	entitlements[autocodesign.ICloudIdentifiersEntitlementKey] = expandedContainers

	return entitlements, nil
}

func expandTargetSetting(value string, buildSettings serialized.Object) (string, error) {
	regexpStr := `^(.*)[$][({](.+?)([:].+)?[})](.*)$`
	r, err := regexp.Compile(regexpStr)
	if err != nil {
		return "", err
	}

	captures := r.FindStringSubmatch(value)

	if len(captures) < 5 {
		return "", fmt.Errorf("failed to match regex '%s' to %s target build setting", regexpStr, value)
	}

	prefix := captures[1]
	envKey := captures[2]
	suffix := captures[4]

	envValue, err := buildSettings.String(envKey)
	if err != nil {
		return "", fmt.Errorf("failed to find environment variable value for key %s: %s", envKey, err)
	}

	return prefix + envValue + suffix, nil
}

func configuration(logger log.Logger, configurationName string, scheme xcscheme.Scheme, xcproj xcodeproj.XcodeProj) (string, error) {
	defaultConfiguration := scheme.ArchiveAction.BuildConfiguration
	var configuration string
	if configurationName == "" || configurationName == defaultConfiguration {
		configuration = defaultConfiguration
	} else if configurationName != defaultConfiguration {
		for _, target := range xcproj.Proj.Targets {
			var configNames []string
			for _, conf := range target.BuildConfigurationList.BuildConfigurations {
				configNames = append(configNames, conf.Name)
			}
			if !sliceutil.IsStringInSlice(configurationName, configNames) {
				return "", fmt.Errorf("build configuration (%s) not defined for target: (%s)", configurationName, target.Name)
			}
		}
		logger.Warnf("buildSettings: Using user defined build configuration: %s instead of the scheme's default one: %s.\nMake sure you use the same configuration in further steps.", configurationName, defaultConfiguration)
		configuration = configurationName
	}

	return configuration, nil
}

// mainTargetOfScheme return the main target
func mainTargetOfScheme(logger log.Logger, proj xcodeproj.XcodeProj, scheme xcscheme.Scheme) (xcodeproj.Target, error) {
	logger.Debugf("buildSettings: Searching %d for scheme main target: %s", len(scheme.BuildAction.BuildActionEntries), scheme.Name)

	var blueIdent string
	for _, entry := range scheme.BuildAction.BuildActionEntries {
		if entry.BuildableReference.IsAppReference() {
			blueIdent = entry.BuildableReference.BlueprintIdentifier
			break
		}
	}

	logger.Debugf("buildSettings: Searching %d targets for: %s", len(proj.Proj.Targets), blueIdent)

	// Search for the main target
	for _, t := range proj.Proj.Targets {
		if t.ID == blueIdent {
			return t, nil
		}
	}

	return xcodeproj.Target{}, fmt.Errorf("failed to find the project's main target for scheme (%v)", scheme)
}

// findBuiltProject returns the Xcode project which will be built for the provided scheme, plus the scheme.
// The scheme is returned as it could be found under the .xcworkspace, and opening based on name from the XcodeProj would fail.
func findBuiltProject(logger log.Logger, pth, schemeName string) (xcodeproj.XcodeProj, xcscheme.Scheme, error) {
	logger.TInfof("buildSettings: Locating built project for xcode project: %s, scheme: %s", pth, schemeName)

	scheme, schemeContainerDir, err := schemeint.Scheme(pth, schemeName)
	if err != nil {
		return xcodeproj.XcodeProj{}, xcscheme.Scheme{}, fmt.Errorf("could not get scheme with name %s from path %s: %w", schemeName, pth, err)
	}

	archiveEntry, archivable := scheme.AppBuildActionEntry()
	if !archivable {
		return xcodeproj.XcodeProj{}, xcscheme.Scheme{}, fmt.Errorf("archive action not defined for scheme: %s", scheme.Name)
	}

	projectPth, err := archiveEntry.BuildableReference.ReferencedContainerAbsPath(filepath.Dir(schemeContainerDir))
	if err != nil {
		return xcodeproj.XcodeProj{}, xcscheme.Scheme{}, err
	}

	xcodeProj, err := xcodeproj.Open(projectPth)
	if err != nil {
		return xcodeproj.XcodeProj{}, xcscheme.Scheme{}, err
	}

	logger.TInfof("Located built project for scheme: %s", schemeName)

	return xcodeProj, *scheme, nil
}
