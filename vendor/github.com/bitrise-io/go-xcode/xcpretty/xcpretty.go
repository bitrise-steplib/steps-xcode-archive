package xcpretty

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/bitrise-io/go-steputils/ruby"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/env"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-xcode/xcodebuild"
	version "github.com/hashicorp/go-version"
)

const (
	toolName = "xcpretty"
)

// CommandModel ...
type CommandModel struct {
	xcodebuildCommand xcodebuild.CommandModel
	customOptions     []string
}

// New ...
func New(xcodebuildCommand xcodebuild.CommandModel) *CommandModel {
	return &CommandModel{
		xcodebuildCommand: xcodebuildCommand,
	}
}

// SetCustomOptions ...
func (c *CommandModel) SetCustomOptions(customOptions []string) *CommandModel {
	c.customOptions = customOptions
	return c
}

// Command ...
func (c CommandModel) Command(opts *command.Opts) command.Command {
	return command.NewFactory(env.NewRepository()).Create(toolName, c.customOptions, opts)
}

// PrintableCmd ...
func (c CommandModel) PrintableCmd() string {
	prettyCmdStr := c.Command(nil).PrintableCommandArgs()
	xcodebuildCmdStr := c.xcodebuildCommand.PrintableCmd()

	return fmt.Sprintf("set -o pipefail && %s | %s", xcodebuildCmdStr, prettyCmdStr)
}

// Run ...
func (c CommandModel) Run() (string, error) {

	// Configure cmd in- and outputs
	pipeReader, pipeWriter := io.Pipe()

	var outBuffer bytes.Buffer
	outWriter := io.MultiWriter(&outBuffer, pipeWriter)

	xcodebuildCmd := c.xcodebuildCommand.Command(&command.Opts{
		Stdin:  nil,
		Stdout: outWriter,
		Stderr: outWriter,
	})

	prettyCmd := c.Command(&command.Opts{
		Stdin:  pipeReader,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})

	// Run
	if err := xcodebuildCmd.Start(); err != nil {
		out := outBuffer.String()
		return out, err
	}
	if err := prettyCmd.Start(); err != nil {
		out := outBuffer.String()
		return out, err
	}

	// Always close xcpretty outputs
	defer func() {
		if err := pipeWriter.Close(); err != nil {
			log.Warnf("Failed to close xcodebuild-xcpretty pipe, error: %s", err)
		}

		if err := prettyCmd.Wait(); err != nil {
			log.Warnf("xcpretty command failed, error: %s", err)
		}
	}()

	if err := xcodebuildCmd.Wait(); err != nil {
		out := outBuffer.String()
		return out, err
	}

	return outBuffer.String(), nil
}

// Xcpretty ...
type Xcpretty interface {
	IsInstalled() (bool, error)
	Install() ([]command.Command, error)
	Version() (*version.Version, error)
}

type xcpretty struct {
}

// NewXcpretty ...
func NewXcpretty() Xcpretty {
	return &xcpretty{}
}

func (x xcpretty) IsInstalled() (bool, error) {
	locator := env.NewCommandLocator()
	factory, err := ruby.NewCommandFactory(command.NewFactory(env.NewRepository()), locator)
	if err != nil {
		return false, err
	}

	return ruby.NewEnvironment(factory, locator).IsGemInstalled("xcpretty", "")
}

// Install ...
func (x xcpretty) Install() ([]command.Command, error) {
	locator := env.NewCommandLocator()
	factory, err := ruby.NewCommandFactory(command.NewFactory(env.NewRepository()), locator)
	if err != nil {
		return nil, err
	}

	cmds := factory.CreateGemInstall("xcpretty", "", false, false, nil)

	return cmds, nil
}

// Version ...
func (x xcpretty) Version() (*version.Version, error) {
	cmd := command.NewFactory(env.NewRepository()).Create("xcpretty", []string{"--version"}, nil)
	versionOut, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return nil, err
	}

	return version.NewVersion(versionOut)
}
