package step

import (
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/v2/xcodecommand"
	"github.com/bitrise-io/go-xcode/xcodebuild"
)

func runIPAExportCommand(xcodeCommandRunner xcodecommand.Runner, exportCmd *xcodebuild.ExportCommandModel, logger log.Logger) (string, error) {
	exportCommandArgs := exportCmd.Command().GetCmd().Args
	if len(exportCommandArgs) <= 1 {
		panic("ToDo should not happen")
	}

	output, err := xcodeCommandRunner.Run(".", exportCommandArgs[1:], []string{})
	return string(output.RawOut), wrapXcodebuildCommandError(exportCmd, string(output.RawOut), err)
}
