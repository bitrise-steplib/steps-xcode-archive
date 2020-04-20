package utils

import (
	"fmt"

	"github.com/bitrise-io/xcode-project/xcscheme"

	"github.com/bitrise-io/go-xcode/plistutil"
	"github.com/bitrise-io/xcode-project/serialized"
	"github.com/bitrise-io/xcode-project/xcodeproj"
)

// ProjectEntitlementsByBundleID ...
func ProjectEntitlementsByBundleID(xcodeProj *xcodeproj.XcodeProj, scheme *xcscheme.Scheme, configurationName string) (map[string]plistutil.PlistData, error) {
	archiveEntry, ok := scheme.AppBuildActionEntry()
	if !ok {
		return nil, fmt.Errorf("archivable entry not found in project: %s, scheme: %s", xcodeProj.Path, scheme.Name)
	}

	mainTarget, ok := xcodeProj.Proj.Target(archiveEntry.BuildableReference.BlueprintIdentifier)
	if !ok {
		return nil, fmt.Errorf("target not found: %s", archiveEntry.BuildableReference.BlueprintIdentifier)
	}

	targets := append([]xcodeproj.Target{mainTarget}, mainTarget.DependentExecutableProductTargets(false)...)

	entitlementsByBundleID := map[string]serialized.Object{}

	for _, target := range targets {
		bundleID, err := xcodeProj.TargetBundleID(target.Name, configurationName)
		if err != nil {
			return nil, fmt.Errorf("failed to get target (%s) bundle id: %s", target.Name, err)
		}

		entitlements, err := xcodeProj.TargetCodeSignEntitlements(target.Name, configurationName)
		if err != nil && !serialized.IsKeyNotFoundError(err) {
			return nil, fmt.Errorf("failed to get target (%s) bundle id: %s", target.Name, err)
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
