package exportoptionsgenerator

import (
	"fmt"

	"github.com/bitrise-io/go-xcode/exportoptions"
	"github.com/bitrise-io/go-xcode/xcodeproject/serialized"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcodeproj"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcscheme"
)

// TargetInfoProvider can determine a target's bundle id and codesign entitlements.
type TargetInfoProvider interface {
	TargetBundleID(target, configuration string) (string, error)
	TargetCodeSignEntitlements(target, configuration string) (serialized.Object, error)
}

// XcodebuildTargetInfoProvider implements TargetInfoProvider.
type XcodebuildTargetInfoProvider struct {
	xcodeProj *xcodeproj.XcodeProj
}

// TargetBundleID ...
func (b XcodebuildTargetInfoProvider) TargetBundleID(target, configuration string) (string, error) {
	return b.xcodeProj.TargetBundleID(target, configuration)
}

// TargetCodeSignEntitlements ...
func (b XcodebuildTargetInfoProvider) TargetCodeSignEntitlements(target, configuration string) (serialized.Object, error) {
	return b.xcodeProj.TargetCodeSignEntitlements(target, configuration)
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

func filterApplicationBundleTargets(targets []xcodeproj.Target, exportMethod exportoptions.Method) (filteredTargets []xcodeproj.Target) {
	fmt.Printf("Filtering %v application bundle targets", len(targets))

	for _, target := range targets {
		if !target.IsExecutableProduct() {
			continue
		}

		// App store exports contain App Clip too. App Clip provisioning profile has to be included in export options:
		// ..
		// <key>provisioningProfiles</key>
		// <dict>
		// 	<key>io.bundle.id</key>
		// 	<string>Development Application Profile</string>
		// 	<key>io.bundle.id.AppClipID</key>
		// 	<string>Development App Clip Profile</string>
		// </dict>
		// ..,
		if !exportMethod.IsAppStore() && target.IsAppClipProduct() {
			continue
		}

		filteredTargets = append(filteredTargets, target)
	}

	fmt.Printf("Found %v application bundle targets", len(filteredTargets))

	return
}
