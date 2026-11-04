package xcodeproj

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bitrise-io/go-xcode/v2/xcodeproject/serialized"
)

const (
	bundleIDBuildSettingKey     = "PRODUCT_BUNDLE_IDENTIFIER"
	entitlementsBuildSettingKey = "CODE_SIGN_ENTITLEMENTS"
	infoPlistBuildSettingKey    = "INFOPLIST_FILE"
)

var (
	// $(KEY) or ${KEY}, optionally with a :modifier.
	bracedReferenceRegexp = regexp.MustCompile(`[$][{(][^$]*?[)}]`)
	// $KEY
	bareReferenceRegexp = regexp.MustCompile(`[$][^$]*`)

	referencePunctuationReplacer = strings.NewReplacer("$", "", "(", "", ")", "", "{", "", "}", "")
)

// ErrCodeSignEntitlementsNotFound is returned when a target declares no CODE_SIGN_ENTITLEMENTS.
var ErrCodeSignEntitlementsNotFound = errors.New("target has no code sign entitlements file")

// ErrInfoPlistNotFound is returned when a target declares no INFOPLIST_FILE, as with a generated
// Info.plist.
var ErrInfoPlistNotFound = errors.New("target has no Info.plist file")

// TargetBuildSettings returns the target's effective build settings, looked up by target (not
// scheme) with xcodebuild. Destination-style extraArgs have no effect in target mode.
func (p *XcodeProj) TargetBuildSettings(target, configuration string, extraArgs ...string) (serialized.Object, error) {
	return p.buildSettings.TargetBuildSettings(p.Path, target, configuration, extraArgs...)
}

// TargetBundleID returns the resolved bundle ID, falling back to the Info.plist's CFBundleIdentifier.
func (p *XcodeProj) TargetBundleID(target, configuration string) (string, error) {
	buildSettings, err := p.TargetBuildSettings(target, configuration)
	if err != nil {
		return "", err
	}

	if bundleID, ok := buildSettings.String(bundleIDBuildSettingKey); ok && bundleID != "" {
		return resolveBundleID(bundleID, buildSettings)
	}

	infoPlist, err := p.readTargetInfoPlist(target, buildSettings)
	if err != nil {
		return "", fmt.Errorf("no %s build setting, and reading the Info.plist instead failed: %w", bundleIDBuildSettingKey, err)
	}

	bundleID, ok := infoPlist.String("CFBundleIdentifier")
	if !ok || bundleID == "" {
		return "", fmt.Errorf("no %s build setting nor CFBundleIdentifier Info.plist entry found for target %s", bundleIDBuildSettingKey, target)
	}

	return resolveBundleID(bundleID, buildSettings)
}

// TargetCodeSignEntitlements returns the target's entitlements, or ErrCodeSignEntitlementsNotFound
// if the target declares none.
func (p *XcodeProj) TargetCodeSignEntitlements(target, configuration string) (serialized.Object, error) {
	buildSettings, err := p.TargetBuildSettings(target, configuration)
	if err != nil {
		return nil, err
	}

	pth, ok := p.buildSettingPath(buildSettings, entitlementsBuildSettingKey)
	if !ok {
		return nil, fmt.Errorf("target %s: %w", target, ErrCodeSignEntitlementsNotFound)
	}

	entitlements, err := p.readPlist(pth)
	if err != nil {
		return nil, fmt.Errorf("failed to read entitlements of target %s: %w", target, err)
	}

	return entitlements, nil
}

// TargetInfoplistPath returns the absolute path of the target's Info.plist, or ErrInfoPlistNotFound
// if the target declares none.
func (p *XcodeProj) TargetInfoplistPath(target, configuration string) (string, error) {
	buildSettings, err := p.TargetBuildSettings(target, configuration)
	if err != nil {
		return "", err
	}

	pth, ok := p.buildSettingPath(buildSettings, infoPlistBuildSettingKey)
	if !ok {
		return "", fmt.Errorf("target %s: %w", target, ErrInfoPlistNotFound)
	}

	return pth, nil
}

func (p *XcodeProj) readTargetInfoPlist(target string, buildSettings serialized.Object) (serialized.Object, error) {
	pth, ok := p.buildSettingPath(buildSettings, infoPlistBuildSettingKey)
	if !ok {
		return nil, fmt.Errorf("target %s: %w", target, ErrInfoPlistNotFound)
	}
	return p.readPlist(pth)
}

func (p *XcodeProj) buildSettingPath(buildSettings serialized.Object, key string) (string, bool) {
	pth, ok := buildSettings.String(key)
	if !ok || pth == "" {
		return "", false
	}

	if isRelativePath(pth) {
		pth = filepath.Join(filepath.Dir(p.Path), pth)
	}

	return pth, true
}

// isRelativePath treats a path starting with a build setting reference ($) as absolute.
func isRelativePath(pth string) bool {
	switch {
	case strings.HasPrefix(pth, "./"):
		return true
	case strings.HasPrefix(pth, "/"):
		return false
	case strings.HasPrefix(pth, "$"):
		return false
	default:
		return true
	}
}

func (p *XcodeProj) readPlist(pth string) (serialized.Object, error) {
	data, _, err := p.plistFileHandler.Read(pth)
	if err != nil {
		return nil, err
	}
	return serialized.Object(data), nil
}

func resolveBundleID(bundleID string, buildSettings serialized.Object) (string, error) {
	seen := map[string]bool{}
	resolved := bundleID

	for strings.Contains(resolved, "$") {
		expanded, err := expandBuildSetting(resolved, buildSettings)
		if err != nil {
			return "", err
		}

		if seen[expanded] {
			return "", fmt.Errorf("bundle id reference cycle found while resolving %s", bundleID)
		}
		seen[expanded] = true

		resolved = expanded
	}

	return resolved, nil
}

func expandBuildSetting(value string, buildSettings serialized.Object) (string, error) {
	if bracedReferenceRegexp.MatchString(value) {
		return expandBracedReference(value, buildSettings)
	}
	return expandBareReference(value, buildSettings)
}

func expandBracedReference(value string, buildSettings serialized.Object) (string, error) {
	reference := bracedReferenceRegexp.FindString(value)

	key := referencePunctuationReplacer.Replace(reference)
	key = strings.Split(key, ":")[0]

	settingValue, ok := buildSettings.String(key)
	if !ok {
		return "", fmt.Errorf("failed to find env in build settings: %s", key)
	}

	return strings.ReplaceAll(value, reference, settingValue), nil
}

// expandBareReference shortens an undelimited $KEY until it matches a build setting.
func expandBareReference(value string, buildSettings serialized.Object) (string, error) {
	if !bareReferenceRegexp.MatchString(value) {
		return "", fmt.Errorf("failed to match a build setting reference in %s", value)
	}

	reference := bareReferenceRegexp.FindString(value)

	var settingValue string
	for len(reference) > 1 {
		var ok bool
		settingValue, ok = buildSettings.String(strings.Replace(reference, "$", "", 1))
		if ok {
			break
		}
		reference = reference[:len(reference)-1]
	}

	return strings.ReplaceAll(value, reference, settingValue), nil
}
