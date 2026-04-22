package buildcache

import (
	"github.com/bitrise-io/go-utils/v2/command"
)

// xcodebuildBinary is the command name the go-xcode xcodecommand runners use
// when invoking xcodebuild. We intercept Create calls with this name and
// rewrite them to route through `<cli> react-native run -- xcodebuild ...`.
const xcodebuildBinary = "xcodebuild"

// NewWrappingCommandFactory returns a command.Factory that forwards all calls
// to inner, except when name == "xcodebuild": those are rewritten to
// `<cliPath> react-native run -- xcodebuild <args...>` so the invocation runs
// under Bitrise Build Cache's React Native wrapper. Non-xcodebuild calls
// (xcbeautify, xcpretty, etc.) pass through untouched.
func NewWrappingCommandFactory(inner command.Factory, cliPath string) command.Factory {
	return &wrappingFactory{inner: inner, cliPath: cliPath}
}

type wrappingFactory struct {
	inner   command.Factory
	cliPath string
}

func (w *wrappingFactory) Create(name string, args []string, opts *command.Opts) command.Command {
	if name != xcodebuildBinary {
		return w.inner.Create(name, args, opts)
	}

	wrappedArgs := make([]string, 0, len(args)+4)
	wrappedArgs = append(wrappedArgs, "react-native", "run", "--", xcodebuildBinary)
	wrappedArgs = append(wrappedArgs, args...)

	return w.inner.Create(w.cliPath, wrappedArgs, opts)
}
