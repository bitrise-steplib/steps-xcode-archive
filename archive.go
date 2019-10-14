package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-xcode/xcodebuild"
	cache "github.com/bitrise-io/go-xcode/xcodecache"
	"github.com/bitrise-io/go-xcode/xcpretty"
)

func runArchiveCommandWithRetry(archiveCmd *xcodebuild.CommandBuilder, useXcpretty bool, swiftPackagesPath string) (string, error) {
	output, err := runArchiveCommand(archiveCmd, useXcpretty)
	if err != nil {
		if swiftPackagesPath != "" && strings.Contains(output, cache.SwiftPackagesStateInvalid) {
			log.RWarnf("xcode-archive", "swift-packages-cache-invalid", nil, "swift packages cache is in an invalid state")
			if err := os.RemoveAll(swiftPackagesPath); err != nil {
				log.Errorf("failed to remove Swift package caches, error: %s", err)
				return output, err
			}
			return runArchiveCommand(archiveCmd, useXcpretty)
		}
	}
	return output, err
}

func runArchiveCommand(archiveCmd *xcodebuild.CommandBuilder, useXcpretty bool) (string, error) {
	if useXcpretty {
		xcprettyCmd := xcpretty.New(archiveCmd)

		logWithTimestamp(colorstring.Green, "$ %s", xcprettyCmd.PrintableCmd())
		fmt.Println()

		if rawXcodebuildOut, err := xcprettyCmd.Run(); err != nil {
			return rawXcodebuildOut, err
		}
	} else {
		logWithTimestamp(colorstring.Green, "$ %s", archiveCmd.PrintableCmd())
		fmt.Println()

		archiveRootCmd := archiveCmd.Command()
		var output bytes.Buffer
		archiveRootCmd.SetStdout(io.MultiWriter(os.Stdout, &output))
		archiveRootCmd.SetStderr(io.MultiWriter(os.Stderr, &output))

		if err := archiveRootCmd.Run(); err != nil {
			return output.String(), err
		}
	}
	return "", nil
}