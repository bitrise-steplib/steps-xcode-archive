package xcodecommand

import (
	"errors"
	"os/exec"
	"regexp"

	"github.com/bitrise-io/go-steputils/v2/ruby"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-io/go-xcode/v2/errorfinder"
	"github.com/bitrise-io/go-xcode/v2/logio"
)

// XcprettyCommandRunner is an xcodebuild command runner that uses xcpretty as log formatter
type XcprettyCommandRunner struct {
	logger         log.Logger
	commandFactory command.Factory
	pathChecker    pathutil.PathChecker
	fileManager    fileutil.FileManager
	xcpretty       xcprettyManager // used by CheckInstall
}

// NewXcprettyCommandRunner crates a new XcprettyCommandRunner
func NewXcprettyCommandRunner(logger log.Logger, commandFactory command.Factory, pathChecker pathutil.PathChecker, fileManager fileutil.FileManager, rubyCommandFactory ruby.CommandFactory, rubyEnv ruby.Environment) Runner {
	return &XcprettyCommandRunner{
		logger:         logger,
		commandFactory: commandFactory,
		pathChecker:    pathChecker,
		fileManager:    fileManager,
		xcpretty: &xcpretty{
			commandFactory:     commandFactory,
			rubyEnv:            rubyEnv,
			rubyCommandFactory: rubyCommandFactory,
		},
	}
}

// Run runs xcodebuild using xcpretty as a log formatter
func (c *XcprettyCommandRunner) Run(workDir string, xcodebuildArgs []string, xcprettyArgs []string) (Output, error) {
	loggingIO := logio.SetupPipeWiring(regexp.MustCompile(`^\[Bitrise.*\].*`))

	c.cleanOutputFile(xcprettyArgs)

	buildCmd := c.commandFactory.Create("xcodebuild", xcodebuildArgs, &command.Opts{
		Stdout:      loggingIO.XcbuildStdout,
		Stderr:      loggingIO.XcbuildStderr,
		Env:         unbufferedIOEnv,
		Dir:         workDir,
		ErrorFinder: errorfinder.FindXcodebuildErrors,
	})

	prettyCmd := c.commandFactory.Create("xcpretty", xcprettyArgs, &command.Opts{
		Stdin:  loggingIO.ToolStdin,
		Stdout: loggingIO.ToolStdout,
		Stderr: loggingIO.ToolStderr,
	})

	defer func() {
		if err := loggingIO.Close(); err != nil {
			c.logger.Warnf("logging IO failure, error: %s", err)
		}

		if err := prettyCmd.Wait(); err != nil {
			c.logger.Warnf("xcbeautify command failed: %s", err)
		}
	}()

	c.logger.TPrintf("$ set -o pipefail && %s | %s", buildCmd.PrintableCommandArgs(), prettyCmd.PrintableCommandArgs())

	err := buildCmd.Start()
	if err == nil {
		err = prettyCmd.Start()
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
		RawOut:   loggingIO.XcbuildRawout.Bytes(),
		ExitCode: exitCode,
	}, err
}

func (c *XcprettyCommandRunner) cleanOutputFile(xcprettyArgs []string) {
	// get and delete the xcpretty output file, if exists
	xcprettyOutputFilePath := ""
	isNextOptOutputPth := false
	for _, aOpt := range xcprettyArgs {
		if isNextOptOutputPth {
			xcprettyOutputFilePath = aOpt
			break
		}
		if aOpt == "--output" {
			isNextOptOutputPth = true
			continue
		}
	}
	if xcprettyOutputFilePath != "" {
		if isExist, err := c.pathChecker.IsPathExists(xcprettyOutputFilePath); err != nil {
			c.logger.Errorf("Failed to check xcpretty output file status (path: %s): %s", xcprettyOutputFilePath, err)
		} else if isExist {
			c.logger.Warnf("=> Deleting existing xcpretty output: %s", xcprettyOutputFilePath)
			if err := c.fileManager.Remove(xcprettyOutputFilePath); err != nil {
				c.logger.Errorf("Failed to delete xcpretty output file (path: %s): %w", xcprettyOutputFilePath, err)
			}
		}
	}
}
