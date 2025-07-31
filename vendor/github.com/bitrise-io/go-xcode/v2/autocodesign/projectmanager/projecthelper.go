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
	"github.com/bitrise-io/go-xcode/xcodeproject/xcworkspace"
	"howett.net/plist"
)

// ProjectHelper ...
type ProjectHelper struct {
	MainTarget       xcodeproj.Target
	DependentTargets []xcodeproj.Target
	UITestTargets    []xcodeproj.Target
	XcProj           xcodeproj.XcodeProj
	XcWorkspace      *xcworkspace.Workspace // nil if working with standalone project
	SchemeName       string                 // The scheme name used
	Configuration    string
	BasePath         string // The original project or workspace path

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

	// Check if we're working with a workspace
	var workspace *xcworkspace.Workspace
	if filepath.Ext(projOrWSPath) == ".xcworkspace" {
		ws, err := xcworkspace.Open(projOrWSPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open workspace: %s", err)
		}
		workspace = &ws
	}

	return &ProjectHelper{
		MainTarget:       mainTarget,
		DependentTargets: dependentTargets,
		UITestTargets:    uiTestTargets,
		XcProj:           xcproj,
		XcWorkspace:      workspace,
		SchemeName:       schemeName,
		Configuration:    conf,
		BasePath:         projOrWSPath,
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
	settings, err := p.buildSettings(p.MainTarget.Name, configurationName)
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
	settings, err := p.buildSettings(targetName, config)
	if err != nil {
		return "", fmt.Errorf("failed to fetch Team ID from target settings (%s): %s", targetName, err)
	}

	devTeam, err := settings.String("DEVELOPMENT_TEAM")
	if serialized.IsKeyNotFoundError(err) {
		return "", nil
	}
	return devTeam, err

}

// buildSettings returns target build settings using workspace or project
// For workspace: uses main SchemeBuildSettings for main target, dedicated SchemeBuildSettings for secondary targets
// For xcodeproj: uses TargetBuildSettings with target name
func (p *ProjectHelper) buildSettings(targetName, conf string) (serialized.Object, error) {
	log.Debugf("🔍 buildSettings: fetching settings for target='%s', config='%s'", targetName, conf)
	log.Debugf("🔍 buildSettings: MainTarget.Name='%s'", p.MainTarget.Name)
	log.Debugf("🔍 buildSettings: Has workspace: %t", p.XcWorkspace != nil)
	if p.XcWorkspace != nil {
		log.Debugf("🔍 buildSettings: Workspace path='%s', Scheme='%s'", p.XcWorkspace.Path, p.SchemeName)
	}
	log.Debugf("🔍 buildSettings: Project path='%s'", p.XcProj.Path)

	targetCache, ok := p.buildSettingsCache[targetName]
	if ok {
		confCache, ok := targetCache[conf]
		if ok {
			log.Debugf("✅ buildSettings: Using cached settings for target='%s'", targetName)
			return confCache, nil
		}
	}

	var settings serialized.Object
	var err error

	if p.XcWorkspace != nil {
		if targetName == p.MainTarget.Name {
			// For main target: use the main scheme's build settings
			log.Debugf("📍 buildSettings: Using main scheme '%s' for MAIN target='%s' (workspace context)", p.SchemeName, targetName)
			settings, err = p.XcWorkspace.SchemeBuildSettings(p.SchemeName, conf)
		} else {
			// For secondary targets: try SchemeBuildSettings first, fallback to TargetBuildSettings if it fails
			log.Debugf("📍 buildSettings: Using SchemeBuildSettings for SECONDARY target='%s' (workspace context - keep first values)", targetName)
			settings, err = p.XcWorkspace.SchemeBuildSettings(targetName, conf)

			// If scheme build settings fail for secondary target, fallback to project target build settings
			if err != nil {
				log.Warnf("⚠️ buildSettings: SchemeBuildSettings failed for secondary target='%s': %s", targetName, err)
				log.Debugf("🔄 buildSettings: Falling back to project TargetBuildSettings for target='%s'", targetName)
				settings, err = p.XcProj.TargetBuildSettings(targetName, conf)
				if err != nil {
					log.Errorf("❌ buildSettings: Fallback TargetBuildSettings also failed for target='%s': %s", targetName, err)
				} else {
					log.Debugf("✅ buildSettings: Fallback TargetBuildSettings succeeded for target='%s'", targetName)
				}
			}
		}
	} else {
		// Use project TargetBuildSettings for standalone projects
		log.Debugf("📍 buildSettings: Using project TargetBuildSettings for STANDALONE project target='%s'", targetName)
		settings, err = p.XcProj.TargetBuildSettings(targetName, conf)
	}

	if err != nil {
		log.Errorf("❌ buildSettings: Failed to fetch settings for target='%s': %s", targetName, err)
		return nil, err
	}

	log.Debugf("✅ buildSettings: Successfully fetched settings for target='%s'", targetName)

	if targetCache == nil {
		targetCache = map[string]serialized.Object{}
	}
	targetCache[conf] = settings

	if p.buildSettingsCache == nil {
		p.buildSettingsCache = map[string]map[string]serialized.Object{}
	}
	p.buildSettingsCache[targetName] = targetCache

	return settings, nil
}

// TargetBundleID returns the target bundle ID
// First it tries to fetch the bundle ID from the `PRODUCT_BUNDLE_IDENTIFIER` build settings
// If it's no available it will fetch the target's Info.plist and search for the `CFBundleIdentifier` key.
// The CFBundleIdentifier's value is not resolved in the Info.plist, so it will try to resolve it by the resolveBundleID()
// It returns  the target bundle ID
func (p *ProjectHelper) TargetBundleID(name, conf string) (string, error) {
	log.Debugf("🆔 TargetBundleID: START - Resolving bundle ID for target='%s', config='%s'", name, conf)
	log.Debugf("🆔 TargetBundleID: MainTarget.Name='%s'", p.MainTarget.Name)
	log.Debugf("🆔 TargetBundleID: Has workspace: %t", p.XcWorkspace != nil)
	if p.XcWorkspace != nil {
		log.Debugf("🆔 TargetBundleID: Workspace path='%s', Scheme='%s'", p.XcWorkspace.Path, p.SchemeName)
	}
	log.Debugf("🆔 TargetBundleID: Project path='%s'", p.XcProj.Path)
	log.Debugf("🆔 TargetBundleID: BasePath='%s'", p.BasePath)

	log.Debugf("📋 TargetBundleID: Fetching build settings for target='%s'", name)
	settings, err := p.buildSettings(name, conf)
	if err != nil {
		log.Errorf("❌ TargetBundleID: Failed to fetch target (%s) settings: %s", name, err)
		return "", fmt.Errorf("failed to fetch target (%s) settings: %s", name, err)
	}
	log.Debugf("✅ TargetBundleID: Successfully fetched build settings for target='%s'", name)

	log.Debugf("🔍 TargetBundleID: Looking for PRODUCT_BUNDLE_IDENTIFIER in build settings...")
	bundleID, err := settings.String("PRODUCT_BUNDLE_IDENTIFIER")
	if err != nil && !serialized.IsKeyNotFoundError(err) {
		log.Errorf("❌ TargetBundleID: Failed to parse PRODUCT_BUNDLE_IDENTIFIER: %s", err)
		return "", fmt.Errorf("failed to parse target (%s) build settings attribute PRODUCT_BUNDLE_IDENTIFIER: %s", name, err)
	}
	if bundleID != "" {
		log.Debugf("✅ TargetBundleID: Found PRODUCT_BUNDLE_IDENTIFIER='%s' for target='%s'", bundleID, name)
		log.Debugf("🆔 TargetBundleID: END - Returning bundle ID from build settings: '%s'", bundleID)
		return bundleID, nil
	}
	log.Debugf("ℹ️ TargetBundleID: PRODUCT_BUNDLE_IDENTIFIER not found in build settings, checking Info.plist...")

	log.Debugf("PRODUCT_BUNDLE_IDENTIFIER env not found in 'xcodebuild -showBuildSettings -project %s -target %s -configuration %s command's output, checking the Info.plist file's CFBundleIdentifier property...", p.XcProj.Path, name, conf)

	log.Debugf("📄 TargetBundleID: Looking for INFOPLIST_FILE in build settings...")
	infoPlistPath, err := settings.String("INFOPLIST_FILE")
	if err != nil {
		log.Errorf("❌ TargetBundleID: Failed to find INFOPLIST_FILE: %s", err)
		return "", fmt.Errorf("failed to find Info.plist file: %s", err)
	}
	log.Debugf("✅ TargetBundleID: Found INFOPLIST_FILE='%s'", infoPlistPath)

	// Use the original base path (workspace or project) to resolve relative paths
	var basePath string
	if filepath.Ext(p.BasePath) == ".xcworkspace" {
		basePath = filepath.Dir(p.BasePath)
		log.Debugf("📁 TargetBundleID: Using workspace base path='%s'", basePath)
	} else {
		basePath = filepath.Dir(p.XcProj.Path)
		log.Debugf("📁 TargetBundleID: Using project base path='%s'", basePath)
	}

	absoluteInfoPlistPath := path.Join(basePath, infoPlistPath)
	log.Debugf("📍 TargetBundleID: Resolved absolute Info.plist path='%s'", absoluteInfoPlistPath)
	infoPlistPath = absoluteInfoPlistPath

	if infoPlistPath == "" {
		log.Errorf("❌ TargetBundleID: Empty Info.plist path after resolution")
		return "", fmt.Errorf("failed to to determine bundle id: xcodebuild -showBuildSettings does not contains PRODUCT_BUNDLE_IDENTIFIER nor INFOPLIST_FILE' unless info_plist_path")
	}

	log.Debugf("📖 TargetBundleID: Reading Info.plist file: '%s'", infoPlistPath)
	b, err := fileutil.ReadBytesFromFile(infoPlistPath)
	if err != nil {
		log.Errorf("❌ TargetBundleID: Failed to read Info.plist file '%s': %s", infoPlistPath, err)
		return "", fmt.Errorf("failed to read Info.plist: %s", err)
	}
	log.Debugf("✅ TargetBundleID: Successfully read %d bytes from Info.plist", len(b))

	log.Debugf("🔧 TargetBundleID: Parsing Info.plist content...")
	var options map[string]interface{}
	if _, err := plist.Unmarshal(b, &options); err != nil {
		log.Errorf("❌ TargetBundleID: Failed to unmarshal Info.plist: %s", err)
		return "", fmt.Errorf("failed to unmarshal Info.plist: %s ", err)
	}
	log.Debugf("✅ TargetBundleID: Successfully parsed Info.plist with %d keys", len(options))

	log.Debugf("🔍 TargetBundleID: Looking for CFBundleIdentifier in Info.plist...")
	bundleID, ok := options["CFBundleIdentifier"].(string)
	if !ok || bundleID == "" {
		log.Errorf("❌ TargetBundleID: CFBundleIdentifier not found or empty in Info.plist")
		log.Debugf("🔍 TargetBundleID: Available keys in Info.plist: %v", func() []string {
			keys := make([]string, 0, len(options))
			for k := range options {
				keys = append(keys, k)
			}
			return keys
		}())
		return "", fmt.Errorf("failed to parse CFBundleIdentifier from the Info.plist")
	}
	log.Debugf("✅ TargetBundleID: Found CFBundleIdentifier='%s' in Info.plist", bundleID)

	log.Debugf("✅ TargetBundleID: Found CFBundleIdentifier='%s' in Info.plist", bundleID)

	log.Debugf("🔍 TargetBundleID: Checking if bundle ID contains variables...")
	if !strings.Contains(bundleID, "$") {
		log.Debugf("✅ TargetBundleID: Bundle ID contains no variables, returning as-is: '%s'", bundleID)
		log.Debugf("🆔 TargetBundleID: END - Returning bundle ID from Info.plist: '%s'", bundleID)
		return bundleID, nil
	}

	log.Debugf("🔧 TargetBundleID: Bundle ID contains variables, need to expand: '%s'", bundleID)
	log.Debugf("CFBundleIdentifier defined with variable: %s, trying to resolve it...", bundleID)

	log.Debugf("🔧 TargetBundleID: Expanding variables in bundle ID...")
	resolved, err := expandTargetSetting(bundleID, settings)
	if err != nil {
		log.Errorf("❌ TargetBundleID: Failed to resolve bundle ID variables: %s", err)
		return "", fmt.Errorf("failed to resolve bundle ID: %s", err)
	}

	log.Debugf("✅ TargetBundleID: Successfully resolved bundle ID: '%s' -> '%s'", bundleID, resolved)
	log.Debugf("resolved CFBundleIdentifier: %s", resolved)
	log.Debugf("🆔 TargetBundleID: END - Returning resolved bundle ID: '%s'", resolved)

	return resolved, nil
}

func (p *ProjectHelper) targetEntitlements(name, config, bundleID string) (autocodesign.Entitlements, error) {
	log.Debugf("🔍 targetEntitlements: fetching entitlements for target='%s', config='%s', bundleID='%s'", name, config, bundleID)
	log.Debugf("🔍 targetEntitlements: MainTarget.Name='%s'", p.MainTarget.Name)
	log.Debugf("🔍 targetEntitlements: Has workspace: %t", p.XcWorkspace != nil)

	var entitlements serialized.Object
	var err error

	if p.XcWorkspace != nil {
		if name == p.MainTarget.Name {
			// For main target: use the main scheme's entitlements
			log.Debugf("📍 targetEntitlements: Using main scheme '%s' for MAIN target='%s' (workspace context)", p.SchemeName, name)
			entitlements, err = p.XcWorkspace.SchemeCodeSignEntitlements(p.SchemeName, config)
		} else {
			// For secondary targets: try SchemeCodeSignEntitlements first, fallback to TargetCodeSignEntitlements if it fails
			log.Debugf("📍 targetEntitlements: Using target-specific entitlements for SECONDARY target='%s' (workspace context)", name)
			entitlements, err = p.XcWorkspace.SchemeCodeSignEntitlements(name, config)

			// If scheme entitlements fail for secondary target, fallback to project target entitlements
			if err != nil {
				log.Warnf("⚠️ targetEntitlements: SchemeCodeSignEntitlements failed for secondary target='%s': %s", name, err)
				log.Debugf("🔄 targetEntitlements: Falling back to project TargetCodeSignEntitlements for target='%s'", name)
				entitlements, err = p.XcProj.TargetCodeSignEntitlements(name, config)
				if err != nil {
					log.Errorf("❌ targetEntitlements: Fallback TargetCodeSignEntitlements also failed for target='%s': %s", name, err)
				} else {
					log.Debugf("✅ targetEntitlements: Fallback TargetCodeSignEntitlements succeeded for target='%s'", name)
				}
			}
		}
	} else {
		// Use project TargetCodeSignEntitlements for standalone projects
		log.Debugf("📍 targetEntitlements: Using project TargetCodeSignEntitlements for STANDALONE project target='%s'", name)
		entitlements, err = p.XcProj.TargetCodeSignEntitlements(name, config)
	}

	if err != nil && !serialized.IsKeyNotFoundError(err) {
		log.Errorf("❌ targetEntitlements: Failed to fetch entitlements for target='%s': %s", name, err)
		return nil, err
	}

	if err != nil && serialized.IsKeyNotFoundError(err) {
		log.Debugf("ℹ️ targetEntitlements: No entitlements found for target='%s' (this is normal for some targets)", name)
	} else {
		log.Debugf("✅ targetEntitlements: Successfully fetched entitlements for target='%s'", name)
	}

	return resolveEntitlementVariables(autocodesign.Entitlements(entitlements), bundleID)
}

// IsSigningManagedAutomatically checks the "Automatically manage signing" checkbox in Xcode
// Note: it only checks the main Target based on the given Scheme and Configuration
func (p *ProjectHelper) IsSigningManagedAutomatically() (bool, error) {
	targetName := p.MainTarget.Name
	settings, err := p.buildSettings(targetName, p.Configuration)
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
