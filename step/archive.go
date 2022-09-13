package step

import (
	"bytes"
	"fmt"
	"os"
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
		return out, wrapXcodebuildCommandError(xcprettyCmd, out, err)
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
	out := output.String()

	return output.String(), wrapXcodebuildCommandError(archiveCmd, out, err)
}
