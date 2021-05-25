package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-xcode/exportoptions"
)

const (
	minSupportedXcodeMajorVersion = 9
)

const (
	bitriseXcodeRawResultTextEnvKey     = "BITRISE_XCODE_RAW_RESULT_TEXT_PATH"
	bitriseIDEDistributionLogsPthEnvKey = "BITRISE_IDEDISTRIBUTION_LOGS_PATH"
	bitriseXCArchivePthEnvKey           = "BITRISE_XCARCHIVE_PATH"
	bitriseXCArchiveZipPthEnvKey        = "BITRISE_XCARCHIVE_ZIP_PATH"
	bitriseAppDirPthEnvKey              = "BITRISE_APP_DIR_PATH"
	bitriseIPAPthEnvKey                 = "BITRISE_IPA_PATH"
	bitriseDSYMDirPthEnvKey             = "BITRISE_DSYM_DIR_PATH"
	bitriseDSYMPthEnvKey                = "BITRISE_DSYM_PATH"
)

// Inputs ...
type Inputs struct {
	ExportMethod               string `env:"export_method,opt[auto-detect,app-store,ad-hoc,enterprise,development]"`
	UploadBitcode              bool   `env:"upload_bitcode,opt[yes,no]"`
	CompileBitcode             bool   `env:"compile_bitcode,opt[yes,no]"`
	ICloudContainerEnvironment string `env:"icloud_container_environment"`
	TeamID                     string `env:"team_id"`

	ForceTeamID                       string `env:"force_team_id"`
	ForceProvisioningProfileSpecifier string `env:"force_provisioning_profile_specifier"`
	ForceProvisioningProfile          string `env:"force_provisioning_profile"`
	ForceCodeSignIdentity             string `env:"force_code_sign_identity"`
	CustomExportOptionsPlistContent   string `env:"custom_export_options_plist_content"`

	OutputTool                string `env:"output_tool,opt[xcpretty,xcodebuild]"`
	Workdir                   string `env:"workdir"`
	ProjectPath               string `env:"project_path,file"`
	Scheme                    string `env:"scheme,required"`
	Configuration             string `env:"configuration"`
	OutputDir                 string `env:"output_dir,required"`
	IsCleanBuild              bool   `env:"is_clean_build,opt[yes,no]"`
	XcodebuildOptions         string `env:"xcodebuild_options"`
	DisableIndexWhileBuilding bool   `env:"disable_index_while_building,opt[yes,no]"`

	ExportAllDsyms bool   `env:"export_all_dsyms,opt[yes,no]"`
	ArtifactName   string `env:"artifact_name"`
	VerboseLog     bool   `env:"verbose_log,opt[yes,no]"`

	CacheLevel string `env:"cache_level,opt[none,swift_packages]"`
}

func fail(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func findIDEDistrubutionLogsPath(output string) (string, error) {
	pattern := `IDEDistribution: -\[IDEDistributionLogging _createLoggingBundleAtPath:\]: Created bundle at path '(?P<log_path>.*)'`
	re := regexp.MustCompile(pattern)

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if match := re.FindStringSubmatch(line); len(match) == 2 {
			return match[1], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", nil
}

func currentTimestamp() string {
	timeStampFormat := "15:04:05"
	currentTime := time.Now()
	return currentTime.Format(timeStampFormat)
}

// ColoringFunc ...
type ColoringFunc func(...interface{}) string

func logWithTimestamp(coloringFunc ColoringFunc, format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	messageWithTimeStamp := fmt.Sprintf("[%s] %s", currentTimestamp(), coloringFunc(message))
	fmt.Println(messageWithTimeStamp)
}

func determineExportMethod(desiredExportMethod string, archiveExportMethod exportoptions.Method) (exportoptions.Method, error) {
	if desiredExportMethod == "auto-detect" {
		log.Printf("auto-detect export method specified: using the archive profile's export method: %s", archiveExportMethod)
		return archiveExportMethod, nil
	}

	exportMethod, err := exportoptions.ParseMethod(desiredExportMethod)
	if err != nil {
		return "", fmt.Errorf("failed to parse export method: %s", err)
	}
	log.Printf("export method specified: %s", desiredExportMethod)

	return exportMethod, nil
}

func exportDSYMs(dsymDir string, dsyms []string) error {
	for _, dsym := range dsyms {
		if err := command.CopyDir(dsym, dsymDir, false); err != nil {
			return fmt.Errorf("could not copy (%s) to directory (%s): %s", dsym, dsymDir, err)
		}
	}
	return nil
}

// RunStep ...
func RunStep() int {
	step := XcodeArchiveStep{}

	config, err := step.ProcessInputs()
	if err != nil {
		log.Errorf(err.Error())
		return 1
	}

	installDepsOpts := InstallDepsOpts{
		InstallXcpretty: config.OutputTool == "xcpretty",
	}
	if err := step.InstallDeps(installDepsOpts); err != nil {
		log.Errorf(err.Error())
		return 1
	}

	runOpts := RunOpts{
		ProjectPath:                config.AbsProjectPath,
		Scheme:                     config.Scheme,
		Configuration:              config.Configuration,
		OutputTool:                 config.OutputTool,
		XcodebuildLogPath:          config.XcodebuildLogPath,
		XcodeMajorVersion:          config.XcodeMajorVersion,
		IDEDistributionLogsZipPath: config.IDEDistributionLogsZipPath,
		OutputDir:                  config.OutputDir,

		ArchivePath:                       config.TmpArchivePath,
		ForceTeamID:                       config.ForceTeamID,
		ForceProvisioningProfileSpecifier: config.ForceProvisioningProfileSpecifier,
		ForceProvisioningProfile:          config.ForceProvisioningProfile,
		ForceCodeSignIdentity:             config.ForceCodeSignIdentity,
		IsCleanBuild:                      config.IsCleanBuild,
		DisableIndexWhileBuilding:         config.DisableIndexWhileBuilding,
		XcodebuildOptions:                 config.XcodebuildOptions,
		CacheLevel:                        config.CacheLevel,

		ExportOptionsPath: config.ExportOptionsPath,
		IPAPath:           config.IPAPath,
		IPAExportDir:      config.IPAExportDir,

		CustomExportOptionsPlistContent: config.CustomExportOptionsPlistContent,

		ExportMethod:               config.ExportMethod,
		ICloudContainerEnvironment: config.ICloudContainerEnvironment,
		TeamID:                     config.TeamID,
		UploadBitcode:              config.UploadBitcode,
		CompileBitcode:             config.CompileBitcode,
	}
	if code, err := step.Run(runOpts); err != nil {
		log.Errorf(err.Error())
		return code
	}

	exportOpts := ExportOpts{
		OutputDir: config.OutputDir,

		IPAExportDir: config.IPAExportDir,

		ArchivePath:    config.TmpArchivePath,
		ArchiveZipPath: config.ArchiveZipPath,
		AppPath:        config.AppPath,
		DSYMZipPath:    config.DSYMZipPath,
		IPAPath:        config.IPAPath,

		ExportAllDsyms: config.ExportAllDsyms,
	}
	if err := step.ExportOutput(exportOpts); err != nil {
		log.Errorf(err.Error())
		return 1
	}

	return 0
}

func main() {
	os.Exit(RunStep())
}
