package xcodeproj

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/bitrise-io/go-utils/v2/sliceutil"
	"github.com/bitrise-io/go-xcode/v2/xcodeproject/serialized"
)

// appIconSetNameBuildSettingKey is read from the project file, not through BuildSettingsProvider
// like other build settings, so no xcodebuild call is needed per target and configuration. As a
// result, values set in .xcconfig files are not seen, and ${...} references are matched as globs.
const appIconSetNameBuildSettingKey = "ASSETCATALOG_COMPILER_APPICON_NAME"

var buildSettingReferenceRegexp = regexp.MustCompile(`\$\{(.+)\}`)

// AppIconSetPaths maps target IDs to the absolute paths of the targets' .appiconset directories.
func (p *XcodeProj) AppIconSetPaths() (map[string][]string, error) {
	objects, ok := p.rawProj.Object("objects")
	if !ok {
		return nil, fmt.Errorf("failed to parse project: no objects")
	}

	targetToAppIcons := map[string][]string{}
	for _, target := range p.targets {
		appIconSetNames := appIconSetNames(target)
		if len(appIconSetNames) == 0 {
			continue
		}

		catalogs, err := assetCatalogs(target, p.projectID, objects)
		if err != nil {
			return nil, err
		} else if len(catalogs) == 0 {
			continue
		}

		var appIcons []string
		for _, appIconSetName := range appIconSetNames {
			paths, err := p.lookupAppIconPaths(catalogs, appIconSetName, objects)
			if err != nil {
				return nil, err
			} else if len(paths) == 0 {
				return nil, fmt.Errorf("not found app icon set (%s) on paths: %s", appIconSetName, catalogs)
			}
			appIcons = append(appIcons, paths...)
		}
		targetToAppIcons[target.ID] = sliceutil.Unique(appIcons)
	}

	return targetToAppIcons, nil
}

func (p *XcodeProj) lookupAppIconPaths(catalogs []fileReference, appIconSetName string, objects serialized.Object) ([]string, error) {
	var icons []string
	for _, catalog := range catalogs {
		resolvedPath, err := resolveObjectAbsolutePath(catalog.id, p.projectID, p.Path, objects)
		if err != nil {
			return nil, err
		} else if resolvedPath == "" {
			return nil, fmt.Errorf("could not resolve path")
		}

		pattern := buildSettingReferenceRegexp.ReplaceAllString(appIconSetName, "*")

		matches, err := p.pathProvider.Glob(path.Join(p.pathModifier.EscapeGlobPath(resolvedPath), pattern+".appiconset"))
		if err != nil {
			return nil, err
		}

		icons = append(icons, matches...)
	}

	return icons, nil
}

func assetCatalogs(target Target, projectID string, objects serialized.Object) ([]fileReference, error) {
	// Aggregate and legacy targets may not have build phases.
	if !target.isNativeTarget() {
		return nil, nil
	}

	buildPhase, err := findResourcesBuildPhase(target.buildPhaseIDs, objects)
	if err != nil {
		return nil, fmt.Errorf("getting resource build phases failed, error: %w", err)
	}

	return filterAssetCatalogs(buildPhase, objects)
}

func findResourcesBuildPhase(buildPhaseIDs []string, objects serialized.Object) (resourcesBuildPhase, error) {
	for _, id := range buildPhaseIDs {
		rawBuildPhase, ok := objects.Object(id)
		if !ok {
			return resourcesBuildPhase{}, fmt.Errorf("build phase %s not found", id)
		}
		if isResourcesBuildPhase(rawBuildPhase) {
			buildPhase, err := parseResourcesBuildPhase(id, objects)
			if err != nil {
				return resourcesBuildPhase{}, fmt.Errorf("failed to parse ResourcesBuildPhase, error: %w", err)
			}
			return buildPhase, nil
		}
	}
	return resourcesBuildPhase{}, fmt.Errorf("resource build phase not found")
}

func filterAssetCatalogs(buildPhase resourcesBuildPhase, objects serialized.Object) ([]fileReference, error) {
	var catalogs []fileReference
	for _, fileID := range buildPhase.files {
		file, err := parseBuildFile(fileID, objects)
		if err != nil {
			// A build file without a file reference, such as "(null) in Resources", is skipped.
			continue
		}

		rawElement, ok := objects.Object(file.fileRef)
		if !ok {
			return nil, fmt.Errorf("object %s not found", file.fileRef)
		}
		isReference, err := isFileReference(rawElement)
		if err != nil {
			return nil, err
		} else if !isReference {
			// PBXVariantGroup
			continue
		}

		reference, err := parseFileReference(file.fileRef, objects)
		if err != nil {
			return nil, err
		}

		if strings.HasSuffix(reference.path, ".xcassets") {
			catalogs = append(catalogs, reference)
		}
	}
	return catalogs, nil
}

// appIconSetNames returns nothing if any configuration lacks the setting.
func appIconSetNames(target Target) []string {
	var names []string
	for _, configuration := range target.BuildConfigurations {
		name, ok := configuration.BuildSetting(appIconSetNameBuildSettingKey)
		if !ok {
			return nil
		}
		names = append(names, name)
	}
	return names
}
