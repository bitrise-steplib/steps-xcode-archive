package utils

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/command/rubycommand"
	"github.com/bitrise-io/go-utils/log"
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
func IsToolInstalled(name, version string) (bool, error) {
	return rubycommand.IsGemInstalled(name, version)
}

// IsXcprettyInstalled ...
func IsXcprettyInstalled() (bool, error) {
	return IsToolInstalled("xcpretty", "")
}

// InstallXcpretty ...
func InstallXcpretty() error {
	cmds, err := rubycommand.GemInstall("xcpretty", "")
	if err != nil {
		return fmt.Errorf("Failed to create command model, error: %s", err)
	}

	for _, cmd := range cmds {
		log.Donef("$ %s", cmd.PrintableCommandArgs())

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("Command failed, error: %s", err)
		}
	}

	return nil
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
