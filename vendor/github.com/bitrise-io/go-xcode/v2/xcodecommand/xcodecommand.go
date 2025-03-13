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
