package main

import (
	"os"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/fileutil"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/pathutil"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-steplib/steps-xcode-archive/step"
)

func main() {
	os.Exit(run())
}

func run() int {
	logger := log.NewLogger()
	archiver := createXcodebuildArchiver(logger)

	config, err := archiver.ProcessInputs()
	if err != nil {
		logger.Errorf("Process Inputs failed: %s", err)
		return 1
	}

	dependenciesOpts := step.EnsureDependenciesOpts{
		XCPretty: config.LogFormatter == "xcpretty",
	}
	if err := archiver.EnsureDependencies(dependenciesOpts); err != nil {
		logger.Warnf(err.Error())
		logger.Warnf("Switching to xcodebuild for output tool")
		config.LogFormatter = "xcodebuild"
	}

	runOpts := createRunOptions(config)
	out, runErr := archiver.Run(runOpts)

	exportOpts := createExportOptions(config, out)
	exportErr := archiver.ExportOutput(exportOpts)

	if runErr != nil {
		return 1
	}
	if exportErr != nil {
		return 1
	}

	return 0
}

func createXcodebuildArchiver(logger log.Logger) step.XcodebuildArchiver {
	xcodeVersionProvider := step.NewXcodebuildXcodeVersionProvider()
	envRepository := env.NewRepository()
	inputParser := stepconf.NewInputParser(envRepository)
	pathProvider := pathutil.NewPathProvider()
	pathChecker := pathutil.NewPathChecker()
	pathModifier := pathutil.NewPathModifier()
	fileManager := fileutil.NewFileManager()
	cmdFactory := command.NewFactory(envRepository)

	return step.NewXcodebuildArchiver(xcodeVersionProvider, inputParser, pathProvider, pathChecker, pathModifier, fileManager, logger, cmdFactory)
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
		ICloudContainerEnvironment:      config.ICloudContainerEnvironment,
		ExportDevelopmentTeam:           config.ExportDevelopmentTeam,
		UploadBitcode:                   config.UploadBitcode,
		CompileBitcode:                  config.CompileBitcode,
	}
}

func createExportOptions(config step.Config, out step.RunOut) step.ExportOpts {
	return step.ExportOpts{
		OutputDir:      config.OutputDir,
		ArtifactName:   out.ArtifactName,
		ExportAllDsyms: config.ExportAllDsyms,

		Archive: out.Archive,

		ExportOptionsPath: out.ExportOptionsPath,
		IPAExportDir:      out.IPAExportDir,

		XcodebuildArchiveLog:       out.XcodebuildArchiveLog,
		XcodebuildExportArchiveLog: out.XcodebuildExportArchiveLog,
		IDEDistrubutionLogsDir:     out.IDEDistrubutionLogsDir,
	}
}
