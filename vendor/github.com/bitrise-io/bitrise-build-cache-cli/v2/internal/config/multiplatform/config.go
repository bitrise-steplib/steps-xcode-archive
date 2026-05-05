package multiplatform

import (
	"fmt"
	"path/filepath"

	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/config/common"
	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/utils"
)

const (
	configPath = ".bitrise/analytics/multiplatform"
	configFile = "config.json"

	ErrFmtOpenConfigFile   = "open multiplatform analytics config file (%s): %w"
	ErrFmtDecodeConfigFile = "decode multiplatform analytics config file (%s): %w"
	ErrFmtCreateConfigFile = "failed to create multiplatform analytics config file: %w"
	ErrFmtEncodeConfigFile = "failed to encode multiplatform analytics config file: %w"
	ErrFmtCreateFolder     = "failed to create %s folder: %w"
)

// Config holds the auth credentials needed by the react-native run wrapper to
// send invocation analytics. It is written by `activate react-native` and read
// by the `react-native run` command, independently of any ccache activation.
type Config struct {
	AuthConfig   common.CacheAuthConfig `json:"authConfig"`
	DebugLogging bool                   `json:"debugLogging,omitempty"`
}

func dirPath(osProxy utils.OsProxy) string {
	if home, err := osProxy.UserHomeDir(); err == nil {
		return filepath.Join(home, configPath)
	}

	if wd, err := osProxy.Getwd(); err == nil {
		return filepath.Join(wd, configPath)
	}

	return filepath.Join(".", configPath)
}

func filePath(osProxy utils.OsProxy) string {
	return filepath.Join(dirPath(osProxy), configFile)
}

// Save writes the config to disk, creating the directory if needed.
func (c Config) Save(osProxy utils.OsProxy, encoderFactory utils.EncoderFactory) error {
	dir := dirPath(osProxy)
	if err := osProxy.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf(ErrFmtCreateFolder, dir, err)
	}

	path := filePath(osProxy)
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
		return fmt.Errorf("failed to sync multiplatform config file: %w", err)
	}

	return nil
}

// ReadConfig loads the config from disk.
func ReadConfig(osProxy utils.OsProxy, decoderFactory utils.DecoderFactory) (Config, error) {
	path := filePath(osProxy)

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
