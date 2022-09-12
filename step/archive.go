package step

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/progress"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/xcodebuild"
	cache "github.com/bitrise-io/go-xcode/xcodecache"
	"github.com/bitrise-io/go-xcode/xcpretty"
)

func runArchiveCommandWithRetry(archiveCmd *xcodebuild.CommandBuilder, useXcpretty bool, swiftPackagesPath string, logger log.Logger) (string, error) {
	output, err := runArchiveCommand(archiveCmd, useXcpretty, logger)
	if err != nil && swiftPackagesPath != "" && strings.Contains(output, cache.SwiftPackagesStateInvalid) {
		logger.Warnf("Archive failed, swift packages cache is in an invalid state, error: %s", err)
		if err := os.RemoveAll(swiftPackagesPath); err != nil {
			return output, fmt.Errorf("failed to remove invalid Swift package caches, error: %s", err)
		}
		return runArchiveCommand(archiveCmd, useXcpretty, logger)
	}
	return output, err
}

func runArchiveCommand(archiveCmd *xcodebuild.CommandBuilder, useXcpretty bool, logger log.Logger) (string, error) {
	if useXcpretty {
		xcprettyCmd := xcpretty.New(archiveCmd)

		logger.TDonef("$ %s", xcprettyCmd.PrintableCmd())
		logger.Println()

		out, err := xcprettyCmd.Run()
		return out, wrapXcodebuildCommandError(xcprettyCmd, err)
	}
	// Using xcodebuild
	logger.TDonef("$ %s", archiveCmd.PrintableCmd())
	logger.Println()

	var output bytes.Buffer
	archiveRootCmd := archiveCmd.Command()
	archiveRootCmd.SetStdout(&output)
	archiveRootCmd.SetStderr(&output)

	var err error
	progress.SimpleProgress(".", time.Minute, func() {
		err = archiveRootCmd.Run()
	})

	return output.String(), wrapXcodebuildCommandError(archiveCmd, err)
}

func wrapXcodebuildCommandError(cmd xcodebuild.CommandModel, err error) error {
	if err == nil {
		return nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return fmt.Errorf("command (%s) failed with exit status %d", cmd.PrintableCmd(), exitErr.ExitCode())
	}

	return fmt.Errorf("executing command (%s) failed: %w", cmd.PrintableCmd(), err)
}
