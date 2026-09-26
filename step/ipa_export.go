package step

import (
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/v2/xcodecommand"
)

func runIPAExportCommand(xcodeCommandRunner xcodecommand.Runner, logFormatter string, exportCmd xcodecommand.Command, logger log.Logger) (string, error) {
	output, err := xcodeCommandRunner.Run("", exportCmd.Args(), []string{})
	if logFormatter == XcodebuildTool {
		// xcodecommand does not output to stdout for xcodebuild log formatter.
		// The export log is short, so we print it in entirety.
		logger.Printf("%s", output.RawOut)
	}

	return string(output.RawOut), err
}
