package wrap

import (
	"context"
	"os"
	"os/exec"

	"github.com/bitrise-io/bitrise-build-cache-cli/v2/pkg/status"
)

// OptOutEnv, when set to "0", skips detection entirely and returns a
// zero-value Detection. Killswitch for operators if the wrapper ever ships
// a regression — set BITRISE_BUILD_CACHE_RN_WRAP=0 on the affected build to
// force the no-wrap path without rolling back.
const OptOutEnv = "BITRISE_BUILD_CACHE_RN_WRAP"

// Detection describes the CLI's reachability and RN-cache activation state on
// this machine. A zero-value Detection means "no wrapping should happen" —
// either because the CLI is absent or RN cache isn't activated.
type Detection struct {
	// CLIPath is the absolute path of the bitrise-build-cache binary on PATH.
	// Empty when the CLI is not installed (or the OptOutEnv killswitch is set).
	CLIPath string

	// ReactNativeEnabled reports whether the CLI considers the React Native
	// build cache active on this machine. Only true when CLIPath is also set.
	ReactNativeEnabled bool
}

// Logger is the small subset of github.com/bitrise-io/go-utils/v2/log.Logger
// this package needs. Any go-utils logger satisfies it implicitly, and tests
// can implement it without stubbing the full Logger surface.
type Logger interface {
	Warnf(format string, args ...any)
	Debugf(format string, args ...any)
}

// DetectParams configures Detect. The zero value uses production defaults
// (real PATH lookup, in-process status check via pkg/status) and a no-op
// logger.
type DetectParams struct {
	// Logger receives a debug line on each skip path. Nil → silent.
	Logger Logger

	// LookPath overrides exec.LookPath. Useful for tests.
	LookPath func(file string) (string, error)

	// IsReactNativeEnabled overrides the in-process RN-activation check.
	// Production default reads the activation marker via pkg/status. Useful
	// for tests that want to drive the enabled/disabled branches without
	// touching the filesystem.
	IsReactNativeEnabled func() bool

	// Getenv overrides os.Getenv. Useful for tests; nil → os.Getenv.
	Getenv func(key string) string
}

// Detect probes the CLI on PATH and queries RN-enablement in-process via
// pkg/status. Any failure degrades to a zero-value Detection (with a debug
// log when applicable) — this function never returns an error so callers can
// drop it straight into a command-construction site without adding
// error-handling branches.
//
// In-process status check (rather than execing `<cli> status …`) is
// deliberate: the wrap pkg is part of the CLI module, so the activation
// marker reader is the same code the binary on PATH runs when versions
// match. The activation-marker layout under pkg/status is treated as a
// stable public contract for step binaries that pin older CLI versions —
// see pkg/status for the format guarantees.
//
// ctx is accepted for API stability; the in-process check has no I/O to
// cancel.
func Detect(ctx context.Context, params DetectParams) Detection {
	_ = ctx

	getenv := params.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}

	if getenv(OptOutEnv) == "0" {
		debug(params.Logger, "Bitrise Build Cache RN wrap: %s=0 set, skipping detection.", OptOutEnv)

		return Detection{}
	}

	lookPath := params.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	path, err := lookPath(CLIBinary)
	if err != nil {
		debug(params.Logger, "Bitrise Build Cache RN wrap: %s not on PATH, skipping (%v).", CLIBinary, err)

		return Detection{}
	}

	isEnabled := params.IsReactNativeEnabled
	if isEnabled == nil {
		isEnabled = defaultRNEnabledChecker
	}

	enabled := isEnabled()
	if !enabled {
		debug(params.Logger, "Bitrise Build Cache RN wrap: CLI at %s reports react-native cache not activated, skipping wrap.", path)
	}

	return Detection{
		CLIPath:            path,
		ReactNativeEnabled: enabled,
	}
}

// defaultRNEnabledChecker queries pkg/status for the react-native feature.
// Errors degrade to "disabled" so Detect never returns spurious enable.
func defaultRNEnabledChecker() bool {
	enabled, err := status.NewChecker(status.CheckerParams{}).IsEnabled(status.FeatureReactNative)
	if err != nil {
		return false
	}

	return enabled
}

func debug(logger Logger, format string, args ...any) {
	if logger == nil {
		return
	}

	logger.Debugf(format, args...)
}
