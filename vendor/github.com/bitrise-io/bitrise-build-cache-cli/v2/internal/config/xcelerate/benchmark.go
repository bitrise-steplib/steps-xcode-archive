package xcelerate

import (
	"github.com/bitrise-io/go-utils/v2/log"

	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/config/common"
)

// EnvExporter abstracts environment variable export for testability.
type EnvExporter interface {
	Export(key, value string)
	ExportToShellRC(blockName, content string)
}

// ApplyBenchmarkPhase queries the benchmark phase and overrides xcode params accordingly.
// Baseline phase disables cache. Warmup phase logs a warning.
// The phase is exported as BITRISE_BUILD_CACHE_BENCHMARK_PHASE env var
// and written to ~/.local/state/xcelerate/benchmark/benchmark-phase.json as fallback.
func ApplyBenchmarkPhase(
	params *Params,
	logger log.Logger,
	benchmarkProvider common.BenchmarkPhaseProvider,
	metadata common.CacheConfigMetadata,
	exporter EnvExporter,
) {
	phase, err := benchmarkProvider.GetBenchmarkPhase(common.BuildToolXcode, metadata)
	if err != nil {
		logger.Debugf("Failed to fetch benchmark phase, using configured flags: %v", err)

		return
	}

	if phase == "" {
		logger.Debugf("No benchmark phase found, using configured flags")

		return
	}

	envVar := common.BenchmarkPhaseEnvVar(common.BuildToolXcode)
	logger.Infof("(i) Benchmark phase: %s", phase)
	exporter.Export(envVar, phase)
	exporter.ExportToShellRC("Bitrise Benchmark Phase", "export "+envVar+"="+phase)
	common.WriteBenchmarkPhaseFile(common.BuildToolXcode, phase, logger)

	switch phase {
	case common.BenchmarkPhaseBaseline:
		logger.Warnf("Benchmark baseline mode: disabling cache and enabling analytics only")
		params.BuildCacheEnabled = false
	case common.BenchmarkPhaseWarmup:
		logger.Infof("(i) Benchmark warmup phase: cache performance might not be ideal")
	}
}
