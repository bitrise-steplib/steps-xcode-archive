package utils

import (
	"fmt"
	"path/filepath"

	"github.com/bitrise-io/go-xcode/plistutil"
	"github.com/bitrise-io/xcode-project/serialized"
	"github.com/bitrise-io/xcode-project/xcodeproj"
	"github.com/bitrise-io/xcode-project"
)

// ProjectEntitlementsByBundleID ...
func ProjectEntitlementsByBundleID(pth, schemeName, configurationName string) (map[string]plistutil.PlistData, error) {
	scheme,schemeContainerDir,err := project.Scheme(pth,schemeName)
	if err != nil {
		return nil, fmt.Errorf("could not get scheme with name %s from path %s", schemeName,pth)
	}
	if configurationName == "" {
		configurationName = scheme.ArchiveAction.BuildConfiguration
	}

	if configurationName == "" {
		return nil, fmt.Errorf("no configuration provided nor default defined for the scheme's (%s) archive action", schemeName)
	}

	archiveEntry, ok := scheme.AppBuildActionEntry()
	if !ok {
		return nil, fmt.Errorf("archivable entry not found")
	}

	projectPth, err := archiveEntry.BuildableReference.ReferencedContainerAbsPath(filepath.Dir(schemeContainerDir))
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

	targets := append([]xcodeproj.Target{mainTarget}, mainTarget.DependentExecutableProductTargets(false)...)

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
