package xcodebuild

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-xcode/xcodeproject/serialized"
)

// ShowBuildSettingsCommandModel ...
type ShowBuildSettingsCommandModel struct {
	commandFactory command.Factory

	projectPath   string
	isWorkspace   bool
	target        string
	scheme        string
	configuration string
	customOptions []string
}

// NewShowBuildSettingsCommand ...
func NewShowBuildSettingsCommand(projectPath string, isWorkspace bool, target, scheme, configuration string, customOptions []string, commandFactory command.Factory) *ShowBuildSettingsCommandModel {
	return &ShowBuildSettingsCommandModel{
		commandFactory: commandFactory,
		projectPath:    projectPath,
		isWorkspace:    isWorkspace,
		target:         target,
		scheme:         scheme,
		configuration:  configuration,
		customOptions:  customOptions,
	}
}

func (c *ShowBuildSettingsCommandModel) args() []string {
	var slice []string

	if c.projectPath != "" {
		if c.isWorkspace {
			slice = append(slice, "-workspace", c.projectPath)
		} else {
			slice = append(slice, "-project", c.projectPath)
		}
	}

	if c.target != "" {
		slice = append(slice, "-target", c.target)
	}

	if c.scheme != "" {
		slice = append(slice, "-scheme", c.scheme)
	}

	if c.configuration != "" {
		slice = append(slice, "-configuration", c.configuration)
	}

	slice = append(slice, c.customOptions...)
	slice = append(slice, "-showBuildSettings")

	return slice
}

// Command ...
func (c ShowBuildSettingsCommandModel) Command(opts *command.Opts) command.Command {
	return c.commandFactory.Create(toolName, c.args(), opts)
}

// PrintableCmd ...
func (c ShowBuildSettingsCommandModel) PrintableCmd() string {
	return c.Command(nil).PrintableCommandArgs()
}

func parseBuildSettings(out string) (serialized.Object, error) {
	settings := serialized.Object{}

	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if split := strings.Split(line, "="); len(split) == 2 {
			key := strings.TrimSpace(split[0])
			value := strings.TrimSpace(split[1])
			value = strings.Trim(value, `"`)

			settings[key] = value
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return settings, nil
}

// RunAndReturnSettings ...
func (c ShowBuildSettingsCommandModel) RunAndReturnSettings() (serialized.Object, error) {
	cmd := c.Command(nil)
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		if errorutil.IsExitStatusError(err) {
			return nil, fmt.Errorf("%s command failed: output: %s", cmd.PrintableCommandArgs(), out)
		}
		return nil, fmt.Errorf("failed to run command %s: %s", cmd.PrintableCommandArgs(), err)
	}

	return parseBuildSettings(out)
}
