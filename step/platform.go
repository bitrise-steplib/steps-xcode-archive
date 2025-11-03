package step

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/v2/log"
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

type BuildSettingProvider interface {
	ReadSchemeBuildSettingString(key string) (string, error)
}

func BuildableTargetPlatform(
	logger log.Logger,
	project BuildSettingProvider,
	addititonalOptions ...string,
) (Platform, error) {
	logger.Printf("Finding platform type")
	sdkValue, err := project.ReadSchemeBuildSettingString("SDKROOT")
	if err != nil {
		return "", fmt.Errorf("failed to read SDKROOT build setting: %w", err)
	}

	platform, err := getPlatform(sdkValue)
	logger.Printf("Platform type: %s", platform)
	return platform, err
}

func getPlatform(sdk string) (Platform, error) {
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
