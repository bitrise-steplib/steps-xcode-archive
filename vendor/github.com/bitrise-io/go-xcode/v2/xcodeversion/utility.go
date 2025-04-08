package xcodeversion

import (
	"fmt"
	"strconv"
	"strings"
)

func getXcodeVersionFromXcodebuildOutput(outStr string) (Version, error) {
	split := strings.Split(outStr, "\n")
	if len(split) == 0 {
		return Version{}, fmt.Errorf("failed to parse xcodebuild version output (%s)", outStr)
	}

	filteredOutput, err := filterXcodeWarnings(split)
	if err != nil {
		return Version{}, err
	}

	xcodebuildVersion := filteredOutput[0]
	buildVersion := filteredOutput[1]

	split = strings.Split(xcodebuildVersion, " ")
	if len(split) != 2 {
		return Version{}, fmt.Errorf("failed to parse xcodebuild version output (%s)", outStr)
	}

	version := split[1]

	split = strings.Split(version, ".")
	majorVersionStr := split[0]

	majorVersion, err := strconv.ParseInt(majorVersionStr, 10, 32)
	if err != nil {
		return Version{}, fmt.Errorf("failed to parse xcodebuild version output (%s), error: %s", outStr, err)
	}

	return Version{
		Version:      xcodebuildVersion,
		BuildVersion: buildVersion,
		MajorVersion: majorVersion,
	}, nil
}

func filterXcodeWarnings(cmdOutputLines []string) ([]string, error) {
	firstLineIndex := -1
	for i, line := range cmdOutputLines {
		if strings.HasPrefix(line, "Xcode ") {
			firstLineIndex = i
			break
		}
	}

	if firstLineIndex < 0 {
		return []string{}, fmt.Errorf("couldn't find Xcode version in output: %s", cmdOutputLines)
	}

	return cmdOutputLines[firstLineIndex:], nil
}
