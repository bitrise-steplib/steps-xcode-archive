package common

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/retryhttp"
	"github.com/hashicorp/go-retryablehttp"
)

const (
	benchmarkMaxRetries = 3

	BenchmarkPhaseBaseline = "baseline"
	BenchmarkPhaseWarmup   = "warmup"

	BuildToolGradle = "gradle"
	BuildToolXcode  = "xcode"
	BuildToolBazel  = "bazel"
)

type benchmarkResponse struct {
	Phase string `json:"phase"`
}

//go:generate moq -rm -stub -pkg mocks -out ./mocks/benchmark_phase_provider.go . BenchmarkPhaseProvider

// BenchmarkPhaseProvider fetches the benchmark phase for a build.
type BenchmarkPhaseProvider interface {
	GetBenchmarkPhase(buildTool string, metadata CacheConfigMetadata) (string, error)
}

// BenchmarkPhaseClient fetches the benchmark phase for a Gradle build from the Bitrise API.
type BenchmarkPhaseClient struct {
	httpClient *retryablehttp.Client
	baseURL    string
	authConfig CacheAuthConfig
	logger     log.Logger
}

// NewBenchmarkPhaseClient creates a new BenchmarkPhaseClient.
func NewBenchmarkPhaseClient(baseURL string, authConfig CacheAuthConfig, logger log.Logger) *BenchmarkPhaseClient {
	httpClient := retryhttp.NewClient(logger)
	httpClient.RetryMax = benchmarkMaxRetries
	httpClient.HTTPClient.Timeout = 10 * time.Second

	return &BenchmarkPhaseClient{
		httpClient: httpClient,
		baseURL:    baseURL,
		authConfig: authConfig,
		logger:     logger,
	}
}

// GetBenchmarkPhase fetches the benchmark phase for the current build.
// The buildTool parameter specifies the build tool (gradle, xcode, bazel).
// Returns empty string if no benchmark phase is active or if the build can't be identified.
func (c *BenchmarkPhaseClient) GetBenchmarkPhase(buildTool string, metadata CacheConfigMetadata) (string, error) {
	params := url.Values{}

	if c.authConfig.WorkspaceID == "" {
		c.logger.Debugf("no workspace ID found, skipping benchmark phase check")

		return "", nil
	}

	if metadata.CIProvider == CIProviderBitrise {
		if metadata.BitriseAppID == "" || metadata.BitriseWorkflowName == "" {
			c.logger.Debugf("no Bitrise metadata found, skipping benchmark phase check")

			return "", nil
		}
		params.Set("app_slug", metadata.BitriseAppID)
		params.Set("workflow_name", metadata.BitriseWorkflowName)
	} else {
		if metadata.ExternalAppID == "" || metadata.ExternalWorkflowName == "" {
			c.logger.Debugf("no external IDs found, skipping benchmark phase check")

			return "", nil
		}
		params.Set("external_app_id", metadata.ExternalAppID)
		params.Set("external_workflow_name", metadata.ExternalWorkflowName)
	}

	requestURL := fmt.Sprintf("%s/build-cache/%s/invocations/%s/command_benchmark_status?%s",
		c.baseURL, c.authConfig.WorkspaceID, buildTool, params.Encode())

	req, err := retryablehttp.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.authConfig.AuthToken))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to perform HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)

		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}

	var result benchmarkResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Phase, nil
}

// BenchmarkPhaseFile is the JSON structure for the benchmark phase file.
type BenchmarkPhaseFile struct {
	Phase string `json:"phase"`
}

// BenchmarkPhaseEnvVar returns the env var name for a build tool's benchmark phase,
// e.g. "gradle" → "BITRISE_BUILD_CACHE_BENCHMARK_PHASE_GRADLE".
func BenchmarkPhaseEnvVar(buildTool string) string {
	return "BITRISE_BUILD_CACHE_BENCHMARK_PHASE_" + strings.ToUpper(buildTool)
}

// BenchmarkPhaseFilePath returns the path to the benchmark phase file for a build tool.
func BenchmarkPhaseFilePath(buildTool string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}

	return filepath.Join(homeDir, ".local", "state", "xcelerate", "benchmark", "benchmark-phase-"+buildTool+".json"), nil
}

// WriteBenchmarkPhaseFile writes the benchmark phase to a JSON file for the given build tool.
func WriteBenchmarkPhaseFile(buildTool, phase string, logger log.Logger) {
	filePath, err := BenchmarkPhaseFilePath(buildTool)
	if err != nil {
		logger.Debugf("Failed to get benchmark phase file path: %v", err)

		return
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		logger.Debugf("Failed to create benchmark phase dir: %v", err)

		return
	}

	data, err := json.Marshal(BenchmarkPhaseFile{Phase: phase})
	if err != nil {
		logger.Debugf("Failed to marshal benchmark phase file: %v", err)

		return
	}

	if err := os.WriteFile(filePath, data, 0o644); err != nil { //nolint:mnd,gosec
		logger.Debugf("Failed to write benchmark phase file: %v", err)

		return
	}

	logger.Debugf("Benchmark phase written to %s", filePath)
}

// ReadBenchmarkPhaseFile reads the benchmark phase from the JSON file for the given build tool.
// Returns empty string if the file doesn't exist or can't be read.
func ReadBenchmarkPhaseFile(buildTool string, logger log.Logger) string {
	filePath, err := BenchmarkPhaseFilePath(buildTool)
	if err != nil {
		logger.Debugf("Failed to get benchmark phase file path: %v", err)

		return ""
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		logger.Debugf("Failed to read benchmark phase file: %v", err)

		return ""
	}

	var result BenchmarkPhaseFile
	if err := json.Unmarshal(data, &result); err != nil {
		logger.Debugf("Failed to unmarshal benchmark phase file: %v", err)

		return ""
	}

	return result.Phase
}
