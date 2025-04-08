package xcodeversion

import (
	"fmt"
	"regexp"
	"strconv"
)

func getXcodeVersionFromXcodebuildOutput(outStr string) (Version, error) {
	versionRegexp := regexp.MustCompile(`(?m)^Xcode +(\d+)(\.(\d+))?.*$`)
	buildVersionRegexp := regexp.MustCompile(`(?m)^Build version +(\w.*)$`)

	xcodeVersionMatch := versionRegexp.FindStringSubmatch(outStr)
	if len(xcodeVersionMatch) < 4 {
		return Version{}, fmt.Errorf("couldn't find Xcode version in output: (%s)", outStr)
	}

	xcodebuildVersion := xcodeVersionMatch[0]
	majorVersionStr := xcodeVersionMatch[1]
	majorVersion, err := strconv.Atoi(majorVersionStr)
	if err != nil {
		return Version{}, fmt.Errorf("failed to parse xcodebuild major version (output %s): %w", outStr, err)
	}

	minorVersion := int(0)
	minorVersionStr := xcodeVersionMatch[3]
	if minorVersionStr != "" {
		if minorVersion, err = strconv.Atoi(minorVersionStr); err != nil {
			return Version{}, fmt.Errorf("failed to parse xcodebuild minor version (output %s): %w", outStr, err)
		}
	}

	buildVersionMatch := buildVersionRegexp.FindStringSubmatch(outStr)
	buildVersion := "unknown"
	if len(buildVersionMatch) >= 2 {
		buildVersion = buildVersionMatch[1]
	}

	return Version{
		Version:      xcodebuildVersion,
		BuildVersion: buildVersion,
		Major:        int64(majorVersion),
		Minor:        int64(minorVersion),
	}, nil
}
