// Package reactnative contains the marker config written by
// `bitrise-build-cache react-native activate` to signal that the React Native
// build cache is active on this machine. Consumers (the `status` command, and
// external step integrations) read this file to decide whether to engage RN
// cache wrapping.
package reactnative

import (
	"fmt"
	"path/filepath"

	"github.com/bitrise-io/go-utils/v2/log"

	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/utils"
)

const (
	reactNativePath = ".bitrise/cache/reactnative/"
	ConfigFileName  = "config.json"

	ErrFmtOpenConfigFile   = "open react-native config file (%s): %w"
	ErrFmtDecodeConfigFile = "decode react-native config file (%s): %w"
	ErrFmtCreateConfigFile = "failed to create react-native config file: %w"
	ErrFmtEncodeConfigFile = "failed to encode react-native config file: %w"
	ErrFmtCreateFolder     = "failed to create %s folder: %w"
)

// Config is the on-disk marker consumers read to decide whether RN cache
// wrapping should engage. The schema is intentionally minimal — add fields
// only when a concrete consumer needs them.
type Config struct {
	Enabled bool `json:"enabled"`
}

func DirPath(osProxy utils.OsProxy) string {
	if home, err := osProxy.UserHomeDir(); err == nil {
		return filepath.Join(home, reactNativePath)
	}

	if wd, err := osProxy.Getwd(); err == nil {
		return filepath.Join(wd, reactNativePath)
	}

	if exe, err := osProxy.Executable(); err == nil {
		if dir := filepath.Dir(exe); dir != "" {
			return filepath.Join(dir, reactNativePath)
		}
	}

	return filepath.Join(".", reactNativePath)
}

func PathFor(osProxy utils.OsProxy, subpath string) string {
	return filepath.Join(DirPath(osProxy), subpath)
}

func (c Config) Save(logger log.Logger, osProxy utils.OsProxy, encoderFactory utils.EncoderFactory) error {
	dir := DirPath(osProxy)
	if err := osProxy.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf(ErrFmtCreateFolder, dir, err)
	}

	path := PathFor(osProxy, ConfigFileName)
	f, err := osProxy.Create(path)
	if err != nil {
		return fmt.Errorf(ErrFmtCreateConfigFile, err)
	}
	defer f.Close()

	enc := encoderFactory.Encoder(f)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(c); err != nil {
		return fmt.Errorf(ErrFmtEncodeConfigFile, err)
	}

	if err := f.Sync(); err != nil {
		return fmt.Errorf("failed to sync react-native config file: %w", err)
	}

	logger.TInfof("React Native build cache marker saved to: %s", path)

	return nil
}

func ReadConfig(osProxy utils.OsProxy, decoderFactory utils.DecoderFactory) (Config, error) {
	path := PathFor(osProxy, ConfigFileName)

	f, err := osProxy.OpenFile(path, 0, 0)
	if err != nil {
		return Config{}, fmt.Errorf(ErrFmtOpenConfigFile, path, err)
	}
	defer f.Close()

	dec := decoderFactory.Decoder(f)
	var cfg Config
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf(ErrFmtDecodeConfigFile, path, err)
	}

	return cfg, nil
}
