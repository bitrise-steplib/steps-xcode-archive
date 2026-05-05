package xcelerate

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/shirou/gopsutil/v4/process"

	configcommon "github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/config/common"
	multiplatformconfig "github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/config/multiplatform"
	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/consts"
	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/envexport"
	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/utils"
)

const (
	ActivateXcodeSuccessful = "✅ Bitrise Build Cache for Xcode activated"
	AddXcelerateToPath      = "ℹ️ To start building, run `export PATH=~/.bitrise-xcelerate/bin:$PATH` or restart your terminal."
	ErrFmtCreateXcodeConfig = "failed to create Xcode config: %w"

	cliBasename                    = "bitrise-build-cache-cli"
	xcodebuildWrapperScriptContent = `#!/bin/bash
set -e

if [ "${1-}" = "-version" ]; then
  %s "$@"
else
  %s/bitrise-build-cache-cli xcelerate xcodebuild "$@"
fi
`
	xcrunWrapperScriptContent = `#!/bin/bash
set -e

if [ "${1-}" = "xcodebuild" ] && [ "${2-}" = "-version" ]; then
  %s "$@"
elif [ "${1-}" = "xcodebuild" ]; then
  shift
  %s/bitrise-build-cache-cli xcelerate xcodebuild "$@"
else
  %s "$@"
fi
`
)

// Activate creates the Xcode build cache configuration, copies the CLI binary,
// and sets up the xcodebuild wrapper script.
func Activate(
	ctx context.Context,
	logger log.Logger,
	osProxy utils.OsProxy,
	commandFunc utils.CommandFunc,
	encoderFactory utils.EncoderFactory,
	decoderFactory utils.DecoderFactory,
	activateXcodeParams Params,
	envs map[string]string,
) error {
	overrideActivateXcodeParamsFromExistingConfig(
		logger, osProxy, &activateXcodeParams, decoderFactory, envs)

	authConfig, _ := configcommon.ReadAuthConfigFromEnvironments(envs)
	benchmarkClient := configcommon.NewBenchmarkPhaseClient(consts.BitriseWebsiteBaseURL, authConfig, logger)

	config, err := NewConfig(
		ctx,
		logger,
		activateXcodeParams,
		envs,
		osProxy,
		commandFunc,
		envexport.New(envs, logger),
		benchmarkClient,
	)
	if err != nil {
		return fmt.Errorf("failed to create xcelerate config: %w", err)
	}

	if err := config.Save(logger, osProxy, encoderFactory); err != nil {
		return fmt.Errorf(ErrFmtCreateXcodeConfig, err)
	}

	// Auth credentials are persisted only in the multiplatform analytics config
	// (single source of truth on disk). The xcelerate config carries auth in-memory
	// at runtime via ReadConfig, but never to JSON.
	mpCfg := multiplatformconfig.Config{
		AuthConfig:   config.AuthConfig,
		DebugLogging: config.DebugLogging,
	}
	if err := mpCfg.Save(osProxy, encoderFactory); err != nil {
		return fmt.Errorf("failed to save multiplatform analytics config: %w", err)
	}

	if err := copyCLIToXcelerateBinDir(ctx, osProxy, logger); err != nil {
		return fmt.Errorf("failed to copy xcelerate cli to ~/.bitrise-xcelerate/bin: %w", err)
	}

	if err := addXcelerateCommandToPathWithScriptWrapper(config, osProxy, logger, envs); err != nil {
		return fmt.Errorf("failed to add xcelerate command: %w", err)
	}

	logger.Debugf("Xcelerate command added to ~/.bashrc and ~/.zshrc")
	logger.TInfof(ActivateXcodeSuccessful)
	logger.TInfof(AddXcelerateToPath)

	return nil
}

// ---------------------------------------------------------------------------
// Private — activation helpers
// ---------------------------------------------------------------------------

func overrideActivateXcodeParamsFromExistingConfig(
	logger log.Logger,
	osProxy utils.OsProxy,
	activateXcodeParams *Params,
	decoderFactory utils.DecoderFactory,
	envs map[string]string,
) {
	if existingConfig, err := ReadConfig(osProxy, decoderFactory); err == nil {
		if strings.Contains(existingConfig.OriginalXcodebuildPath, PathFor(osProxy, BinDir)) {
			logger.Warnf("Removing xcelerate wrapper as original xcodebuild path...")
			existingConfig.OriginalXcodebuildPath = ""
		}

		activateXcodeParams.XcodePathOverride = cmp.Or(
			activateXcodeParams.XcodePathOverride,
			existingConfig.OriginalXcodebuildPath,
		)

		if strings.Contains(existingConfig.OriginalXcrunPath, PathFor(osProxy, BinDir)) {
			logger.Warnf("Removing xcelerate wrapper as original xcrun path...")
			existingConfig.OriginalXcrunPath = ""
		}

		activateXcodeParams.XcrunPathOverride = cmp.Or(
			activateXcodeParams.XcrunPathOverride,
			existingConfig.OriginalXcrunPath,
		)
	} else if isXcelerateInPath(osProxy, envs) {
		logger.Warnf("It seems that the xcelerate config file is missing, but xcelerate is already in the PATH. \n" +
			"This will lead to unexpected behavior when determining the xcodebuild path. \n" +
			"Defaulting to /usr/bin/xcodebuild...")
		activateXcodeParams.XcodePathOverride = "/usr/bin/xcodebuild"
	}
}

func isXcelerateInPath(osProxy utils.OsProxy, envs map[string]string) bool {
	path := envs["PATH"]
	for _, p := range strings.Split(path, ":") {
		if strings.Contains(p, PathFor(osProxy, BinDir)) {
			return true
		}
	}

	return false
}

func copyCLIToXcelerateBinDir(ctx context.Context, osProxy utils.OsProxy, logger log.Logger) error {
	src, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to determine executable path: %w", err)
	}

	reader, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source executable: %w", err)
	}
	defer reader.Close()

	binPath := PathFor(osProxy, BinDir)
	if err := osProxy.MkdirAll(binPath, 0o755); err != nil {
		return fmt.Errorf("failed to create bin dir: %w", err)
	}

	target := filepath.Join(binPath, cliBasename)

	if err := makeSureCLIIsNotRunning(ctx, target, logger); err != nil {
		return fmt.Errorf("failed to ensure cli is not running: %w", err)
	}

	writer, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create destination executable: %w", err)
	}
	defer writer.Close()

	if _, err = io.Copy(writer, reader); err != nil {
		return fmt.Errorf("failed to copy executable: %w", err)
	}

	logger.TInfof("Copied CLI to %s", target)

	return nil
}

func makeSureCLIIsNotRunning(ctx context.Context, target string, logger log.Logger) error {
	processes, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to list processes: %w", err)
	}

	for _, p := range processes {
		exe, err := p.ExeWithContext(ctx)
		if err != nil {
			continue
		}

		if exe != target {
			continue
		}

		logger.TWarnf("Terminating already running CLI (pid: %d)", p.Pid)

		if err := p.TerminateWithContext(ctx); err != nil {
			logger.TWarnf("Failed to terminate already running CLI, attempting to kill it")

			if err := p.KillWithContext(ctx); err != nil {
				return fmt.Errorf("failed to kill already running CLI (pid: %d): %w", p.Pid, err)
			}
		}
	}

	return nil
}

func addXcelerateCommandToPathWithScriptWrapper(
	config Config,
	osProxy utils.OsProxy,
	logger log.Logger,
	envs map[string]string,
) error {
	binPath := PathFor(osProxy, BinDir)
	if err := osProxy.MkdirAll(binPath, 0o755); err != nil {
		return fmt.Errorf("failed to create bin dir: %w", err)
	}

	scriptPath := filepath.Join(binPath, "xcodebuild")
	logger.Debugf("Creating xcodebuild wrapper script: %s", scriptPath)

	if err := osProxy.WriteFile(scriptPath,
		[]byte(fmt.Sprintf(xcodebuildWrapperScriptContent,
			config.OriginalXcodebuildPath,
			binPath)), 0o755); err != nil {
		return fmt.Errorf("failed to create xcodebuild wrapper script: %w", err)
	}

	scriptPath = filepath.Join(binPath, "xcrun")
	logger.Debugf("Creating xcrun wrapper script: %s", scriptPath)

	if err := osProxy.WriteFile(scriptPath,
		[]byte(fmt.Sprintf(xcrunWrapperScriptContent,
			config.OriginalXcrunPath,
			binPath,
			config.OriginalXcrunPath)), 0o755); err != nil {
		return fmt.Errorf("failed to create xcrun wrapper script: %w", err)
	}

	path := strings.ReplaceAll(envs["PATH"], binPath+":", "")
	path = strings.Join([]string{binPath, path}, ":")

	exporter := envexport.New(envs, logger)
	exporter.Export("PATH", path)
	exporter.ExportToShellRC("Bitrise Xcelerate", fmt.Sprintf("export PATH=%s:$PATH", binPath))

	return nil
}
