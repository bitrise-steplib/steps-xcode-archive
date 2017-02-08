package utils

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	version "github.com/hashicorp/go-version"
)

// XcodebuildVersionModel ...
type XcodebuildVersionModel struct {
	XcodeVersion *version.Version
	BuildVersion string
}

func createXcodeVersionFromOutput(versionOut string) (XcodebuildVersionModel, error) {
	split := strings.Split(versionOut, "\n")
	if len(split) != 2 {
		return XcodebuildVersionModel{}, fmt.Errorf("invalid version output: %s", versionOut)
	}

	xcodeVersionStr := strings.TrimPrefix(split[0], "Xcode ")
	xcodeVersion, err := version.NewVersion(xcodeVersionStr)
	if err != nil {
		return XcodebuildVersionModel{}, fmt.Errorf("failed to parse xcode version (%s), error: %s", xcodeVersionStr, err)
	}

	buildVersion := strings.TrimPrefix(split[1], "Build version ")

	return XcodebuildVersionModel{
		XcodeVersion: xcodeVersion,
		BuildVersion: buildVersion,
	}, nil
}

// XcodeBuildVersion ...
func XcodeBuildVersion() (XcodebuildVersionModel, error) {
	cmd := command.New("xcodebuild", "-version")
	versionOut, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return XcodebuildVersionModel{}, err
	}

	return createXcodeVersionFromOutput(versionOut)
}

// IsToolInstalled ...
func IsToolInstalled(name string) bool {
	cmd := command.New("which", name)
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	return err == nil && out != ""
}

// IsXcprettyInstalled ...
func IsXcprettyInstalled() bool {
	return IsToolInstalled("xcpretty")
}

func parseXcprettyVersionOut(versionOut string) (*version.Version, error) {
	return version.NewVersion(versionOut)
}

// XcprettyVersion ...
func XcprettyVersion() (*version.Version, error) {
	cmd := command.New("xcpretty", "--version")
	versionOut, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return nil, err
	}

	return parseXcprettyVersionOut(versionOut)
}
