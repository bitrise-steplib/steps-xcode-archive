package xcpretty

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-tools/go-xcode/xcodebuild"
)

const (
	toolName = "xcpretty"
)

// CommandModel ...
type CommandModel struct {
	xcodebuildCommand xcodebuild.CommandModel

	customOptions []string
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

func (c CommandModel) cmdSlice() []string {
	slice := []string{toolName}
	slice = append(slice, c.customOptions...)
	return slice
}

// Command ...
func (c CommandModel) Command() *cmdex.CommandModel {
	cmdSlice := c.cmdSlice()
	return cmdex.NewCommand(cmdSlice[0])
}

// PrintableCmd ...
func (c CommandModel) PrintableCmd() string {
	prettyCmdSlice := c.cmdSlice()
	prettyCmdStr := cmdex.PrintableCommandArgs(false, prettyCmdSlice)

	cmdStr := c.xcodebuildCommand.PrintableCmd()

	return fmt.Sprintf("set -o pipefail && %s | %s", cmdStr, prettyCmdStr)
}

// Run ...
func (c CommandModel) Run() (string, error) {
	prettyCmd := c.Command()
	xcodebuildCmd := c.xcodebuildCommand.Command()

	// Configure cmd in- and outputs
	pipeReader, pipeWriter := io.Pipe()

	var outBuffer bytes.Buffer
	outWriter := io.MultiWriter(&outBuffer, pipeWriter)

	xcodebuildCmd.SetStdin(nil)
	xcodebuildCmd.SetStdout(outWriter)
	xcodebuildCmd.SetStderr(outWriter)

	prettyCmd.SetStdin(pipeReader)
	prettyCmd.SetStdout(os.Stdout)
	prettyCmd.SetStderr(os.Stdout)

	// Run
	if err := xcodebuildCmd.GetCmd().Start(); err != nil {
		out := outBuffer.String()
		return out, err
	}
	if err := prettyCmd.GetCmd().Start(); err != nil {
		out := outBuffer.String()
		return out, err
	}

	// Always close xcpretty outputs
	defer func() {
		if err := pipeWriter.Close(); err != nil {
			log.Warn("Failed to close xcodebuild-xcpretty pipe, error: %s", err)
		}

		if err := prettyCmd.GetCmd().Wait(); err != nil {
			log.Warn("xcpretty command failed, error: %s", err)
		}
	}()

	if err := xcodebuildCmd.GetCmd().Wait(); err != nil {
		out := outBuffer.String()
		return out, err
	}

	return outBuffer.String(), nil
}
