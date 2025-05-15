package exportoptionsgenerator

import (
	"fmt"

	"github.com/bitrise-io/go-xcode/plistutil"
	"github.com/bitrise-io/go-xcode/xcodeproject/serialized"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcodeproj"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcscheme"
)

// ArchiveInfo contains the distribution bundle ID(s)	and entitlements of the main target and its dependencies.
type ArchiveInfo struct {
	AppBundleID            string
	AppClipBundleID        string
	EntitlementsByBundleID map[string]plistutil.PlistData
}

// ReadArchiveInfoFromXcodeproject reads the Bundle ID for the given scheme and configuration.
func ReadArchiveInfoFromXcodeproject(xcodeProj *xcodeproj.XcodeProj, scheme *xcscheme.Scheme, configuration string) (ArchiveInfo, error) {
	mainTarget, err := ArchivableApplicationTarget(xcodeProj, scheme)
	if err != nil {
		return ArchiveInfo{}, err
	}

	dependentTargets := filterApplicationBundleTargets(xcodeProj.DependentTargetsOfTarget(*mainTarget))
	targets := append([]xcodeproj.Target{*mainTarget}, dependentTargets...)

	mainTargetBundleID := ""
	appClipBundleID := ""
	entitlementsByBundleID := map[string]plistutil.PlistData{}
	for i, target := range targets {
		bundleID, err := xcodeProj.TargetBundleID(target.Name, configuration)
		if err != nil {
			return ArchiveInfo{}, fmt.Errorf("failed to get target (%s) bundle id: %s", target.Name, err)
		}

		entitlements, err := xcodeProj.TargetCodeSignEntitlements(target.Name, configuration)
		if err != nil && !serialized.IsKeyNotFoundError(err) {
			return ArchiveInfo{}, fmt.Errorf("failed to get target (%s) bundle id: %s", target.Name, err)
		}

		entitlementsByBundleID[bundleID] = plistutil.PlistData(entitlements)

		if target.IsAppClipProduct() {
			appClipBundleID = bundleID
		}
		if i == 0 {
			mainTargetBundleID = bundleID
		}
	}

	return ArchiveInfo{
		AppBundleID:            mainTargetBundleID,
		AppClipBundleID:        appClipBundleID,
		EntitlementsByBundleID: entitlementsByBundleID,
	}, nil
}

// ArchivableApplicationTarget locate archivable app target from a given project and scheme
func ArchivableApplicationTarget(xcodeProj *xcodeproj.XcodeProj, scheme *xcscheme.Scheme) (*xcodeproj.Target, error) {
	archiveEntry, ok := scheme.AppBuildActionEntry()
	if !ok {
		return nil, fmt.Errorf("archivable entry not found in project: %s for scheme: %s", xcodeProj.Path, scheme.Name)
	}

	mainTarget, ok := xcodeProj.Proj.Target(archiveEntry.BuildableReference.BlueprintIdentifier)
	if !ok {
		return nil, fmt.Errorf("target not found: %s", archiveEntry.BuildableReference.BlueprintIdentifier)
	}

	return &mainTarget, nil
}

func filterApplicationBundleTargets(targets []xcodeproj.Target) (filteredTargets []xcodeproj.Target) {
	for _, target := range targets {
		if !target.IsExecutableProduct() {
			continue
		}

		filteredTargets = append(filteredTargets, target)
	}

	return
}
