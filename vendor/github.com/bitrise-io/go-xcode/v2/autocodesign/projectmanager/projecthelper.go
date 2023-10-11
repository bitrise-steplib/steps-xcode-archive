package projectmanager

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-io/go-xcode/v2/autocodesign"
	"github.com/bitrise-io/go-xcode/xcodeproject/schemeint"
	"github.com/bitrise-io/go-xcode/xcodeproject/serialized"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcodeproj"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcscheme"
	"howett.net/plist"
)

// ProjectHelper ...
type ProjectHelper struct {
	MainTarget       xcodeproj.Target
	DependentTargets []xcodeproj.Target
	UITestTargets    []xcodeproj.Target
	XcProj           xcodeproj.XcodeProj
	Configuration    string

	buildSettingsCache map[string]map[string]serialized.Object // target/config/buildSettings(serialized.Object)
}

// NewProjectHelper checks the provided project or workspace and generate a ProjectHelper with the provided scheme and configuration
// Previously in the ruby version the initialize method did the same
// It returns a new ProjectHelper, whose Configuration field contains is the selected configuration (even when configurationName parameter is empty)
func NewProjectHelper(projOrWSPath, schemeName, configurationName string) (*ProjectHelper, error) {
	if exits, err := pathutil.IsPathExists(projOrWSPath); err != nil {
		return nil, err
	} else if !exits {
		return nil, fmt.Errorf("provided path does not exists: %s", projOrWSPath)
	}

	// Get the project and scheme of the provided .xcodeproj or .xcworkspace
	// It is important to keep the returned scheme, as it can be located under the .xcworkspace and not the .xcodeproj.
	// Fetching the scheme from the project based on name is not possible later.
	xcproj, scheme, err := findBuiltProject(projOrWSPath, schemeName)
	if err != nil {
		return nil, fmt.Errorf("failed to find build project: %s", err)
	}

	mainTarget, err := mainTargetOfScheme(xcproj, scheme)
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

	conf, err := configuration(configurationName, scheme, xcproj)
	if err != nil {
		return nil, err
	}
	if conf == "" {
		return nil, fmt.Errorf("no configuration provided nor default defined for the scheme's (%s) archive action", schemeName)
	}

	return &ProjectHelper{
		MainTarget:       mainTarget,
		DependentTargets: dependentTargets,
		UITestTargets:    uiTestTargets,
		XcProj:           xcproj,
		Configuration:    conf,
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
	settings, err := p.targetBuildSettings(p.MainTarget.Name, configurationName)
	if err != nil {
		return "", fmt.Errorf("failed to fetch project (%s) build settings: %s", p.XcProj.Path, err)
	}

	platformDisplayName, err := settings.String("PLATFORM_DISPLAY_NAME")
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
			log.Debugf("%", err)
		} else {
			log.Debugf("Target (%s) build settings/DEVELOPMENT_TEAM Team ID: %s", target.Name, currentTeamID)
		}

		if currentTeamID == "" {
			targetAttributes, err := p.XcProj.Proj.Attributes.TargetAttributes.Object(target.ID)
			if err != nil {
				// Skip projects not using target attributes
				if serialized.IsKeyNotFoundError(err) {
					log.Debugf("Target (%s) does not have TargetAttributes: No Team ID found.", target.Name)
					continue
				}

				return "", fmt.Errorf("failed to parse target (%s) attributes: %s", target.ID, err)
			}

			targetAttributesTeamID, err := targetAttributes.String("DevelopmentTeam")
			if err != nil && !serialized.IsKeyNotFoundError(err) {
				return "", fmt.Errorf("failed to parse development team for target (%s): %s", target.ID, err)
			}

			log.Debugf("Target (%s) DevelopmentTeam attribute: %s", target.Name, targetAttributesTeamID)

			if targetAttributesTeamID == "" {
				log.Debugf("Target (%s): No Team ID found.", target.Name)
				continue
			}

			currentTeamID = targetAttributesTeamID
		}

		if teamID == "" {
			teamID = currentTeamID
			continue
		}

		if teamID != currentTeamID {
			log.Warnf("Target (%s) Team ID (%s) does not match to the already registered team ID: %s\nThis causes build issue like: `Embedded binary is not signed with the same certificate as the parent app. Verify the embedded binary target's code sign settings match the parent app's.`", target.Name, currentTeamID, teamID)
			teamID = ""
			break
		}
	}

	return teamID, nil
}

func (p *ProjectHelper) targetTeamID(targetName, config string) (string, error) {
	settings, err := p.targetBuildSettings(targetName, config)
	if err != nil {
		return "", fmt.Errorf("failed to fetch Team ID from target settings (%s): %s", targetName, err)
	}

	devTeam, err := settings.String("DEVELOPMENT_TEAM")
	if serialized.IsKeyNotFoundError(err) {
		return "", nil
	}
	return devTeam, err

}

func (p *ProjectHelper) targetBuildSettings(name, conf string) (serialized.Object, error) {
	targetCache, ok := p.buildSettingsCache[name]
	if ok {
		confCache, ok := targetCache[conf]
		if ok {
			return confCache, nil
		}
	}

	settings, err := p.XcProj.TargetBuildSettings(name, conf)
	if err != nil {
		return nil, err
	}

	if targetCache == nil {
		targetCache = map[string]serialized.Object{}
	}
	targetCache[conf] = settings

	if p.buildSettingsCache == nil {
		p.buildSettingsCache = map[string]map[string]serialized.Object{}
	}
	p.buildSettingsCache[name] = targetCache

	return settings, nil
}

// TargetBundleID returns the target bundle ID
// First it tries to fetch the bundle ID from the `PRODUCT_BUNDLE_IDENTIFIER` build settings
// If it's no available it will fetch the target's Info.plist and search for the `CFBundleIdentifier` key.
// The CFBundleIdentifier's value is not resolved in the Info.plist, so it will try to resolve it by the resolveBundleID()
// It returns  the target bundle ID
func (p *ProjectHelper) TargetBundleID(name, conf string) (string, error) {
	settings, err := p.targetBuildSettings(name, conf)
	if err != nil {
		return "", fmt.Errorf("failed to fetch target (%s) settings: %s", name, err)
	}

	bundleID, err := settings.String("PRODUCT_BUNDLE_IDENTIFIER")
	if err != nil && !serialized.IsKeyNotFoundError(err) {
		return "", fmt.Errorf("failed to parse target (%s) build settings attribute PRODUCT_BUNDLE_IDENTIFIER: %s", name, err)
	}
	if bundleID != "" {
		return bundleID, nil
	}

	log.Debugf("PRODUCT_BUNDLE_IDENTIFIER env not found in 'xcodebuild -showBuildSettings -project %s -target %s -configuration %s command's output, checking the Info.plist file's CFBundleIdentifier property...", p.XcProj.Path, name, conf)

	infoPlistPath, err := settings.String("INFOPLIST_FILE")
	if err != nil {
		return "", fmt.Errorf("failed to find Info.plist file: %s", err)
	}
	infoPlistPath = path.Join(path.Dir(p.XcProj.Path), infoPlistPath)

	if infoPlistPath == "" {
		return "", fmt.Errorf("failed to to determine bundle id: xcodebuild -showBuildSettings does not contains PRODUCT_BUNDLE_IDENTIFIER nor INFOPLIST_FILE' unless info_plist_path")
	}

	b, err := fileutil.ReadBytesFromFile(infoPlistPath)
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

	log.Debugf("CFBundleIdentifier defined with variable: %s, trying to resolve it...", bundleID)

	resolved, err := expandTargetSetting(bundleID, settings)
	if err != nil {
		return "", fmt.Errorf("failed to resolve bundle ID: %s", err)
	}

	log.Debugf("resolved CFBundleIdentifier: %s", resolved)

	return resolved, nil
}

func (p *ProjectHelper) targetEntitlements(name, config, bundleID string) (autocodesign.Entitlements, error) {
	entitlements, err := p.XcProj.TargetCodeSignEntitlements(name, config)
	if err != nil && !serialized.IsKeyNotFoundError(err) {
		return nil, err
	}

	return resolveEntitlementVariables(autocodesign.Entitlements(entitlements), bundleID)
}

// IsSigningManagedAutomatically checks the "Automatically manage signing" checkbox in Xcode
// Note: it only checks the main Target based on the given Scheme and Configuration
func (p *ProjectHelper) IsSigningManagedAutomatically() (bool, error) {
	targetName := p.MainTarget.Name
	settings, err := p.targetBuildSettings(targetName, p.Configuration)
	if err != nil {
		return false, fmt.Errorf("failed to fetch code signing info from target (%s) settings: %s", targetName, err)
	}
	codeSignStyle, err := settings.String("CODE_SIGN_STYLE")
	if err != nil {
		if errors.As(err, &serialized.KeyNotFoundError{}) {
			log.Debugf("setting CODE_SIGN_STYLE unspecified for target (%s), defaulting to `Manual`", targetName)

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
func resolveEntitlementVariables(entitlements autocodesign.Entitlements, bundleID string) (autocodesign.Entitlements, error) {
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
				log.Warnf("Ignoring iCloud container ID (%s) as can not expand variable: %v", container, err)
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

func configuration(configurationName string, scheme xcscheme.Scheme, xcproj xcodeproj.XcodeProj) (string, error) {
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
		log.Warnf("Using user defined build configuration: %s instead of the scheme's default one: %s.\nMake sure you use the same configuration in further steps.", configurationName, defaultConfiguration)
		configuration = configurationName
	}

	return configuration, nil
}

// mainTargetOfScheme return the main target
func mainTargetOfScheme(proj xcodeproj.XcodeProj, scheme xcscheme.Scheme) (xcodeproj.Target, error) {
	log.Debugf("Searching %d for scheme main target: %s", len(scheme.BuildAction.BuildActionEntries), scheme.Name)

	var blueIdent string
	for _, entry := range scheme.BuildAction.BuildActionEntries {
		if entry.BuildableReference.IsAppReference() {
			blueIdent = entry.BuildableReference.BlueprintIdentifier
			break
		}
	}

	log.Debugf("Searching %d targets for: %s", len(proj.Proj.Targets), blueIdent)

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
func findBuiltProject(pth, schemeName string) (xcodeproj.XcodeProj, xcscheme.Scheme, error) {
	log.TInfof("Locating built project for xcode project: %s, scheme: %s", pth, schemeName)

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

	log.TInfof("Located built project for scheme: %s", schemeName)

	return xcodeProj, *scheme, nil
}
