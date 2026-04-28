package wrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

const (
	// OptOutEnv, when set to "0", skips detection entirely and returns a
	// zero-value Detection. Killswitch for operators if the wrapper ever
	// ships a regression — set BITRISE_BUILD_CACHE_RN_WRAP=0 on the affected
	// build to force the no-wrap path without rolling back.
	OptOutEnv = "BITRISE_BUILD_CACHE_RN_WRAP"

	// DefaultLookupTimeout caps the `<cli> --version` reachability probe.
	DefaultLookupTimeout = 2 * time.Second

	// DefaultStatusTimeout caps the `<cli> status --feature=react-native` probe.
	DefaultStatusTimeout = 5 * time.Second
)

// Detection describes the CLI's reachability and RN-cache activation state on
// this machine. A zero-value Detection means "no wrapping should happen" —
// either because the CLI is absent, unhealthy, or RN cache isn't activated.
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
// (real PATH lookup, real exec, the default timeouts) and a no-op logger.
type DetectParams struct {
	// Logger receives a warn line if the CLI is found but its probe fails,
	// and a debug line on each skip path. Nil → silent.
	Logger Logger

	// LookPath overrides exec.LookPath. Useful for tests.
	LookPath func(file string) (string, error)

	// CommandContext overrides exec.CommandContext. Useful for tests; signature
	// matches the stdlib so tests can inject a fake binary.
	CommandContext func(ctx context.Context, name string, args ...string) *exec.Cmd

	// LookupTimeout caps the `<cli> --version` probe. Zero → DefaultLookupTimeout.
	LookupTimeout time.Duration

	// StatusTimeout caps the RN-status probe. Zero → DefaultStatusTimeout.
	StatusTimeout time.Duration

	// Getenv overrides os.Getenv. Useful for tests; nil → os.Getenv.
	Getenv func(key string) string
}

// Detect probes the CLI on PATH and queries RN-enablement. Any failure
// degrades to a zero-value Detection (with a warn log when applicable) — this
// function never returns an error so callers can drop it straight into a
// command-construction site without adding error-handling branches.
func Detect(ctx context.Context, params DetectParams) Detection {
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

	commandContext := params.CommandContext
	if commandContext == nil {
		commandContext = exec.CommandContext
	}

	lookupTimeout := params.LookupTimeout
	if lookupTimeout <= 0 {
		lookupTimeout = DefaultLookupTimeout
	}

	statusTimeout := params.StatusTimeout
	if statusTimeout <= 0 {
		statusTimeout = DefaultStatusTimeout
	}

	path, err := lookPath(CLIBinary)
	if err != nil {
		debug(params.Logger, "Bitrise Build Cache RN wrap: %s not on PATH, skipping (%v).", CLIBinary, err)

		return Detection{}
	}

	if err := probeCLI(ctx, commandContext, path, lookupTimeout); err != nil {
		warn(params.Logger, "Bitrise Build Cache CLI found at %s but --version failed: %s. Skipping RN cache wrap.", path, err)

		return Detection{}
	}

	enabled, err := queryRNEnabled(ctx, commandContext, path, statusTimeout)
	if err != nil {
		warn(params.Logger, "Bitrise Build Cache status probe failed (%s). Skipping RN cache wrap.", err)

		return Detection{CLIPath: path}
	}

	if !enabled {
		debug(params.Logger, "Bitrise Build Cache RN wrap: CLI at %s reports react-native cache not activated, skipping wrap.", path)
	}

	return Detection{
		CLIPath:            path,
		ReactNativeEnabled: enabled,
	}
}

func probeCLI(ctx context.Context, commandContext func(context.Context, string, ...string) *exec.Cmd, path string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := commandContext(ctx, path, "--version").Run(); err != nil {
		return fmt.Errorf("run --version probe: %w", err)
	}

	return nil
}

// queryRNEnabled calls `<cli> status --feature=react-native --quiet`. Exit 0
// means enabled, exit 1 means disabled. Any other outcome is a probe failure.
func queryRNEnabled(ctx context.Context, commandContext func(context.Context, string, ...string) *exec.Cmd, path string, timeout time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := commandContext(ctx, path, "status", "--feature=react-native", "--quiet").Run()
	if err == nil {
		return true, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}

	return false, fmt.Errorf("run status probe: %w", err)
}

func warn(logger Logger, format string, args ...any) {
	if logger == nil {
		return
	}

	logger.Warnf(format, args...)
}

func debug(logger Logger, format string, args ...any) {
	if logger == nil {
		return
	}

	logger.Debugf(format, args...)
}
