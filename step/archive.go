package step

import (
	"fmt"
	"os"
	"strings"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/v2/xcodecommand"
	"github.com/bitrise-io/go-xcode/xcodebuild"
	cache "github.com/bitrise-io/go-xcode/xcodecache"
)

func runArchiveCommandWithRetry(xcodeCommandRunner xcodecommand.Runner, logFormatter string, archiveCmd *xcodebuild.CommandBuilder, swiftPackagesPath string, logger log.Logger) (string, error) {
	output, err := runArchiveCommand(xcodeCommandRunner, logFormatter, archiveCmd, logger)
	if err != nil && swiftPackagesPath != "" && strings.Contains(output, cache.SwiftPackagesStateInvalid) {
		logger.Warnf("Archive failed, swift packages cache is in an invalid state, error: %s", err)
		if err := os.RemoveAll(swiftPackagesPath); err != nil {
			return output, fmt.Errorf("failed to remove invalid Swift package caches, error: %s", err)
		}
		return runArchiveCommand(xcodeCommandRunner, logFormatter, archiveCmd, logger)
	}
	return output, err
}

func runArchiveCommand(xcodeCommandRunner xcodecommand.Runner, logFormatter string, archiveCmd *xcodebuild.CommandBuilder, logger log.Logger) (string, error) {
	output, err := xcodeCommandRunner.Run("", archiveCmd.CommandArgs(), []string{})
	if logFormatter == XcodebuildTool || err != nil {
		printLastLinesOfXcodebuildLog(logger, string(output.RawOut), err == nil)
	}

	return string(output.RawOut), err
}
