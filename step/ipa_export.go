package step

import (
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/xcodebuild"
	v1xcpretty "github.com/bitrise-io/go-xcode/xcpretty"
)

func runIPAExportCommand(exportCmd *xcodebuild.ExportCommandModel, useXcpretty bool, logger log.Logger) (string, error) {
	if useXcpretty {
		xcprettyCmd := v1xcpretty.New(exportCmd)

		logger.TDonef("$ %s", xcprettyCmd.PrintableCmd())
		logger.Println()

		out, err := xcprettyCmd.Run()
		return out, wrapXcodebuildCommandError(xcprettyCmd, out, err)
	}

	// Using xcodebuild
	logger.TDonef("$ %s", exportCmd.PrintableCmd())
	logger.Println()

	out, err := exportCmd.RunAndReturnOutput()
	return out, wrapXcodebuildCommandError(exportCmd, out, err)
}
