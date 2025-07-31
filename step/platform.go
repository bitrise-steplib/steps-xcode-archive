package step

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/xcodebuild"
	"github.com/bitrise-io/go-xcode/xcodeproject/schemeint"
	"github.com/bitrise-io/go-xcode/xcodeproject/serialized"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcodeproj"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcscheme"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcworkspace"
)

type Platform string

const (
	detectPlatform Platform = "detect"
	iOS            Platform = "iOS"
	osX            Platform = "OS X"
	tvOS           Platform = "tvOS"
	watchOS        Platform = "watchOS"
	visionOS       Platform = "visionOS"
)

// ArchivableProject represents either a workspace or a project that can be archived
type ArchivableProject interface {
	BuildSettings(scheme, configuration string, customOptions ...string) (serialized.Object, error)
	GetProject() (*xcodeproj.XcodeProj, error)
}

// WorkspaceProject wraps a workspace and its main project
type WorkspaceProject struct {
	Workspace xcworkspace.Workspace
	XcodeProj *xcodeproj.XcodeProj
}

func (w WorkspaceProject) BuildSettings(scheme, configuration string, customOptions ...string) (serialized.Object, error) {
	return w.Workspace.SchemeBuildSettings(scheme, configuration, customOptions...)
}

func (w WorkspaceProject) GetProject() (*xcodeproj.XcodeProj, error) {
	return w.XcodeProj, nil
}

// XcodeProjWrapper wraps a standalone project
type XcodeProjWrapper struct {
	XcodeProj *xcodeproj.XcodeProj
}

func (p XcodeProjWrapper) BuildSettings(scheme, configuration string, customOptions ...string) (serialized.Object, error) {
	// For xcodeproj projects, use xcodebuild command directly with the project path
	commandModel := xcodebuild.NewShowBuildSettingsCommand(p.XcodeProj.Path)
	commandModel.SetScheme(scheme)
	commandModel.SetConfiguration(configuration)
	commandModel.SetCustomOptions(customOptions)
	return commandModel.RunAndReturnSettings(true)
}

func (p XcodeProjWrapper) GetProject() (*xcodeproj.XcodeProj, error) {
	return p.XcodeProj, nil
}

func parsePlatform(platform string) (Platform, error) {
	switch strings.ToLower(platform) {
	case "detect":
		return detectPlatform, nil
	case "ios":
		return iOS, nil
	case "tvos":
		return tvOS, nil
	case "watchos":
		return watchOS, nil
	case "visionos":
		return visionOS, nil
	default:
		return "", fmt.Errorf("unknown platform: %s", platform)
	}
}

func OpenArchivableProject(pth, schemeName, configurationName string) (ArchivableProject, *xcscheme.Scheme, string, error) {
	scheme, schemeContainerDir, err := schemeint.Scheme(pth, schemeName)
	if err != nil {
		return nil, nil, "", fmt.Errorf("could not get scheme (%s) from path (%s): %s", schemeName, pth, err)
	}
	if configurationName == "" {
		configurationName = scheme.ArchiveAction.BuildConfiguration
	}

	if configurationName == "" {
		return nil, nil, "", fmt.Errorf("no configuration provided nor default defined for the scheme's (%s) archive action", schemeName)
	}

	archiveEntry, ok := scheme.AppBuildActionEntry()
	if !ok {
		return nil, nil, "", fmt.Errorf("archivable entry not found")
	}

	projectPth, err := archiveEntry.BuildableReference.ReferencedContainerAbsPath(filepath.Dir(schemeContainerDir))
	if err != nil {
		return nil, nil, "", err
	}

	xcodeProj, err := xcodeproj.Open(projectPth)
	if err != nil {
		return nil, nil, "", err
	}

	// Check if the original path is a workspace
	if strings.HasSuffix(pth, xcworkspace.XCWorkspaceExtension) {
		workspace, err := xcworkspace.Open(pth)
		if err != nil {
			return nil, nil, "", fmt.Errorf("failed to open workspace: %s", err)
		}
		return WorkspaceProject{Workspace: workspace, XcodeProj: &xcodeProj}, scheme, configurationName, nil
	}

	// Otherwise it's a standalone project
	return XcodeProjWrapper{XcodeProj: &xcodeProj}, scheme, configurationName, nil
}

type BuildSettingsProvider interface {
	BuildSettings(archivableProject ArchivableProject, schemeName, target, configuration string, customOptions ...string) (serialized.Object, error)
}

type XcodeBuild struct {
}

func (x XcodeBuild) BuildSettings(archivableProject ArchivableProject, schemeName, target, configuration string, customOptions ...string) (serialized.Object, error) {
	// Use the archivable project's scheme build settings method
	return archivableProject.BuildSettings(schemeName, configuration, customOptions...)
}

func BuildableTargetPlatform(
	archivableProject ArchivableProject,
	scheme *xcscheme.Scheme,
	configurationName string,
	additionalOptions []string,
	provider BuildSettingsProvider,
	logger log.Logger,
) (Platform, error) {
	archiveEntry, ok := scheme.AppBuildActionEntry()
	if !ok {
		return "", fmt.Errorf("archivable entry not found in scheme: %s", scheme.Name)
	}

	xcodeProj, err := archivableProject.GetProject()
	if err != nil {
		return "", fmt.Errorf("failed to get project: %s", err)
	}

	mainTarget, ok := xcodeProj.Proj.Target(archiveEntry.BuildableReference.BlueprintIdentifier)
	if !ok {
		return "", fmt.Errorf("target not found: %s", archiveEntry.BuildableReference.BlueprintIdentifier)
	}

	settings, err := provider.BuildSettings(archivableProject, scheme.Name, mainTarget.Name, configurationName, additionalOptions...)
	if err != nil {
		return "", fmt.Errorf("failed to get target (%s) build settings: %s", mainTarget.Name, err)
	}

	platform, err := getPlatform(settings)

	logger.Printf("Platform type: %s", platform)

	return platform, err
}

func getPlatform(buildSettings serialized.Object) (Platform, error) {
	/*
		Xcode help:
		Base SDK (SDKROOT)
		The name or path of the base SDK being used during the build.
		The product will be built against the headers and libraries located inside the indicated SDK.
		This path will be prepended to all search paths, and will be passed through the environment to the compiler and linker.
		Additional SDKs can be specified in the Additional SDKs (ADDITIONAL_SDKS) setting.

		Examples:
		- /Applications/Xcode.app/Contents/Developer/Platforms/AppleTVOS.platform/Developer/SDKs/AppleTVOS.sdk
		- /Applications/Xcode.app/Contents/Developer/Platforms/AppleTVSimulator.platform/Developer/SDKs/AppleTVSimulator13.4.sdk
		- /Applications/Xcode.app/Contents/Developer/Platforms/iPhoneOS.platform/Developer/SDKs/iPhoneOS13.4.sdk
		- /Applications/Xcode.app/Contents/Developer/Platforms/iPhoneSimulator.platform/Developer/SDKs/iPhoneSimulator.sdk
		- /Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX10.15.sdk
		- /Applications/Xcode.app/Contents/Developer/Platforms/WatchOS.platform/Developer/SDKs/WatchOS.sdk
		- /Applications/Xcode.app/Contents/Developer/Platforms/WatchSimulator.platform/Developer/SDKs/WatchSimulator.sdk
		- iphoneos
		- macosx
		- appletvos
		- watchos
	*/
	sdk, err := buildSettings.String("SDKROOT")
	if err != nil {
		return "", fmt.Errorf("failed to get SDKROOT: %s", err)
	}

	sdk = strings.ToLower(sdk)
	if filepath.Ext(sdk) == ".sdk" {
		sdk = filepath.Base(sdk)
	}

	switch {
	case strings.HasPrefix(sdk, "iphoneos"):
		return iOS, nil
	case strings.HasPrefix(sdk, "macosx"):
		return osX, nil
	case strings.HasPrefix(sdk, "appletvos"):
		return tvOS, nil
	case strings.HasPrefix(sdk, "watchos"):
		return watchOS, nil
	case strings.HasPrefix(sdk, "xros"):
		// visionOS SDK is called xros (as of Xcode 15.2), but the platform is called visionOS (e.g. in the destination specifier)
		return visionOS, nil
	default:
		return "", fmt.Errorf("unkown SDKROOT: %s", sdk)
	}
}
