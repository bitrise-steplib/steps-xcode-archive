package step

import (
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/v2/xcodecommand"
	"github.com/bitrise-io/go-xcode/xcodebuild"
)

func runIPAExportCommand(xcodeCommandRunner xcodecommand.Runner, exportCmd *xcodebuild.ExportCommandModel, logger log.Logger) (string, error) {
	output, err := xcodeCommandRunner.Run("", exportCmd.CommandArgs(), []string{})
	return string(output.RawOut), wrapXcodebuildCommandError(exportCmd, string(output.RawOut), err)
}
