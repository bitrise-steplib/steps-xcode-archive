package cache

import (
	"fmt"
	"path"
	"strings"

	"github.com/bitrise-io/go-steputils/cache"
)

// SwiftPackagesStateInvalid is the partial error message printed out if swift packages cache is invalid.
// Can be used to detect invalid state and clear the path returned by SwiftPackagesPath.
// xcodebuild: error: Could not resolve package dependencies:
//   The repository at [path] is invalid; try resetting package caches
const SwiftPackagesStateInvalid = "Could not resolve package dependencies:"

// SwiftPackagesPath returns the Swift packages cache dir path. The input must be an absolute path.
// The directory is: $HOME/Library/Developer/Xcode/DerivedData/[PER_PROJECT_DERIVED_DATA]/SourcePackages.
func SwiftPackagesPath(xcodeProjectPath string) (string, error) {
	if !path.IsAbs(xcodeProjectPath) {
		return "", fmt.Errorf("project path not an absolute path: %s", xcodeProjectPath)
	}

	if !strings.HasSuffix(xcodeProjectPath, ".xcodeproj") && !strings.HasSuffix(xcodeProjectPath, ".xcworkspace") {
		return "", fmt.Errorf("invalid Xcode project path %s, no .xcodeproj or .xcworkspace suffix found", xcodeProjectPath)
	}

	projectDerivedData, err := xcodeProjectDerivedDataPath(xcodeProjectPath)
	if err != nil {
		return "", err
	}

	return path.Join(projectDerivedData, "SourcePackages"), nil
}

// CollectSwiftPackages marks the Swift Package Manager packages directory to be added the cache.
// The directory cached is: $HOME/Library/Developer/Xcode/DerivedData/[PER_PROJECT_DERIVED_DATA]/SourcePackages.
func CollectSwiftPackages(xcodeProjectPath string) error {
	swiftPackagesDir, err := SwiftPackagesPath(xcodeProjectPath)
	if err != nil {
		return fmt.Errorf("failed to get Swift packages path, error %s", err)
	}

	cache := cache.New()
	cache.IncludePath(swiftPackagesDir)
	// Excluding manifest.db will result in a stable cache, as this file is modified in every build.
	cache.ExcludePath("!" + path.Join(swiftPackagesDir, "manifest.db"))

	if err := cache.Commit(); err != nil {
		return fmt.Errorf("failed to commit cache, error: %s", err)
	}
	return nil
}
