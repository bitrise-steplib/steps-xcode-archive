package utils

import (
	"fmt"
	"path/filepath"

	project "github.com/bitrise-io/xcode-project"
	"github.com/bitrise-io/xcode-project/serialized"
	"github.com/bitrise-io/xcode-project/xcodeproj"
	"github.com/bitrise-io/xcode-project/xcscheme"
)

// Platform ...
type Platform string

const (
	iOS     Platform = "iOS"
	osX     Platform = "OS X"
	tvOS    Platform = "tvOS"
	watchOS Platform = "watchOS"
)

// OpenArchivableProject ...
func OpenArchivableProject(pth, schemeName, configurationName string) (*xcodeproj.XcodeProj, *xcscheme.Scheme, string, error) {
	scheme, schemeContainerDir, err := project.Scheme(pth, schemeName)
	if err != nil {
		return nil, nil, "", fmt.Errorf("could not get scheme with name %s from path %s", schemeName, pth)
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
	return &xcodeProj, scheme, configurationName, nil
}

// ProjectPlatform ...
func ProjectPlatform(xcodeProj *xcodeproj.XcodeProj, configurationName string) (Platform, error) {
	var projectConfig *xcodeproj.BuildConfiguration
	for _, config := range xcodeProj.Proj.BuildConfigurationList.BuildConfigurations {
		if config.Name == configurationName {
			projectConfig = &config
		}
	}
	if projectConfig == nil {
		return "", fmt.Errorf("%s project configuration not found", configurationName)
	}

	return getPlatform(projectConfig.BuildSettings)
}

func getPlatform(buildSettings serialized.Object) (Platform, error) {
	sdk, err := buildSettings.String("SDKROOT")
	if err != nil {
		return "", fmt.Errorf("failed to get SDKROOT: %s", err)
	}
	switch sdk {
	case "iphoneos":
		return iOS, nil
	case "macosx":
		return osX, nil
	case "appletvos":
		return tvOS, nil
	case "watchos":
		return watchOS, nil
	default:
		return "", fmt.Errorf("unkown SDKROOT: %s", sdk)
	}
}
