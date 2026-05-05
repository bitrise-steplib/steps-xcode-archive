package xcelerate

import (
	"path/filepath"

	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/utils"
)

const (
	xceleratePath       = ".bitrise-xcelerate/"
	BinDir              = "bin"
	ErrFmtDetermineHome = `could not determine home: %w`
)

func DirPath(osProxy utils.OsProxy) string {
	if home, err := osProxy.UserHomeDir(); err == nil {
		return filepath.Join(home, xceleratePath)
	}

	if wd, err := osProxy.Getwd(); err == nil {
		return filepath.Join(wd, xceleratePath)
	}

	if exe, err := osProxy.Executable(); err == nil {
		if dir := filepath.Dir(exe); dir != "" {
			return filepath.Join(dir, xceleratePath)
		}
	}

	// last resort
	return filepath.Join(".", xceleratePath)
}

func PathFor(osProxy utils.OsProxy, subpath string) string {
	return filepath.Join(DirPath(osProxy), subpath)
}
