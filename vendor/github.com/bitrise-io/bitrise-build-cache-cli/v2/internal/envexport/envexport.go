package envexport

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/bitrise-io/go-utils/v2/log"

	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/stringmerge"
)

// EnvExporter exports environment variables to the current process and CI-specific mechanisms.
// It sets os env vars, calls envman (Bitrise CI), and writes to GITHUB_ENV (GitHub Actions).
type EnvExporter struct {
	envs   map[string]string
	logger log.Logger
}

// New creates an EnvExporter. The envs map is used to read CI-specific file paths (e.g. GITHUB_ENV).
func New(envs map[string]string, logger log.Logger) *EnvExporter {
	return &EnvExporter{
		envs:   envs,
		logger: logger,
	}
}

// Export sets the environment variable in the current process, calls envman for Bitrise CI,
// and writes to the GITHUB_ENV file for GitHub Actions.
// All errors are logged as debug and do not fail the caller.
func (e *EnvExporter) Export(key, value string) {
	if err := os.Setenv(key, value); err != nil {
		e.logger.Debugf("Failed to set env var %s: %v", key, err)
	}

	e.exportViaEnvman(key, value)
	e.exportViaGitHubEnv(key, value)
}

func (e *EnvExporter) exportViaEnvman(key, value string) {
	output, err := exec.Command("envman", "add", //nolint:noctx
		"--key", key,
		"--value", value,
	).CombinedOutput()
	if err != nil {
		e.logger.Debugf("Failed to export %s via envman: %s (%v)", key, string(output), err)
	}
}

func (e *EnvExporter) exportViaGitHubEnv(key, value string) {
	filePath := e.envs["GITHUB_ENV"]
	if filePath == "" {
		return
	}

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644) //nolint:mnd
	if err != nil {
		e.logger.Debugf("Failed to open GITHUB_ENV file: %v", err)

		return
	}
	defer f.Close()

	writer := bufio.NewWriter(f)
	if _, err := fmt.Fprintf(writer, "%s=%s\n", key, value); err != nil {
		e.logger.Debugf("Failed to write to GITHUB_ENV file: %v", err)

		return
	}

	if err := writer.Flush(); err != nil {
		e.logger.Debugf("Failed to flush GITHUB_ENV file: %v", err)
	}
}

// ExportToShellRC writes an export statement to ~/.bashrc and ~/.zshrc using a marker block.
// The blockName identifies the block in the file (e.g. "Bitrise Build Cache").
// The content is the raw shell content to write (e.g. "export KEY=VALUE").
func (e *EnvExporter) ExportToShellRC(blockName, content string) {
	homeDir := e.envs["HOME"]
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			e.logger.Debugf("Failed to get home directory: %v", err)

			return
		}
	}

	for _, rcFile := range []string{".bashrc", ".zshrc"} {
		rcPath := filepath.Join(homeDir, rcFile)
		if err := writeShellRCBlock(rcPath, blockName, content); err != nil {
			e.logger.Debugf("Failed to update %s: %v", rcFile, err)
		}
	}
}

func writeShellRCBlock(filePath, blockName, content string) error {
	currentContent, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	newContent := stringmerge.ChangeContentInBlock(
		string(currentContent),
		fmt.Sprintf("# [start] %s", blockName),
		fmt.Sprintf("# [end] %s", blockName),
		content,
	)

	if err := os.WriteFile(filePath, []byte(newContent), 0o644); err != nil { //nolint:mnd,gosec
		return fmt.Errorf("failed to write %s: %w", filePath, err)
	}

	return nil
}
