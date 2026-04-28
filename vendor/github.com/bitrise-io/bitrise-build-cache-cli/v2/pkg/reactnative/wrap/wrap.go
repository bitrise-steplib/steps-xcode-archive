// Package wrap exposes the small helper Bitrise steps use to transparently
// route a native compiler invocation (gradle / xcodebuild / ...) through the
// CLI's `react-native run -- ...` wrapper, so the compile becomes a child of
// the active React Native parent invocation.
//
// The two consumer patterns covered:
//
//  1. Direct command construction. A step that already builds an
//     argv ([]string) on its own can call Wrap to get the rewritten
//     (binary, args) pair, and pass it to whatever exec/command runner it uses.
//
//  2. command.Factory interception. A step that uses
//     github.com/bitrise-io/go-utils/v2/command can wrap its existing
//     command.Factory with NewWrappingCommandFactory, which leaves all
//     command creations untouched except for the configured target binaries
//     (e.g. `xcodebuild`, `gradlew`). Hands-off integration: no changes to
//     the call sites that build the commands.
//
// Both patterns gate on Detect: if the CLI is missing or RN cache is not
// activated on the machine, the helpers behave as identity (no wrap).
package wrap

import (
	"path/filepath"

	"github.com/bitrise-io/go-utils/v2/command"
)

const (
	// CLIBinary is the name of the bitrise-build-cache CLI binary on PATH.
	CLIBinary = "bitrise-build-cache"

	// SubcommandReactNative is the CLI subcommand that wraps an external
	// invocation as a child of the active React Native parent invocation.
	// Form: <cli> react-native run -- <binary> <args...>
	SubcommandReactNative = "react-native"
	SubcommandRun         = "run"
)

// Wrap returns the (binary, args) pair the caller should execute. When
// detection says React Native cache is active, the original (name, args) is
// rewritten to (cliPath, ["react-native", "run", "--", name, args...]). When
// it isn't, (name, args) is returned unchanged so the caller does not need a
// second branch in the call site.
//
// The original args slice is never mutated; the returned slice is a fresh
// allocation safe to mutate.
func Wrap(det Detection, name string, args []string) (string, []string) {
	if !det.ReactNativeEnabled || det.CLIPath == "" {
		// No-wrap path — return a defensive copy so callers cannot accidentally
		// observe aliasing differences between the wrap and no-wrap branches.
		out := make([]string, len(args))
		copy(out, args)

		return name, out
	}

	wrapped := make([]string, 0, len(args)+4)
	wrapped = append(wrapped, SubcommandReactNative, SubcommandRun, "--", name)
	wrapped = append(wrapped, args...)

	return det.CLIPath, wrapped
}

// NewWrappingCommandFactory returns a command.Factory that forwards every
// call to inner unchanged, except when the requested binary's basename matches
// one of wrappedBinaries — those are rewritten as `<cliPath> react-native run
// -- <name> <args...>` so the underlying invocation runs as a child of the
// React Native parent invocation.
//
// When Detection says no wrap should happen (CLI absent, probe failed, RN
// cache not activated), inner is returned directly so the call sites observe
// the same factory shape with no overhead.
//
// wrappedBinaries are matched by basename (filepath.Base) so callers that
// invoke a target via an absolute path (`/usr/bin/xcodebuild`) or a relative
// path (`./gradlew`) still get wrapped. Compare with the bare-name targets
// the caller passes (e.g. "xcodebuild", "gradlew"). An empty wrappedBinaries
// list disables interception entirely (effectively the no-wrap branch).
func NewWrappingCommandFactory(inner command.Factory, det Detection, wrappedBinaries ...string) command.Factory {
	if !det.ReactNativeEnabled || det.CLIPath == "" || len(wrappedBinaries) == 0 {
		return inner
	}

	targets := make(map[string]struct{}, len(wrappedBinaries))
	for _, b := range wrappedBinaries {
		targets[b] = struct{}{}
	}

	return &wrappingFactory{
		inner:   inner,
		cliPath: det.CLIPath,
		targets: targets,
	}
}

type wrappingFactory struct {
	inner   command.Factory
	cliPath string
	targets map[string]struct{}
}

func (w *wrappingFactory) Create(name string, args []string, opts *command.Opts) command.Command {
	if _, hit := w.targets[filepath.Base(name)]; !hit {
		return w.inner.Create(name, args, opts)
	}

	wrappedArgs := make([]string, 0, len(args)+4)
	wrappedArgs = append(wrappedArgs, SubcommandReactNative, SubcommandRun, "--", name)
	wrappedArgs = append(wrappedArgs, args...)

	return w.inner.Create(w.cliPath, wrappedArgs, opts)
}
