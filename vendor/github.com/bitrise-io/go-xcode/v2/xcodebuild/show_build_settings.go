package xcodebuild

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/v2/errorfinder"
	"github.com/bitrise-io/go-xcode/v2/xcodeproject/serialized"
)

const (
	toolName = "xcodebuild"

	showBuildSettingsFlag = "-showBuildSettings"
	projectFlag           = "-project"
	workspaceFlag         = "-workspace"
	targetFlag            = "-target"
	schemeFlag            = "-scheme"
	configurationFlag     = "-configuration"

	xcworkspaceExtension = ".xcworkspace"
)

// BuildSettingsProvider returns effective build settings using xcodebuild -showBuildSettings.
type BuildSettingsProvider interface {
	TargetBuildSettings(projectPath, target, configuration string, extraArgs ...string) (serialized.Object, error)
	SchemeBuildSettings(projectPath, scheme, configuration string, extraArgs ...string) (serialized.Object, error)
}

type showBuildSettingsProvider struct {
	commandFactory command.Factory
	logger         log.Logger
}

// NewShowBuildSettingsProvider returns a BuildSettingsProvider backed by xcodebuild.
func NewShowBuildSettingsProvider(commandFactory command.Factory, logger log.Logger) BuildSettingsProvider {
	return showBuildSettingsProvider{
		commandFactory: commandFactory,
		logger:         logger,
	}
}

// TargetBuildSettings returns the effective build settings of one target.
func (p showBuildSettingsProvider) TargetBuildSettings(projectPath, target, configuration string, extraArgs ...string) (serialized.Object, error) {
	return p.run(showBuildSettingsArgs(projectPath, targetFlag, target, configuration, extraArgs))
}

// SchemeBuildSettings returns the effective build settings of one scheme.
func (p showBuildSettingsProvider) SchemeBuildSettings(projectPath, scheme, configuration string, extraArgs ...string) (serialized.Object, error) {
	return p.run(showBuildSettingsArgs(projectPath, schemeFlag, scheme, configuration, extraArgs))
}

func (p showBuildSettingsProvider) run(args []string) (serialized.Object, error) {
	// The error finder puts xcodebuild's error lines into the returned error; without it, a failure
	// with combined output only says to check the command's output.
	cmd := p.commandFactory.Create(toolName, args, &command.Opts{ErrorFinder: errorfinder.FindXcodebuildErrors})

	// Logged at normal level, so the command shows up in step logs.
	p.logger.TPrintf("Reading build settings...")
	p.logger.TDonef("$ %s", cmd.PrintableCommandArgs())

	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		// The output is captured, not logged, so it is kept when it has no line the finder recognises.
		if out != "" && len(errorfinder.FindXcodebuildErrors(out)) == 0 {
			return nil, fmt.Errorf("failed to read build settings: %w, output: %s", err, out)
		}
		return nil, fmt.Errorf("failed to read build settings: %w", err)
	}

	p.logger.TPrintf("Read target settings.")

	return parseShowBuildSettingsOutput(out)
}

func showBuildSettingsArgs(projectPath, nameFlag, name, configuration string, extraArgs []string) []string {
	var args []string

	if projectPath != "" {
		containerFlag := projectFlag
		if filepath.Ext(projectPath) == xcworkspaceExtension {
			containerFlag = workspaceFlag
		}
		args = append(args, containerFlag, projectPath)
	}

	if name != "" {
		args = append(args, nameFlag, name)
	}

	if configuration != "" {
		args = append(args, configurationFlag, configuration)
	}

	args = append(args, showBuildSettingsFlag)

	return append(args, extraArgs...)
}

// parseShowBuildSettingsOutput keeps the first occurrence of a repeated key, which for a
// multi-target scheme is the main target. ReadLine is used because values can exceed
// bufio.Scanner's line limit.
func parseShowBuildSettingsOutput(out string) (serialized.Object, error) {
	settings := serialized.Object{}

	reader := bufio.NewReader(strings.NewReader(out))
	var buffer bytes.Buffer

	for {
		fragment, isPrefix, err := reader.ReadLine()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}

		buffer.Write(fragment)

		if isPrefix {
			continue
		}

		line := strings.TrimSpace(buffer.String())
		buffer.Reset()

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"`)

		if _, exists := settings[key]; !exists {
			settings[key] = value
		}
	}

	return settings, nil
}
