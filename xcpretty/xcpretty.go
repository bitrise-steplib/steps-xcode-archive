package xcpretty

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"bytes"

	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-io/go-utils/log"
)

const (
	toolName = "xcpretty"
)

// Model ...
type Model struct {
	cmdToPretty *exec.Cmd
}

// New ...
func New() *Model {
	return &Model{}
}

// SetCmdToPretty ...
func (xcp *Model) SetCmdToPretty(cmd *exec.Cmd) *Model {
	xcp.cmdToPretty = cmd
	return xcp
}

func (xcp Model) commandSlice() []string {
	slice := []string{toolName}
	return slice
}

func (xcp Model) cmd() (*cmdex.CommandModel, error) {
	cmdSlice := xcp.commandSlice()
	return cmdex.NewCommandFromSlice(cmdSlice)
}

// PrintableCmd ...
func (xcp Model) PrintableCmd() string {
	prettyCmdSlice := xcp.commandSlice()
	prettyCmdStr := cmdex.PrintableCommandArgs(false, prettyCmdSlice)

	cmdSlice := xcp.cmdToPretty.Args
	cmdStr := cmdex.PrintableCommandArgs(false, cmdSlice)

	return fmt.Sprintf("set -o pipefail && %s | %s", cmdStr, prettyCmdStr)
}

// Run ...
func (xcp Model) Run() (string, error) {
	prettyCmd, err := xcp.cmd()
	if err != nil {
		return "", err
	}

	// Configure cmd in- and outputs
	pipeReader, pipeWriter := io.Pipe()

	var outBuffer bytes.Buffer
	outWriter := io.MultiWriter(&outBuffer, pipeWriter)

	xcp.cmdToPretty.Stdin = nil
	xcp.cmdToPretty.Stdout = outWriter
	xcp.cmdToPretty.Stderr = outWriter

	prettyCmd.SetStdin(pipeReader)
	prettyCmd.SetStdout(os.Stdout)
	prettyCmd.SetStderr(os.Stdout)

	// run
	if err := xcp.cmdToPretty.Start(); err != nil {
		out := outBuffer.String()
		return out, err
	}
	if err := prettyCmd.GetCmd().Start(); err != nil {
		out := outBuffer.String()
		return out, err
	}

	defer func() {
		if err := pipeWriter.Close(); err != nil {
			log.Warn("Failed to close xcodebuild-xcpretty pipe, error: %s", err)
		}

		if err := prettyCmd.GetCmd().Wait(); err != nil {
			log.Warn("xcpretty command failed, error: %s", err)
		}
	}()

	if err := xcp.cmdToPretty.Wait(); err != nil {
		out := outBuffer.String()
		return out, err
	}

	return outBuffer.String(), nil
}
