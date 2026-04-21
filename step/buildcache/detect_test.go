package buildcache

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetect_NoCLIOnPath(t *testing.T) {
	emptyDir := t.TempDir()
	t.Setenv("PATH", emptyDir)

	got := Detect(context.Background(), log.NewLogger())

	assert.Equal(t, Detection{}, got)
}

func TestDetect_Enabled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script stub isn't portable to windows")
	}

	dir := t.TempDir()
	installStub(t, dir, `#!/bin/sh
if [ "$1" = "--version" ]; then
  echo "stub 1.0.0"
  exit 0
fi
if [ "$1" = "status" ] && [ "$2" = "--feature=react-native" ] && [ "$3" = "--quiet" ]; then
  exit 0
fi
exit 99
`)
	t.Setenv("PATH", dir)

	got := Detect(context.Background(), log.NewLogger())

	assert.True(t, got.ReactNativeEnabled)
	assert.Equal(t, filepath.Join(dir, "bitrise-build-cache"), got.CLIPath)
}

func TestDetect_Disabled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script stub isn't portable to windows")
	}

	dir := t.TempDir()
	installStub(t, dir, `#!/bin/sh
if [ "$1" = "--version" ]; then
  exit 0
fi
if [ "$1" = "status" ]; then
  exit 1
fi
exit 99
`)
	t.Setenv("PATH", dir)

	got := Detect(context.Background(), log.NewLogger())

	assert.False(t, got.ReactNativeEnabled)
	assert.Equal(t, filepath.Join(dir, "bitrise-build-cache"), got.CLIPath)
}

func TestDetect_VersionFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script stub isn't portable to windows")
	}

	dir := t.TempDir()
	installStub(t, dir, `#!/bin/sh
exit 42
`)
	t.Setenv("PATH", dir)

	got := Detect(context.Background(), log.NewLogger())

	// Broken CLI → zero-value (no wrap). CLIPath is left blank so callers
	// don't accidentally invoke a broken binary later.
	assert.Equal(t, Detection{}, got)
}

func TestDetect_StatusFailsUnexpectedly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script stub isn't portable to windows")
	}

	dir := t.TempDir()
	installStub(t, dir, `#!/bin/sh
if [ "$1" = "--version" ]; then
  exit 0
fi
if [ "$1" = "status" ]; then
  exit 7
fi
exit 99
`)
	t.Setenv("PATH", dir)

	got := Detect(context.Background(), log.NewLogger())

	assert.False(t, got.ReactNativeEnabled)
	// CLIPath is populated because probeCLI succeeded; only the status probe failed.
	assert.Equal(t, filepath.Join(dir, "bitrise-build-cache"), got.CLIPath)
}

func installStub(t *testing.T, dir, script string) {
	t.Helper()
	path := filepath.Join(dir, "bitrise-build-cache")
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
}
