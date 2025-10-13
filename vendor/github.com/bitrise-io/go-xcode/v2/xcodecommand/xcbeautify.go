package xcodecommand

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/v2/errorfinder"
	"github.com/bitrise-io/go-xcode/v2/logio"
	version "github.com/hashicorp/go-version"
)

const (
	xcbeautify = "xcbeautify"
)

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
	loggingIO := logio.SetupPipeWiring(regexp.MustCompile(`^\[Bitrise.*\].*`))

	// For parallel and concurrent destination testing, it helps to use unbuffered I/O for stdout and to redirect stderr to stdout.
	// NSUnbufferedIO=YES xcodebuild [args] 2>&1 | xcbeautify
	buildCmd := c.commandFactory.Create("xcodebuild", xcodebuildArgs, &command.Opts{
		Stdout:      loggingIO.XcbuildStdout,
		Stderr:      loggingIO.XcbuildStderr,
		Env:         unbufferedIOEnv,
		Dir:         workDir,
		ErrorFinder: errorfinder.FindXcodebuildErrors,
	})

	beautifyCmd := c.commandFactory.Create(xcbeautify, xcbeautifyArgs, &command.Opts{
		Stdin:  loggingIO.ToolStdin,
		Stdout: loggingIO.ToolStdout,
		Stderr: loggingIO.ToolStderr,
		Env:    unbufferedIOEnv,
	})

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

	if err := loggingIO.Close(); err != nil {
		c.logger.Warnf("logging IO failure, error: %s", err)
	}

	if err := beautifyCmd.Wait(); err != nil {
		c.logger.Warnf("xcbeautify command failed: %s", err)
	}

	return Output{
		RawOut:   loggingIO.XcbuildRawout.Bytes(),
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
