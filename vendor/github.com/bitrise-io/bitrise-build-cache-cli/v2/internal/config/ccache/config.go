package ccache

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/bitrise-io/go-utils/v2/log"

	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/config/common"
	multiplatformconfig "github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/config/multiplatform"
	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/utils"
)

const (
	ccachePath       = ".bitrise/cache/ccache/"
	ccacheConfigFile = "config.json"

	defaultLogFile            = "ccache-%s.log"
	defaultErrLogFile         = "ccache-err.log"
	defaultIdleTimeout        = "15m"
	defaultCRSHDataTimeout    = "2s"
	defaultCRSHRequestTimeout = "20s"

	ErrFmtOpenConfigFile   = "open ccache config file (%s): %w"
	ErrFmtDecodeConfigFile = "decode ccache config file (%s): %w"
	ErrFmtCreateConfigFile = "failed to create ccache config file: %w"
	ErrFmtEncodeConfigFile = "failed to encode ccache config file: %w"
	ErrFmtCreateFolder     = "failed to create .bitrise/cache/ccache folder (%s): %w"
	ErrNoAuthConfig        = "read auth config: %w"
)

// Params holds the parameters for creating a ccache activate config.
type Params struct {
	BuildCacheEndpoint    string
	PushEnabled           bool
	IPCSocketPathOverride string
	BaseDirOverride       string
}

type Config struct {
	LogFile            string        `json:"logFile,omitempty"`
	ErrLogFile         string        `json:"errLogFile,omitempty"`
	IPCEndpoint        string        `json:"ipcEndpoint,omitempty"`
	IdleTimeout        time.Duration `json:"idleTimeout,omitempty"`
	PushEnabled        bool          `json:"pushEnabled"`
	Enabled            bool          `json:"enabled"`
	DebugLogging       bool          `json:"debugLogging,omitempty"`
	BuildCacheEndpoint string        `json:"buildCacheEndpoint,omitempty"`

	// AuthConfig is populated at runtime from the multiplatform analytics
	// config (single canonical source for auth credentials on disk). Not
	// persisted in the ccache config JSON.
	AuthConfig common.CacheAuthConfig `json:"-"`
}

func DirPath(osProxy utils.OsProxy) string {
	if home, err := osProxy.UserHomeDir(); err == nil {
		return filepath.Join(home, ccachePath)
	}

	if wd, err := osProxy.Getwd(); err == nil {
		return filepath.Join(wd, ccachePath)
	}

	if exe, err := osProxy.Executable(); err == nil {
		if dir := filepath.Dir(exe); dir != "" {
			return filepath.Join(dir, ccachePath)
		}
	}

	return filepath.Join(".", ccachePath)
}

func PathFor(osProxy utils.OsProxy, subpath string) string {
	return filepath.Join(DirPath(osProxy), subpath)
}

func DefaultParams() Params {
	return Params{
		PushEnabled: true,
	}
}

func NewConfig(envs map[string]string, osProxy utils.OsProxy, params Params) (Config, error) {
	authConfig, err := common.ReadAuthConfigFromEnvironments(envs)
	if err != nil {
		return Config{}, fmt.Errorf(ErrNoAuthConfig, err)
	}

	ipcEndpoint := params.IPCSocketPathOverride
	if ipcEndpoint == "" {
		wd, err := osProxy.Getwd()
		if err != nil {
			wd = "."
		}
		ipcEndpoint = filepath.Join(wd, "ccache-ipc.sock")
	}

	buildCacheEndpoint := common.SelectCacheEndpointURL(params.BuildCacheEndpoint, envs)
	idleTimeout, _ := time.ParseDuration(defaultIdleTimeout)

	return Config{
		AuthConfig:         authConfig,
		IPCEndpoint:        ipcEndpoint,
		LogFile:            defaultLogFile,
		ErrLogFile:         defaultErrLogFile,
		IdleTimeout:        idleTimeout,
		PushEnabled:        params.PushEnabled,
		Enabled:            true,
		BuildCacheEndpoint: buildCacheEndpoint,
	}, nil
}

func (config Config) CRSHRemoteStorageURL() string {
	return fmt.Sprintf("crsh:%s data-timeout=%s request-timeout=%s",
		config.IPCEndpoint, defaultCRSHDataTimeout, defaultCRSHRequestTimeout)
}

func (config Config) Save(logger log.Logger, osProxy utils.OsProxy, encoderFactory utils.EncoderFactory) error {
	ccacheDir := DirPath(osProxy)
	if err := osProxy.MkdirAll(ccacheDir, 0o755); err != nil {
		return fmt.Errorf(ErrFmtCreateFolder, ccacheDir, err)
	}

	configFilePath := PathFor(osProxy, ccacheConfigFile)
	f, err := osProxy.Create(configFilePath)
	if err != nil {
		return fmt.Errorf(ErrFmtCreateConfigFile, err)
	}
	defer f.Close()

	enc := encoderFactory.Encoder(f)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(config); err != nil {
		return fmt.Errorf(ErrFmtEncodeConfigFile, err)
	}

	if err := f.Sync(); err != nil {
		return fmt.Errorf("failed to sync ccache config file: %w", err)
	}

	logger.TInfof("Config saved to: %s", configFilePath)

	return nil
}

func ReadConfig(osProxy utils.OsProxy, decoderFactory utils.DecoderFactory) (Config, error) {
	configFilePath := PathFor(osProxy, ccacheConfigFile)

	f, err := osProxy.OpenFile(configFilePath, 0, 0)
	if err != nil {
		return Config{}, fmt.Errorf(ErrFmtOpenConfigFile, configFilePath, err)
	}
	defer f.Close()

	dec := decoderFactory.Decoder(f)
	var config Config
	if err := dec.Decode(&config); err != nil {
		return Config{}, fmt.Errorf(ErrFmtDecodeConfigFile, configFilePath, err)
	}

	// Auth credentials live in the multiplatform analytics config. Populate the
	// in-memory AuthConfig from there so callers can keep using config.AuthConfig.
	// Guard against an empty/decoded-but-unauthenticated multiplatform config —
	// matches the xcelerate fallback so we don't silently wipe credentials when
	// the file exists but carries no token.
	if mpCfg, mpErr := multiplatformconfig.ReadConfig(osProxy, decoderFactory); mpErr == nil && mpCfg.AuthConfig.AuthToken != "" {
		config.AuthConfig = mpCfg.AuthConfig
	}

	return config, nil
}
