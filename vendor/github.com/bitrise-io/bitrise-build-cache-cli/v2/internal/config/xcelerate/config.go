package xcelerate

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/v2/log"

	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/config/common"
	multiplatformconfig "github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/config/multiplatform"
	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/utils"
)

const (
	DefaultXcodePath        = "/usr/bin/xcodebuild"
	DefaultXcrunPath        = "/usr/bin/xcrun"
	xcelerateConfigFileName = "config.json"

	ErrFmtCreateConfigFile = `failed to create xcelerate config file: %w`
	ErrFmtEncodeConfigFile = `failed to encode xcelerate config file: %w`
	ErrFmtCreateFolder     = `failed to create .xcelerate folder (%s): %w`
	ErrNoAuthConfig        = "read auth config: %w"
)

type Params struct {
	BuildCacheEnabled           bool
	BuildCacheEndpoint          string
	BuildCacheSkipFlags         bool
	DebugLogging                bool
	Silent                      bool
	XcodePathOverride           string
	XcrunPathOverride           string
	ProxySocketPathOverride     string
	PushEnabled                 bool
	XcodebuildTimestampsEnabled bool
}

// Config is the xcelerate config saved to ~/.bitrise-xcelerate/config.json.
// Note: the benchmark phase is NOT stored here — matching gradle, it is exported as
// the BITRISE_BUILD_CACHE_BENCHMARK_PHASE env var and written to
// ~/.local/state/xcelerate/benchmark/benchmark-phase.json during activation.
type Config struct {
	ProxyVersion           string `json:"proxyVersion"`
	ProxySocketPath        string `json:"proxySocketPath"`
	CLIVersion             string `json:"cliVersion"`
	WrapperVersion         string `json:"wrapperVersion"`
	OriginalXcodebuildPath string `json:"originalXcodebuildPath"`
	OriginalXcrunPath      string `json:"originalXcrunPath"`
	BuildCacheEnabled      bool   `json:"buildCacheEnabled"`
	BuildCacheSkipFlags    bool   `json:"buildCacheSkipFlags"`
	BuildCacheEndpoint     string `json:"buildCacheEndpoint"`
	PushEnabled            bool   `json:"pushEnabled"`
	DebugLogging           bool   `json:"debugLogging,omitempty"`
	Silent                 bool   `json:"silent,omitempty"`
	XcodebuildTimestamps   bool   `json:"xcodebuildTimestamps,omitempty"`
	// AuthConfig is sourced from the multiplatform analytics config at runtime
	// (single canonical source for auth credentials on disk). The JSON tag is
	// preserved for read-side backwards compatibility with older xcelerate
	// configs that still have `authConfig` on disk from a previous CLI version;
	// Save zeroes it before writing and `omitzero` keeps it out of the file.
	AuthConfig           common.CacheAuthConfig `json:"authConfig,omitzero"`
	ExternalAppID        string                 `json:"externalAppId,omitempty"`
	ExternalBuildID      string                 `json:"externalBuildId,omitempty"`
	ExternalWorkflowName string                 `json:"externalWorkflowName,omitempty"`
}

func ReadConfig(osProxy utils.OsProxy, decoderFactory utils.DecoderFactory) (Config, error) {
	configFilePath := PathFor(osProxy, xcelerateConfigFileName)

	f, err := osProxy.OpenFile(configFilePath, 0, 0)
	if err != nil {
		return Config{}, fmt.Errorf("open xcelerate config file (%s): %w", configFilePath, err)
	}
	defer f.Close()

	dec := decoderFactory.Decoder(f)
	var config Config
	if err := dec.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("decode xcelerate config file (%s): %w", configFilePath, err)
	}

	// Auth credentials live in the multiplatform analytics config. Prefer that
	// source so callers can keep using config.AuthConfig; fall back to whatever
	// the legacy xcelerate config (decoded above) carried, for users upgrading
	// from a CLI version that still persisted auth in the xcelerate config.
	if mpCfg, mpErr := multiplatformconfig.ReadConfig(osProxy, decoderFactory); mpErr == nil && mpCfg.AuthConfig.AuthToken != "" {
		config.AuthConfig = mpCfg.AuthConfig
	}

	return config, nil
}

func DefaultParams() Params {
	return Params{
		BuildCacheEnabled:           true,
		BuildCacheSkipFlags:         false,
		BuildCacheEndpoint:          "",
		Silent:                      false,
		DebugLogging:                false,
		XcodePathOverride:           "",
		ProxySocketPathOverride:     "",
		PushEnabled:                 true,
		XcodebuildTimestampsEnabled: false,
	}
}

func DefaultConfig() Config {
	return Config{}
}

func NewConfig(ctx context.Context,
	logger log.Logger,
	params Params,
	envs map[string]string,
	osProxy utils.OsProxy,
	cmdFunc utils.CommandFunc,
	exporter EnvExporter,
	benchmarkProvider common.BenchmarkPhaseProvider,
) (Config, error) {
	authConfig, err := common.ReadAuthConfigFromEnvironments(envs)
	if err != nil {
		return Config{}, fmt.Errorf(ErrNoAuthConfig, err)
	}

	metadata := common.NewMetadata(envs,
		func(name string, v ...string) (string, error) {
			output, err := exec.Command(name, v...).Output() //nolint:noctx

			return string(output), err
		},
		logger)

	// Check benchmark phase and override params if needed (only on CI).
	// The phase is exported as BITRISE_BUILD_CACHE_BENCHMARK_PHASE env var
	// and written to ~/.local/state/xcelerate/benchmark/benchmark-phase.json
	if metadata.CIProvider != "" && benchmarkProvider != nil {
		logger.Debugf("Checking benchmark phase...CI Provider: %s", metadata.CIProvider)
		ApplyBenchmarkPhase(&params, logger, benchmarkProvider, metadata, exporter)
	}

	xcodePath := params.XcodePathOverride
	if xcodePath == "" {
		logger.Debugf("No xcodebuild path override specified, determining original xcodebuild path...")
		originalXcodebuildPath, err := getOriginalXcodebuildPath(ctx, logger, cmdFunc)
		if err != nil {
			logger.Warnf("Failed to determine xcodebuild path: %s. Using default: %s", err, DefaultXcodePath)
			originalXcodebuildPath = DefaultXcodePath
		}
		xcodePath = originalXcodebuildPath
	}
	logger.Infof("Using xcodebuild path: %s. You can always override this by supplying --xcode-path.", xcodePath)

	xcrunPath := params.XcrunPathOverride
	if xcrunPath == "" {
		logger.Debugf("No xcrun path override specified, determining original xcrun path...")
		originalXcrunPath, err := getOriginalXcrunPath(ctx, logger, cmdFunc)
		if err != nil {
			logger.Warnf("Failed to determine xcrun path: %s. Using default: %s", err, DefaultXcrunPath)
			originalXcrunPath = DefaultXcrunPath
		}
		xcrunPath = originalXcrunPath
	}
	logger.Infof("Using xcrun path: %s. You can always override this by supplying --xcrun-path.", xcrunPath)

	proxySocketPath := params.ProxySocketPathOverride
	if proxySocketPath == "" {
		proxySocketPath = envs["BITRISE_XCELERATE_PROXY_SOCKET_PATH"]
		if proxySocketPath == "" {
			proxySocketPath = filepath.Join(osProxy.TempDir(), "xcelerate-proxy.sock")
			logger.Infof("Using new proxy socket path: %s", proxySocketPath)
		} else {
			logger.Infof("Using proxy socket path from environment: %s", proxySocketPath)
		}
	}

	if params.BuildCacheEndpoint == "" {
		params.BuildCacheEndpoint = common.SelectCacheEndpointURL("", envs)
	}
	logger.Infof("Using Build Cache Endpoint: %s. You can always override this by supplying --cache-endpoint.", params.BuildCacheEndpoint)

	if params.DebugLogging && params.Silent {
		logger.Warnf("Both debug and silent logging specified, silent will take precedence.")
		params.DebugLogging = false
	}
	if params.XcodebuildTimestampsEnabled && params.Silent {
		logger.Warnf("Both timestamps and silent logging specified, silent will take precedence.")
		params.XcodebuildTimestampsEnabled = false
	}

	return Config{
		ProxyVersion:           envs["BITRISE_XCELERATE_PROXY_VERSION"],
		ProxySocketPath:        proxySocketPath,
		WrapperVersion:         envs["BITRISE_XCELERATE_WRAPPER_VERSION"],
		CLIVersion:             common.GetCLIVersion(logger),
		OriginalXcodebuildPath: xcodePath,
		OriginalXcrunPath:      xcrunPath,
		BuildCacheEnabled:      params.BuildCacheEnabled,
		BuildCacheSkipFlags:    params.BuildCacheSkipFlags,
		BuildCacheEndpoint:     params.BuildCacheEndpoint,
		PushEnabled:            params.PushEnabled,
		DebugLogging:           params.DebugLogging,
		Silent:                 params.Silent,
		XcodebuildTimestamps:   params.XcodebuildTimestampsEnabled,
		AuthConfig:             authConfig,
		ExternalAppID:          metadata.ExternalAppID,
		ExternalBuildID:        metadata.ExternalBuildID,
		ExternalWorkflowName:   metadata.ExternalWorkflowName,
	}, nil
}

func getOriginalXcodebuildPath(ctx context.Context, logger log.Logger, cmdFunc utils.CommandFunc) (string, error) {
	logger.Debugf("Determining original xcodebuild path...")
	cmd := cmdFunc(ctx, "which", "xcodebuild")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get xcodebuild output: %w", err)
	}
	trimmed := strings.TrimSpace(string(output))
	if len(trimmed) == 0 {
		logger.Warnf("No xcodebuild path found, using default: %s", DefaultXcodePath)

		return DefaultXcodePath, nil
	}

	return trimmed, nil
}

func getOriginalXcrunPath(ctx context.Context, logger log.Logger, cmdFunc utils.CommandFunc) (string, error) {
	logger.Debugf("Determining original xcrun path...")
	cmd := cmdFunc(ctx, "which", "xcrun")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get xcrun output: %w", err)
	}
	trimmed := strings.TrimSpace(string(output))
	if len(trimmed) == 0 {
		logger.Warnf("No xcrun path found, using default: %s", DefaultXcrunPath)

		return DefaultXcrunPath, nil
	}

	return trimmed, nil
}

func (config Config) Save(logger log.Logger, os utils.OsProxy, encoderFactory utils.EncoderFactory) error {
	xcelerateFolder := DirPath(os)

	if err := os.MkdirAll(xcelerateFolder, 0o755); err != nil {
		return fmt.Errorf(ErrFmtCreateFolder, xcelerateFolder, err)
	}

	configFilePath := PathFor(os, xcelerateConfigFileName)
	f, err := os.Create(configFilePath)
	if err != nil {
		return fmt.Errorf(ErrFmtCreateConfigFile, err)
	}
	defer f.Close()

	// Auth credentials live in the multiplatform analytics config now. Strip
	// them before writing the xcelerate config so we don't persist a second
	// copy on disk. Older configs that still carry `authConfig` on disk are
	// tolerated on read (see ReadConfig).
	config.AuthConfig = common.CacheAuthConfig{}

	enc := encoderFactory.Encoder(f)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(config); err != nil {
		return fmt.Errorf(ErrFmtEncodeConfigFile, err)
	}

	if err := f.Sync(); err != nil {
		return fmt.Errorf("failed to sync xcelerate config file: %w", err)
	}

	logger.TInfof("Config saved to: %s", configFilePath)

	return nil
}
