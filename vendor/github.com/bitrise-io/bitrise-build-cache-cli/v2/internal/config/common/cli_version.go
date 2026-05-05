package common

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/bitrise-io/go-utils/v2/log"
)

// injectedVersion is the CLI version stamped at build time via:
//
//	-ldflags "-X github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/config/common.injectedVersion=<ver>"
//
// Goreleaser sets this on every release build. Local `go build` leaves it
// empty and we fall back to debug.ReadBuildInfo below.
//
//nolint:gochecknoglobals
var injectedVersion string

//nolint:gochecknoglobals
var versionLogOnce sync.Once

// GetCLIVersion returns the CLI version, resolved in this order:
//
//  1. The build-time injected `injectedVersion` (set by goreleaser ldflags).
//  2. When the CLI is imported as a library (e.g. from a Bitrise step),
//     the bitrise-build-cache-cli entry in debug.BuildInfo.Deps.
//  3. The main module's debug.BuildInfo Main.Version, when it's a real
//     module-aware tag (i.e. not the "(devel)" placeholder Go reports for
//     ad-hoc local builds).
//  4. "devel" as a last-resort sentinel.
//
// The logger parameter is kept for call-site compatibility; resolution is
// non-failing so it isn't used here.
func GetCLIVersion(_ log.Logger) string {
	if v := strings.TrimSpace(injectedVersion); v != "" {
		return v
	}

	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "devel"
	}

	for _, mod := range bi.Deps {
		if mod == nil {
			continue
		}

		if !strings.Contains(mod.Path, "bitrise-build-cache-cli") {
			continue
		}

		if v := strings.TrimSpace(mod.Version); v != "" {
			return v
		}
	}

	if v := strings.TrimSpace(bi.Main.Version); v != "" && v != "(devel)" {
		return v
	}

	return "devel"
}

// LogCLIVersion writes a single line with the resolved CLI version to STDERR,
// at most once per process. Subsequent calls are no-ops, which lets every
// public entry point (cobra PersistentPreRun hooks, pkg/* Activator/Runner
// methods used by step libraries) call this without producing duplicate lines.
//
// Stderr (not stdout) is intentional: some callers — e.g. xcodebuild wrappers
// fronted by `@react-native-community/cli-platform-apple` — parse the CLI's
// stdout as JSON. Writing the version line to stdout breaks that JSON parse.
func LogCLIVersion(logger log.Logger) {
	versionLogOnce.Do(func() {
		_, _ = fmt.Fprintf(os.Stderr, "Bitrise Build Cache CLI version: %s\n", GetCLIVersion(logger))
	})
}
