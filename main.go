package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/bitrise-io/go-steputils/v2/ruby"
	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/errorutil"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-io/go-xcode/v2/xcodecommand"
	"github.com/bitrise-steplib/steps-xcode-archive/step"
)

func main() {
	os.Exit(run())
}

func run() int {
	logger := log.NewLogger()
	configParser := createConfigParser(logger)
	config, err := configParser.ProcessInputs()
	if err != nil {
		logger.Errorf("%s", errorutil.FormattedError(fmt.Errorf("Failed to process Step inputs: %w", err)))
		return 1
	}

	archiver, err := createXcodebuildArchiver(logger, config.LogFormatter)
	if err != nil {
		logger.Errorf("%s", errorutil.FormattedError(fmt.Errorf("Failed to process Step inputs: %w", err)))
		return 1
	}

	if err := archiver.EnsureDependencies(); err != nil {
		var logFormatterErr step.LogFormatterErr
		if errors.As(err, &logFormatterErr) {
			config.LogFormatter = step.XcodebuildTool
		} else {
			logger.Errorf("%s", errorutil.FormattedError(fmt.Errorf("Failed to ensure dependencies: %w", err)))
		}
	}

	exitCode := 0
	runOpts := createRunOptions(config)
	result, err := archiver.Run(runOpts)
	if err != nil {
		logger.Errorf("%s", errorutil.FormattedError(fmt.Errorf("Failed to execute Step main logic: %w", err)))
		exitCode = 1
		// don't return as step outputs needs to be exported even in case of failure (for example the xcodebuild logs)
	}

	exportOpts := createExportOptions(config, result)
	if err := archiver.ExportOutput(exportOpts); err != nil {
		logger.Errorf("%s", errorutil.FormattedError(fmt.Errorf("Failed to export Step outputs: %w", err)))
		return 1
	}

	return exitCode
}

func createConfigParser(logger log.Logger) step.XcodebuildArchiveConfigParser {
	envRepository := env.NewRepository()
	inputParser := stepconf.NewInputParser(envRepository)
	xcodeVersionProvider := step.NewXcodebuildXcodeVersionProvider()
	fileManager := fileutil.NewFileManager()
	cmdFactory := command.NewFactory(envRepository)

	return step.NewXcodeArchiveConfigParser(inputParser, xcodeVersionProvider, fileManager, cmdFactory, logger)
}

func createXcodebuildArchiver(logger log.Logger, logFormatter string) (step.XcodebuildArchiver, error) {
	envRepository := env.NewRepository()
	pathProvider := pathutil.NewPathProvider()
	pathChecker := pathutil.NewPathChecker()
	pathModifier := pathutil.NewPathModifier()
	fileManager := fileutil.NewFileManager()
	cmdFactory := command.NewFactory(envRepository)

	xcodeCommandRunner := xcodecommand.Runner(nil)
	switch logFormatter {
	case step.XcodebuildTool:
		xcodeCommandRunner = xcodecommand.NewRawCommandRunner(logger, cmdFactory)
	case step.XcbeautifyTool:
		xcodeCommandRunner = xcodecommand.NewXcbeautifyRunner(logger, cmdFactory)
	case step.XcprettyTool:
		commandLocator := env.NewCommandLocator()
		rubyComamndFactory, err := ruby.NewCommandFactory(cmdFactory, commandLocator)
		if err != nil {
			return step.XcodebuildArchiver{}, fmt.Errorf("failed to install xcpretty: %s", err)
		}
		rubyEnv := ruby.NewEnvironment(rubyComamndFactory, commandLocator, logger)

		xcodeCommandRunner = xcodecommand.NewXcprettyCommandRunner(logger, cmdFactory, pathChecker, fileManager, rubyComamndFactory, rubyEnv)
	default:
		panic(fmt.Sprintf("Unknown log formatter: %s", logFormatter))
	}

	return step.NewXcodebuildArchiver(xcodeCommandRunner, pathProvider, pathChecker, pathModifier, fileManager, cmdFactory, logger), nil
}

func createRunOptions(config step.Config) step.RunOpts {
	return step.RunOpts{
		ProjectPath:       config.ProjectPath,
		Scheme:            config.Scheme,
		Configuration:     config.Configuration,
		LogFormatter:      config.LogFormatter,
		XcodeMajorVersion: config.XcodeMajorVersion,
		ArtifactName:      config.ArtifactName,

		CodesignManager: config.CodesignManager,

		PerformCleanAction:          config.PerformCleanAction,
		XcconfigContent:             config.XcconfigContent,
		XcodebuildAdditionalOptions: config.XcodebuildAdditionalOptions,
		CacheLevel:                  config.CacheLevel,

		CustomExportOptionsPlistContent: config.ExportOptionsPlistContent,
		ExportMethod:                    config.ExportMethod,
		TestFlightInternalTestingOnly:   config.TestFlightInternalTestingOnly,
		ICloudContainerEnvironment:      config.ICloudContainerEnvironment,
		ExportDevelopmentTeam:           config.ExportDevelopmentTeam,
		UploadBitcode:                   config.UploadBitcode,
		CompileBitcode:                  config.CompileBitcode,
	}
}

func createExportOptions(config step.Config, result step.RunResult) step.ExportOpts {
	return step.ExportOpts{
		OutputDir:      config.OutputDir,
		ArtifactName:   result.ArtifactName,
		ExportAllDsyms: config.ExportAllDsyms,

		Archive: result.Archive,

		ExportOptionsPath: result.ExportOptionsPath,
		IPAExportDir:      result.IPAExportDir,

		XcodebuildArchiveLog:       result.XcodebuildArchiveLog,
		XcodebuildExportArchiveLog: result.XcodebuildExportArchiveLog,
		IDEDistrubutionLogsDir:     result.IDEDistrubutionLogsDir,
	}
}
