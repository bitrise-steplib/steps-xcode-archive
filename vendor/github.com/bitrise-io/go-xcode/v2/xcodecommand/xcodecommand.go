// Package xcodecommand assembles and runs xcodebuild commands.
//
// One params struct per action (ArchiveParams, BuildParams, ...) renders into a Command;
// zero-valued fields are omitted. A step's additional options are parsed (Options),
// checked against the action and laid over the derived flags; findings surface as
// Command.Diagnostics and, under Fail validation, as an error. Runner and its
// implementations run the assembled arguments through xcodebuild, xcpretty or xcbeautify.
package xcodecommand

import (
	"github.com/hashicorp/go-version"
)

// Output is the direct output of the xcodebuild command, unchanged by log formatters
type Output struct {
	RawOut   []byte
	ExitCode int
}

// Runner abstarcts an xcodebuild command runner, it can use any log formatter
type Runner interface {
	CheckInstall() (*version.Version, error)
	Run(workDir string, xcodebuildOpts []string, logFormatterOpts []string) (Output, error)
}
