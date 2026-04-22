// Package buildcache detects whether the Bitrise Build Cache CLI is installed
// on this machine and whether the React Native build cache has been activated.
// The step uses the result to decide whether to wrap its xcodebuild invocation
// in `bitrise-build-cache react-native run --`.
package buildcache

import (
	"context"
	"errors"
	"os/exec"
	"time"

	"github.com/bitrise-io/go-utils/v2/log"
)

const (
	cliBinary = "bitrise-build-cache"

	lookupTimeout = 2 * time.Second
	statusTimeout = 5 * time.Second
)

// Detection describes the CLI's reachability and RN-cache activation state on
// this machine. A zero-value Detection means "no wrapping should happen" —
// either because the CLI is absent, unhealthy, or RN cache isn't activated.
type Detection struct {
	CLIPath            string
	ReactNativeEnabled bool
}

// Detect probes the CLI on PATH and queries RN-enablement. Any failure
// degrades to a zero-value Detection with a warn log — this function must
// never cause the step to fail.
func Detect(ctx context.Context, logger log.Logger) Detection {
	path, err := exec.LookPath(cliBinary)
	if err != nil {
		return Detection{}
	}

	if err := probeCLI(ctx, path); err != nil {
		logger.Warnf("Bitrise Build Cache CLI found at %s but --version failed: %s. Skipping RN cache wrap.", path, err)

		return Detection{}
	}

	enabled, err := queryRNEnabled(ctx, path)
	if err != nil {
		logger.Warnf("Bitrise Build Cache status probe failed (%s). Skipping RN cache wrap.", err)

		return Detection{CLIPath: path}
	}

	return Detection{
		CLIPath:            path,
		ReactNativeEnabled: enabled,
	}
}

func probeCLI(ctx context.Context, path string) error {
	ctx, cancel := context.WithTimeout(ctx, lookupTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, "--version")
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

// queryRNEnabled calls `<cli> status --feature=react-native --quiet`. Exit 0
// means enabled, exit 1 means disabled. Any other outcome is a probe failure.
func queryRNEnabled(ctx context.Context, path string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, statusTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, "status", "--feature=react-native", "--quiet")
	err := cmd.Run()
	if err == nil {
		return true, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.ExitCode() == 1 {
			return false, nil
		}
	}

	return false, err
}
