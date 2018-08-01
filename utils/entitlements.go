package utils

import (
	"fmt"
	"path/filepath"

	"github.com/bitrise-tools/go-xcode/plistutil"
	"github.com/bitrise-tools/xcode-project/serialized"
	"github.com/bitrise-tools/xcode-project/xcodeproj"
	"github.com/bitrise-tools/xcode-project/xcscheme"
	"github.com/bitrise-tools/xcode-project/xcworkspace"
)

// ProjectEntitlementsByBundleID ...
func ProjectEntitlementsByBundleID(pth, schemeName, configurationName string) (map[string]plistutil.PlistData, error) {
	var scheme xcscheme.Scheme
	var schemeContainerDir string

	if xcodeproj.IsXcodeProj(pth) {
		project, err := xcodeproj.Open(pth)
		if err != nil {
			return nil, err
		}

		var ok bool
		scheme, ok = project.Scheme(schemeName)
		if !ok {
			return nil, fmt.Errorf("no scheme found with name: %s in project: %s", schemeName, pth)
		}
		schemeContainerDir = filepath.Dir(pth)
	} else if xcworkspace.IsWorkspace(pth) {
		workspace, err := xcworkspace.Open(pth)
		if err != nil {
			return nil, err
		}

		var ok bool
		var containerProject string
		scheme, containerProject, ok = workspace.Scheme(schemeName)
		if !ok {
			return nil, fmt.Errorf("no scheme found with name: %s in workspace: %s", schemeName, pth)
		}
		schemeContainerDir = filepath.Dir(containerProject)
	} else {
		return nil, fmt.Errorf("unknown project extension: %s", filepath.Ext(pth))
	}

	if configurationName == "" {
		configurationName = scheme.ArchiveAction.BuildConfiguration
	}

	if configurationName == "" {
		return nil, fmt.Errorf("no configuration provided nor default defined for the scheme's (%s) archive action", schemeName)
	}

	var archiveEntry xcscheme.BuildActionEntry
	for _, entry := range scheme.BuildAction.BuildActionEntries {
		if entry.BuildForArchiving != "YES" {
			continue
		}
		archiveEntry = entry
		break
	}

	if archiveEntry.BuildableReference.BlueprintIdentifier == "" {
		return nil, fmt.Errorf("archivable entry not found")
	}

	projectPth, err := archiveEntry.BuildableReference.ReferencedContainerAbsPath(schemeContainerDir)
	if err != nil {
		return nil, err
	}

	project, err := xcodeproj.Open(projectPth)
	if err != nil {
		return nil, err
	}

	mainTarget, ok := project.Proj.Target(archiveEntry.BuildableReference.BlueprintIdentifier)
	if !ok {
		return nil, fmt.Errorf("target not found: %s", archiveEntry.BuildableReference.BlueprintIdentifier)
	}

	targets := append([]xcodeproj.Target{mainTarget}, mainTarget.DependentTargets()...)

	entitlementsByBundleID := map[string]serialized.Object{}

	for _, target := range targets {
		bundleID, err := project.TargetBundleID(target.Name, configurationName)
		if err != nil {
			return nil, fmt.Errorf("Failed to get target (%s) bundle id: %s", target.Name, err)
		}

		entitlements, err := project.TargetCodeSignEntitlements(target.Name, configurationName)
		if err != nil && !serialized.IsKeyNotFoundError(err) {
			return nil, fmt.Errorf("Failed to get target (%s) bundle id: %s", target.Name, err)
		}

		entitlementsByBundleID[bundleID] = entitlements
	}

	return toMapStringPlistData(entitlementsByBundleID), nil
}

func toMapStringPlistData(object map[string]serialized.Object) map[string]plistutil.PlistData {
	converted := map[string]plistutil.PlistData{}
	for key, value := range object {
		converted[key] = plistutil.PlistData(value)
	}
	return converted
}
