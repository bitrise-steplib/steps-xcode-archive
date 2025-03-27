package xcodecommand

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/v2/errorfinder"
	version "github.com/hashicorp/go-version"
)

const xcbeautify = "xcbeautify"

// XcbeautifyRunner is a xcodebuild runner that uses xcbeautify as log formatter
type XcbeautifyRunner struct {
	logger         log.Logger
	commandFactory command.Factory
}

// NewXcbeautifyRunner returns a new xcbeautify runner
func NewXcbeautifyRunner(logger log.Logger, commandFactory command.Factory) Runner {
	return &XcbeautifyRunner{
		logger:         logger,
		commandFactory: commandFactory,
	}
}

// Run runs xcodebuild using xcbeautify as an output formatter
func (c *XcbeautifyRunner) Run(workDir string, xcodebuildArgs []string, xcbeautifyArgs []string) (Output, error) {
	var (
		buildOutBuffer         bytes.Buffer
		pipeReader, pipeWriter = io.Pipe()
		buildOutWriter         = io.MultiWriter(&buildOutBuffer, pipeWriter)
	)

	// For parallel and concurrent destination testing, it helps to use unbuffered I/O for stdout and to redirect stderr to stdout.
	// NSUnbufferedIO=YES xcodebuild [args] 2>&1 | xcbeautify
	buildCmd := c.commandFactory.Create("xcodebuild", xcodebuildArgs, &command.Opts{
		Stdout:      buildOutWriter,
		Stderr:      buildOutWriter,
		Env:         unbufferedIOEnv,
		Dir:         workDir,
		ErrorFinder: errorfinder.FindXcodebuildErrors,
	})

	beautifyCmd := c.commandFactory.Create(xcbeautify, xcbeautifyArgs, &command.Opts{
		Stdin:  pipeReader,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Env:    unbufferedIOEnv,
	})

	defer func() {
		if err := pipeWriter.Close(); err != nil {
			c.logger.Warnf("Failed to close xcodebuild-xcbeautify pipe: %s", err)
		}

		if err := beautifyCmd.Wait(); err != nil {
			c.logger.Warnf("xcbeautify command failed: %s", err)
		}
	}()

	c.logger.TPrintf("$ set -o pipefail && %s | %s", buildCmd.PrintableCommandArgs(), beautifyCmd.PrintableCommandArgs())

	err := buildCmd.Start()
	if err == nil {
		err = beautifyCmd.Start()
	}
	if err == nil {
		err = buildCmd.Wait()
	}

	exitCode := 0
	if err != nil {
		exitCode = -1

		var exerr *exec.ExitError
		if errors.As(err, &exerr) {
			exitCode = exerr.ExitCode()
		}
	}

	return Output{
		RawOut:   buildOutBuffer.Bytes(),
		ExitCode: exitCode,
	}, err
}

// CheckInstall checks if xcbeautify is on the PATH and returns its version
func (c *XcbeautifyRunner) CheckInstall() (*version.Version, error) {
	c.logger.Println()
	c.logger.Infof("Checking log formatter (xcbeautify) version")

	versionCmd := c.commandFactory.Create(xcbeautify, []string{"--version"}, nil)

	out, err := versionCmd.RunAndReturnTrimmedOutput()
	if err != nil {
		var exerr *exec.ExitError
		if errors.As(err, &exerr) {
			return nil, fmt.Errorf("xcbeautify version command failed: %w", err)
		}

		return nil, fmt.Errorf("failed to run xcbeautify command: %w", err)
	}

	return version.NewVersion(out)
}
