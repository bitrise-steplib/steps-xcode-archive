package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bitrise-io/go-steputils/input"
	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-io/go-utils/stringutil"
	"github.com/bitrise-io/go-xcode/exportoptions"
	"github.com/bitrise-io/go-xcode/models"
	"github.com/bitrise-io/go-xcode/profileutil"
	"github.com/bitrise-io/go-xcode/utility"
	"github.com/bitrise-io/go-xcode/xcarchive"
	"github.com/bitrise-io/go-xcode/xcodebuild"
	cache "github.com/bitrise-io/go-xcode/xcodecache"
	"github.com/bitrise-io/go-xcode/xcpretty"
	"github.com/bitrise-steplib/steps-xcode-archive/utils"
	"github.com/kballard/go-shellquote"
	"howett.net/plist"
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

// Config ...
type Config struct {
	Inputs

	AbsProjectPath             string
	TmpArchivePath             string
	IDEDistributionLogsZipPath string
	XcodebuildLogPath          string
	ExportOptionsPath          string
	IPAPath                    string
	ArchiveZipPath             string
	DSYMZipPath                string
	AppPath                    string
	IPAExportDir               string

	XcodeMajorVersion int
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

// XcodeVersionProvider ...
type XcodeVersionProvider interface {
	GetXcodeVersion() (models.XcodebuildVersionModel, error)
}

// XcodebuildXcodeVersionProvider ...
type XcodebuildXcodeVersionProvider struct {
}

// NewXcodebuildXcodeVersionProvider ...
func NewXcodebuildXcodeVersionProvider() XcodeVersionProvider {
	return XcodebuildXcodeVersionProvider{}
}

// GetXcodeVersion ...
func (p XcodebuildXcodeVersionProvider) GetXcodeVersion() (models.XcodebuildVersionModel, error) {
	return utility.GetXcodeVersion()
}

// StepInputParser ...
type StepInputParser interface {
	Parse(conf interface{}) error
}

// EnvStepInputParser ...
type EnvStepInputParser struct {
}

// NewEnvStepInputParser ...
func NewEnvStepInputParser() EnvStepInputParser {
	return EnvStepInputParser{}
}

// Parse ...
func (p EnvStepInputParser) Parse(conf interface{}) error {
	return stepconf.Parse(conf)
}

// XcodeArchiveStep ...
type XcodeArchiveStep struct {
	xcodeVersionProvider XcodeVersionProvider
	stepInputParser      StepInputParser
}

// NewXcodeArchiveStep ...
func NewXcodeArchiveStep() XcodeArchiveStep {
	return XcodeArchiveStep{
		xcodeVersionProvider: NewXcodebuildXcodeVersionProvider(),
		stepInputParser:      NewEnvStepInputParser(),
	}
}

// ProcessInputs ...
func (s XcodeArchiveStep) ProcessInputs() (Config, error) {
	var inputs Inputs
	if err := s.stepInputParser.Parse(&inputs); err != nil {
		return Config{}, fmt.Errorf("issue with input: %s", err)
	}

	stepconf.Print(inputs)
	fmt.Println()

	config := Config{Inputs: inputs}
	log.SetEnableDebugLog(config.VerboseLog)

	if config.ExportMethod == "auto-detect" {
		exportMethods := []exportoptions.Method{exportoptions.MethodAppStore, exportoptions.MethodAdHoc, exportoptions.MethodEnterprise, exportoptions.MethodDevelopment}
		log.Warnf("Export method: auto-detect is DEPRECATED, use a direct export method %s", exportMethods)
		fmt.Println()
	}

	if config.Workdir != "" {
		if err := input.ValidateIfDirExists(config.Workdir); err != nil {
			return Config{}, fmt.Errorf("issue with input Workdir: " + err.Error())
		}
	}

	if config.CustomExportOptionsPlistContent != "" {
		var options map[string]interface{}
		if _, err := plist.Unmarshal([]byte(config.CustomExportOptionsPlistContent), &options); err != nil {
			return Config{}, fmt.Errorf("issue with input CustomExportOptionsPlistContent: " + err.Error())
		}
	}

	if filepath.Ext(config.ProjectPath) != ".xcodeproj" && filepath.Ext(config.ProjectPath) != ".xcworkspace" {
		return Config{}, fmt.Errorf("issue with input ProjectPath: should be and .xcodeproj or .xcworkspace path")
	}

	log.Infof("step determined configs:")

	// Detect Xcode major version
	xcodebuildVersion, err := s.xcodeVersionProvider.GetXcodeVersion()
	if err != nil {
		return Config{}, fmt.Errorf("failed to determine xcode version, error: %s", err)
	}
	log.Printf("- xcodebuildVersion: %s (%s)", xcodebuildVersion.Version, xcodebuildVersion.BuildVersion)

	xcodeMajorVersion := xcodebuildVersion.MajorVersion
	if xcodeMajorVersion < minSupportedXcodeMajorVersion {
		return Config{}, fmt.Errorf("invalid xcode major version (%d), should not be less then min supported: %d", xcodeMajorVersion, minSupportedXcodeMajorVersion)
	}
	config.XcodeMajorVersion = int(xcodeMajorVersion)

	////

	// Validation CustomExportOptionsPlistContent
	customExportOptionsPlistContent := strings.TrimSpace(config.CustomExportOptionsPlistContent)
	if customExportOptionsPlistContent != config.CustomExportOptionsPlistContent {
		fmt.Println()
		log.Warnf("CustomExportOptionsPlistContent is stripped to remove spaces and new lines:")
		log.Printf(customExportOptionsPlistContent)
	}

	if customExportOptionsPlistContent != "" {
		if xcodeMajorVersion < 7 {
			fmt.Println()
			log.Warnf("CustomExportOptionsPlistContent is set, but CustomExportOptionsPlistContent only used if xcodeMajorVersion > 6")
			customExportOptionsPlistContent = ""
		} else {
			fmt.Println()
			log.Warnf("Ignoring the following options because CustomExportOptionsPlistContent provided:")
			log.Printf("- ExportMethod: %s", config.ExportMethod)
			log.Printf("- UploadBitcode: %s", config.UploadBitcode)
			log.Printf("- CompileBitcode: %s", config.CompileBitcode)
			log.Printf("- TeamID: %s", config.TeamID)
			log.Printf("- ICloudContainerEnvironment: %s", config.ICloudContainerEnvironment)
			fmt.Println()
		}
	}

	if config.ForceProvisioningProfileSpecifier != "" &&
		xcodeMajorVersion < 8 {
		fmt.Println()
		log.Warnf("ForceProvisioningProfileSpecifier is set, but ForceProvisioningProfileSpecifier only used if xcodeMajorVersion > 7")
		config.ForceProvisioningProfileSpecifier = ""
	}

	if config.ForceTeamID != "" &&
		xcodeMajorVersion < 8 {
		fmt.Println()
		log.Warnf("ForceTeamID is set, but ForceTeamID only used if xcodeMajorVersion > 7")
		config.ForceTeamID = ""
	}

	if config.ForceProvisioningProfileSpecifier != "" &&
		config.ForceProvisioningProfile != "" {
		fmt.Println()
		log.Warnf("both ForceProvisioningProfileSpecifier and ForceProvisioningProfile are set, using ForceProvisioningProfileSpecifier")
		config.ForceProvisioningProfile = ""
	}

	fmt.Println()

	absProjectPath, err := filepath.Abs(config.ProjectPath)
	if err != nil {
		return Config{}, fmt.Errorf("failed to get absolute project path, error: %s", err)
	}
	config.AbsProjectPath = absProjectPath

	// abs out dir pth
	absOutputDir, err := pathutil.AbsPath(config.OutputDir)
	if err != nil {
		return Config{}, fmt.Errorf("failed to expand OutputDir (%s), error: %s", config.OutputDir, err)
	}
	config.OutputDir = absOutputDir

	if exist, err := pathutil.IsPathExists(config.OutputDir); err != nil {
		return Config{}, fmt.Errorf("failed to check if OutputDir exist, error: %s", err)
	} else if !exist {
		if err := os.MkdirAll(config.OutputDir, 0777); err != nil {
			return Config{}, fmt.Errorf("failed to create OutputDir (%s), error: %s", config.OutputDir, err)
		}
	}

	// output files
	tmpArchiveDir, err := pathutil.NormalizedOSTempDirPath("__archive__")
	if err != nil {
		return Config{}, fmt.Errorf("failed to create temp dir for archives, error: %s", err)
	}
	config.TmpArchivePath = filepath.Join(tmpArchiveDir, config.ArtifactName+".xcarchive")

	appPath := filepath.Join(config.OutputDir, config.ArtifactName+".app")
	config.AppPath = appPath

	ipaPath := filepath.Join(config.OutputDir, config.ArtifactName+".ipa")
	config.IPAPath = ipaPath

	exportOptionsPath := filepath.Join(config.OutputDir, "export_options.plist")
	config.ExportOptionsPath = exportOptionsPath

	rawXcodebuildOutputLogPath := filepath.Join(config.OutputDir, "raw-xcodebuild-output.log")
	config.XcodebuildLogPath = rawXcodebuildOutputLogPath

	dsymZipPath := filepath.Join(config.OutputDir, config.ArtifactName+".dSYM.zip")
	config.DSYMZipPath = dsymZipPath

	archiveZipPath := filepath.Join(config.OutputDir, config.ArtifactName+".xcarchive.zip")
	config.ArchiveZipPath = archiveZipPath

	ideDistributionLogsZipPath := filepath.Join(config.OutputDir, "xcodebuild.xcdistributionlogs.zip")
	config.IDEDistributionLogsZipPath = ideDistributionLogsZipPath

	tmpIPAExportDir, err := pathutil.NormalizedOSTempDirPath("__export__")
	if err != nil {
		return Config{}, fmt.Errorf("failed to create tmp dir, error: %s", err)
	}
	config.IPAExportDir = tmpIPAExportDir

	// cleanup
	filesToCleanup := []string{
		appPath,
		ipaPath,
		exportOptionsPath,
		rawXcodebuildOutputLogPath,

		dsymZipPath,
		archiveZipPath,
		ideDistributionLogsZipPath,
	}

	for _, pth := range filesToCleanup {
		if exist, err := pathutil.IsPathExists(pth); err != nil {
			return Config{}, fmt.Errorf("failed to check if path (%s) exist, error: %s", pth, err)
		} else if exist {
			if err := os.RemoveAll(pth); err != nil {
				return Config{}, fmt.Errorf("failed to remove path (%s), error: %s", pth, err)
			}
		}
	}

	return config, nil
}

// InstallDepsOpts ...
type InstallDepsOpts struct {
	InstallXcpretty bool
}

// InstallDeps ...
func (s XcodeArchiveStep) InstallDeps(opts InstallDepsOpts) error {
	if opts.InstallXcpretty {
		fmt.Println()
		log.Infof("Checking if output tool (xcpretty) is installed")

		installed, err := xcpretty.IsInstalled()
		if err != nil {
			return fmt.Errorf("failed to check if xcpretty is installed, error: %s", err)
		} else if !installed {
			log.Warnf(`xcpretty is not installed`)
			fmt.Println()
			log.Printf("Installing xcpretty")

			cmds, err := xcpretty.Install()
			if err != nil {
				return fmt.Errorf("failed to create xcpretty install command: %s", err)
			}

			for _, cmd := range cmds {
				if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
					if errorutil.IsExitStatusError(err) {
						return fmt.Errorf("%s failed: %s", cmd.PrintableCommandArgs(), out)
					}
					return fmt.Errorf("%s failed: %s", cmd.PrintableCommandArgs(), err)
				}
			}

		}
	}

	xcprettyVersion, err := xcpretty.Version()
	if err != nil {
		return fmt.Errorf("failed to determine xcpretty version, error: %s", err)
	}
	log.Printf("- xcprettyVersion: %s", xcprettyVersion.String())

	return nil
}

// XcodeArchiveOpts ...
type XcodeArchiveOpts struct {
	// Shared
	ProjectPath       string
	Scheme            string
	Configuration     string
	OutputTool        string
	XcodebuildLogPath string
	XcodeMajorVersion int

	// Archive
	ArchivePath string

	ForceTeamID                       string
	ForceProvisioningProfileSpecifier string
	ForceProvisioningProfile          string
	ForceCodeSignIdentity             string
	IsCleanBuild                      bool
	DisableIndexWhileBuilding         bool
	XcodebuildOptions                 string

	CacheLevel string
}

func (s XcodeArchiveStep) xcodeArchive(opts XcodeArchiveOpts) error {
	//
	// Open Xcode project
	xcodeProj, scheme, configuration, err := utils.OpenArchivableProject(opts.ProjectPath, opts.Scheme, opts.Configuration)
	if err != nil {
		return fmt.Errorf("failed to open project: %s: %s", opts.ProjectPath, err)
	}

	platform, err := utils.BuildableTargetPlatform(xcodeProj, scheme, configuration, utils.XcodeBuild{})
	if err != nil {
		return fmt.Errorf("failed to read project platform: %s: %s", opts.ProjectPath, err)
	}

	mainTarget, err := archivableApplicationTarget(xcodeProj, scheme, configuration)
	if err != nil {
		return fmt.Errorf("failed to read main application target: %s", err)
	}
	if mainTarget.ProductType == appClipProductType {
		log.Errorf("Selected scheme: '%s' targets an App Clip target (%s),", opts.Scheme, mainTarget.Name)
		log.Errorf("'Xcode Archive & Export for iOS' step is intended to archive the project using a scheme targeting an Application target.")
		log.Errorf("Please select a scheme targeting an Application target to archive and export the main Application")
		log.Errorf("and use 'Export iOS and tvOS Xcode archive' step to export an App Clip.")
		os.Exit(1)
	}

	//
	// Create the Archive with Xcode Command Line tools
	log.Infof("Create the Archive ...")
	fmt.Println()

	isWorkspace := false
	ext := filepath.Ext(opts.ProjectPath)
	if ext == ".xcodeproj" {
		isWorkspace = false
	} else if ext == ".xcworkspace" {
		isWorkspace = true
	} else {
		return fmt.Errorf("project file extension should be .xcodeproj or .xcworkspace, but got: %s", ext)
	}

	archiveCmd := xcodebuild.NewCommandBuilder(opts.ProjectPath, isWorkspace, xcodebuild.ArchiveAction)
	archiveCmd.SetScheme(opts.Scheme)
	archiveCmd.SetConfiguration(opts.Configuration)

	if opts.ForceTeamID != "" {
		log.Printf("Forcing Development Team: %s", opts.ForceTeamID)
		archiveCmd.SetForceDevelopmentTeam(opts.ForceTeamID)
	}
	if opts.ForceProvisioningProfileSpecifier != "" {
		log.Printf("Forcing Provisioning Profile Specifier: %s", opts.ForceProvisioningProfileSpecifier)
		archiveCmd.SetForceProvisioningProfileSpecifier(opts.ForceProvisioningProfileSpecifier)
	}
	if opts.ForceProvisioningProfile != "" {
		log.Printf("Forcing Provisioning Profile: %s", opts.ForceProvisioningProfile)
		archiveCmd.SetForceProvisioningProfile(opts.ForceProvisioningProfile)
	}
	if opts.ForceCodeSignIdentity != "" {
		log.Printf("Forcing Code Signing Identity: %s", opts.ForceCodeSignIdentity)
		archiveCmd.SetForceCodeSignIdentity(opts.ForceCodeSignIdentity)
	}

	if opts.IsCleanBuild {
		archiveCmd.SetCustomBuildAction("clean")
	}

	archiveCmd.SetDisableIndexWhileBuilding(opts.DisableIndexWhileBuilding)
	archiveCmd.SetArchivePath(opts.ArchivePath)

	destination := "generic/platform=" + string(platform)
	options := []string{"-destination", destination}
	if opts.XcodebuildOptions != "" {
		userOptions, err := shellquote.Split(opts.XcodebuildOptions)
		if err != nil {
			return fmt.Errorf("failed to shell split XcodebuildOptions (%s), error: %s", opts.XcodebuildOptions, err)
		}

		if sliceutil.IsStringInSlice("-destination", userOptions) {
			options = userOptions
		} else {
			options = append(options, userOptions...)
		}
	}
	archiveCmd.SetCustomOptions(options)

	var swiftPackagesPath string
	if opts.XcodeMajorVersion >= 11 {
		var err error
		if swiftPackagesPath, err = cache.SwiftPackagesPath(opts.ProjectPath); err != nil {
			return fmt.Errorf("failed to get Swift Packages path, error: %s", err)
		}
	}

	rawXcodebuildOut, err := runArchiveCommandWithRetry(archiveCmd, opts.OutputTool == "xcpretty", swiftPackagesPath)
	if err != nil || opts.OutputTool == "xcodebuild" {
		const lastLinesMsg = "\nLast lines of the Xcode's build log:"
		if err != nil {
			log.Infof(colorstring.Red(lastLinesMsg))
		} else {
			log.Infof(lastLinesMsg)
		}
		fmt.Println(stringutil.LastNLines(rawXcodebuildOut, 20))

		if err := utils.ExportOutputFileContent(rawXcodebuildOut, opts.XcodebuildLogPath, bitriseXcodeRawResultTextEnvKey); err != nil {
			log.Warnf("Failed to export %s, error: %s", bitriseXcodeRawResultTextEnvKey, err)
		} else {
			log.Infof(colorstring.Magenta(fmt.Sprintf(`You can find the last couple of lines of Xcode's build log above, but the full log is also available in the raw-xcodebuild-output.log
The log file is stored in $BITRISE_DEPLOY_DIR, and its full path is available in the $BITRISE_XCODE_RAW_RESULT_TEXT_PATH environment variable
(value: %s)`, opts.XcodebuildLogPath)))
		}
	}
	if err != nil {
		return fmt.Errorf("archive failed, error: %s", err)
	}

	fmt.Println()

	// Ensure xcarchive exists
	if exist, err := pathutil.IsPathExists(opts.ArchivePath); err != nil {
		return fmt.Errorf("failed to check if archive exist, error: %s", err)
	} else if !exist {
		return fmt.Errorf("no archive generated at: %s", opts.ArchivePath)
	}

	// Cache swift PM
	if opts.XcodeMajorVersion >= 11 && opts.CacheLevel == "swift_packages" {
		if err := cache.CollectSwiftPackages(opts.ProjectPath); err != nil {
			log.Warnf("Failed to mark swift packages for caching, error: %s", err)
		}
	}

	envsToUnset := []string{"GEM_HOME", "GEM_PATH", "RUBYLIB", "RUBYOPT", "BUNDLE_BIN_PATH", "_ORIGINAL_GEM_PATH", "BUNDLE_GEMFILE"}
	for _, key := range envsToUnset {
		if err := os.Unsetenv(key); err != nil {
			return fmt.Errorf("failed to unset (%s), error: %s", key, err)
		}
	}

	archive, err := xcarchive.NewIosArchive(opts.ArchivePath)
	if err != nil {
		return fmt.Errorf("failed to parse archive, error: %s", err)
	}

	mainApplication := archive.Application
	archiveExportMethod := mainApplication.ProvisioningProfile.ExportType
	archiveCodeSignIsXcodeManaged := profileutil.IsXcodeManaged(mainApplication.ProvisioningProfile.Name)

	log.Infof("Archive infos:")
	log.Printf("team: %s (%s)", mainApplication.ProvisioningProfile.TeamName, mainApplication.ProvisioningProfile.TeamID)
	log.Printf("profile: %s (%s)", mainApplication.ProvisioningProfile.Name, mainApplication.ProvisioningProfile.UUID)
	log.Printf("export: %s", archiveExportMethod)
	log.Printf("xcode managed profile: %v", archiveCodeSignIsXcodeManaged)
	fmt.Println()

	return nil
}

// XcodeIPAExportOpts ...
type XcodeIPAExportOpts struct {
	// Shared
	ProjectPath       string
	Scheme            string
	Configuration     string
	OutputTool        string
	XcodebuildLogPath string
	XcodeMajorVersion int

	// IPA Export
	ExportOptionsPath          string
	IPAPath                    string
	ArchivePath                string
	IDEDistributionLogsZipPath string
	IPAExportDir               string

	CustomExportOptionsPlistContent string

	ExportMethod        string
	ArchiveExportMethod string

	ICloudContainerEnvironment string
	TeamID                     string
	UploadBitcode              bool
	CompileBitcode             bool
}

func (s XcodeArchiveStep) xcodeIPAExport(opts XcodeIPAExportOpts) error {
	//
	// Exporting the ipa with Xcode Command Line tools

	/*
		You'll get a "Error Domain=IDEDistributionErrorDomain Code=14 "No applicable devices found."" error
		if $GEM_HOME is set and the project's directory includes a Gemfile - to fix this
		we'll unset GEM_HOME as that's not required for xcodebuild anyway.
		This probably fixes the RVM issue too, but that still should be tested.
		See also:
		- http://stackoverflow.com/questions/33041109/xcodebuild-no-applicable-devices-found-when-exporting-archive
		- https://gist.github.com/claybridges/cea5d4afd24eda268164
	*/
	log.Infof("Exporting ipa from the archive...")
	fmt.Println()

	log.Printf("Exporting ipa with ExportOptions.plist")

	if opts.CustomExportOptionsPlistContent != "" {
		log.Printf("Custom export options content provided, using it:")
		fmt.Println(opts.CustomExportOptionsPlistContent)

		if err := fileutil.WriteStringToFile(opts.ExportOptionsPath, opts.CustomExportOptionsPlistContent); err != nil {
			return fmt.Errorf("failed to write export options to file, error: %s", err)
		}
	} else {
		log.Printf("No custom export options content provided, generating export options...")

		exportMethod, err := determineExportMethod(opts.ExportMethod, exportoptions.Method(opts.ArchiveExportMethod))
		if err != nil {
			return err
		}

		xcodeProj, scheme, configuration, err := utils.OpenArchivableProject(opts.ProjectPath, opts.Scheme, opts.Configuration)
		if err != nil {
			return fmt.Errorf("failed to open project: %s: %s", opts.ProjectPath, err)
		}

		archive, err := xcarchive.NewIosArchive(opts.ArchivePath)
		if err != nil {
			return fmt.Errorf("failed to parse archive, error: %s", err)
		}

		mainApplication := archive.Application
		archiveCodeSignIsXcodeManaged := profileutil.IsXcodeManaged(mainApplication.ProvisioningProfile.Name)

		generator := NewExportOptionsGenerator(xcodeProj, scheme, configuration)
		exportOptions, err := generator.GenerateApplicationExportOptions(exportMethod, opts.ICloudContainerEnvironment, opts.TeamID,
			opts.UploadBitcode, opts.CompileBitcode, archiveCodeSignIsXcodeManaged, int64(opts.XcodeMajorVersion))
		if err != nil {
			return err
		}

		fmt.Println()
		log.Printf("generated export options content:")
		fmt.Println()
		fmt.Println(exportOptions.String())

		if err := exportOptions.WriteToFile(opts.ExportOptionsPath); err != nil {
			return err
		}
	}

	fmt.Println()

	exportCmd := xcodebuild.NewExportCommand()
	exportCmd.SetArchivePath(opts.ArchivePath)
	exportCmd.SetExportDir(opts.IPAExportDir)
	exportCmd.SetExportOptionsPlist(opts.ExportOptionsPath)

	if opts.OutputTool == "xcpretty" {
		xcprettyCmd := xcpretty.New(exportCmd)

		logWithTimestamp(colorstring.Green, xcprettyCmd.PrintableCmd())
		fmt.Println()

		if xcodebuildOut, err := xcprettyCmd.Run(); err != nil {
			// xcodebuild raw output
			if err := utils.ExportOutputFileContent(xcodebuildOut, opts.XcodebuildLogPath, bitriseXcodeRawResultTextEnvKey); err != nil {
				log.Warnf("Failed to export %s, error: %s", bitriseXcodeRawResultTextEnvKey, err)
			} else {
				log.Warnf(`If you can't find the reason of the error in the log, please check the raw-xcodebuild-output.log
The log file is stored in $BITRISE_DEPLOY_DIR, and its full path
is available in the $BITRISE_XCODE_RAW_RESULT_TEXT_PATH environment variable`)
			}

			// xcdistributionlogs
			if logsDirPth, err := findIDEDistrubutionLogsPath(xcodebuildOut); err != nil {
				log.Warnf("Failed to find xcdistributionlogs, error: %s", err)
			} else if err := utils.ExportOutputDirAsZip(logsDirPth, opts.IDEDistributionLogsZipPath, bitriseIDEDistributionLogsPthEnvKey); err != nil {
				log.Warnf("Failed to export %s, error: %s", bitriseIDEDistributionLogsPthEnvKey, err)
			} else {
				criticalDistLogFilePth := filepath.Join(logsDirPth, "IDEDistribution.critical.log")
				log.Warnf("IDEDistribution.critical.log:")
				if criticalDistLog, err := fileutil.ReadStringFromFile(criticalDistLogFilePth); err == nil {
					log.Printf(criticalDistLog)
				}

				log.Warnf(`Also please check the xcdistributionlogs
The logs directory is stored in $BITRISE_DEPLOY_DIR, and its full path
is available in the $BITRISE_IDEDISTRIBUTION_LOGS_PATH environment variable`)
			}

			return fmt.Errorf("export failed, error: %s", err)
		}
	} else {
		logWithTimestamp(colorstring.Green, exportCmd.PrintableCmd())
		fmt.Println()

		if xcodebuildOut, err := exportCmd.RunAndReturnOutput(); err != nil {
			// xcdistributionlogs
			if logsDirPth, err := findIDEDistrubutionLogsPath(xcodebuildOut); err != nil {
				log.Warnf("Failed to find xcdistributionlogs, error: %s", err)
			} else if err := utils.ExportOutputDirAsZip(logsDirPth, opts.IDEDistributionLogsZipPath, bitriseIDEDistributionLogsPthEnvKey); err != nil {
				log.Warnf("Failed to export %s, error: %s", bitriseIDEDistributionLogsPthEnvKey, err)
			} else {
				criticalDistLogFilePth := filepath.Join(logsDirPth, "IDEDistribution.critical.log")
				log.Warnf("IDEDistribution.critical.log:")
				if criticalDistLog, err := fileutil.ReadStringFromFile(criticalDistLogFilePth); err == nil {
					log.Printf(criticalDistLog)
				}

				log.Warnf(`If you can't find the reason of the error in the log, please check the xcdistributionlogs
The logs directory is stored in $BITRISE_DEPLOY_DIR, and its full path
is available in the $BITRISE_IDEDISTRIBUTION_LOGS_PATH environment variable`)
			}

			return fmt.Errorf("export failed, error: %s", err)
		}
	}

	return nil
}

// RunOpts ...
type RunOpts struct {
	// Shared
	ProjectPath                string
	Scheme                     string
	Configuration              string
	OutputTool                 string
	XcodebuildLogPath          string
	XcodeMajorVersion          int
	IDEDistributionLogsZipPath string
	OutputDir                  string

	// Archive
	ArchivePath string

	ForceTeamID                       string
	ForceProvisioningProfileSpecifier string
	ForceProvisioningProfile          string
	ForceCodeSignIdentity             string
	IsCleanBuild                      bool
	DisableIndexWhileBuilding         bool
	XcodebuildOptions                 string

	CacheLevel string

	// IPA Export
	ExportOptionsPath string
	IPAPath           string
	IPAExportDir      string

	CustomExportOptionsPlistContent string

	ExportMethod               string
	ICloudContainerEnvironment string
	TeamID                     string
	UploadBitcode              bool
	CompileBitcode             bool
}

// Run ...
func (s XcodeArchiveStep) Run(opts RunOpts) error {
	archiveOpts := XcodeArchiveOpts{
		ProjectPath:       opts.ProjectPath,
		Scheme:            opts.Scheme,
		Configuration:     opts.Configuration,
		OutputTool:        opts.OutputTool,
		XcodebuildLogPath: opts.XcodebuildLogPath,
		XcodeMajorVersion: opts.XcodeMajorVersion,

		ArchivePath:                       opts.ArchivePath,
		ForceTeamID:                       opts.ForceTeamID,
		ForceProvisioningProfileSpecifier: opts.ForceProvisioningProfileSpecifier,
		ForceProvisioningProfile:          opts.ForceProvisioningProfile,
		ForceCodeSignIdentity:             opts.ForceCodeSignIdentity,
		IsCleanBuild:                      opts.IsCleanBuild,
		DisableIndexWhileBuilding:         opts.DisableIndexWhileBuilding,
		XcodebuildOptions:                 opts.XcodebuildOptions,
		CacheLevel:                        opts.CacheLevel,
	}
	err := s.xcodeArchive(archiveOpts)
	if err != nil {
		return err
	}

	IPAExportOpts := XcodeIPAExportOpts{
		ProjectPath:       opts.ProjectPath,
		Scheme:            opts.Scheme,
		Configuration:     opts.Configuration,
		OutputTool:        opts.OutputTool,
		XcodebuildLogPath: opts.XcodebuildLogPath,
		XcodeMajorVersion: opts.XcodeMajorVersion,

		ArchivePath: opts.ArchivePath,

		ExportOptionsPath: opts.ExportOptionsPath,
		IPAPath:           opts.IPAPath,
		IPAExportDir:      opts.IPAExportDir,

		CustomExportOptionsPlistContent: opts.CustomExportOptionsPlistContent,

		ExportMethod:               opts.ExportMethod,
		ICloudContainerEnvironment: opts.ICloudContainerEnvironment,
		TeamID:                     opts.TeamID,
		UploadBitcode:              opts.UploadBitcode,
		CompileBitcode:             opts.CompileBitcode,
	}
	return s.xcodeIPAExport(IPAExportOpts)
}

// ExportOpts ...
type ExportOpts struct {
	OutputDir string

	IPAExportDir string

	ArchivePath    string
	ArchiveZipPath string
	AppPath        string
	DSYMZipPath    string
	IPAPath        string

	ExportAllDsyms bool
}

// ExportOutput ...
func (s XcodeArchiveStep) ExportOutput(opts ExportOpts) error {
	// Search for ipa
	fileList := []string{}
	ipaFiles := []string{}
	if walkErr := filepath.Walk(opts.IPAExportDir, func(pth string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		fileList = append(fileList, pth)

		if filepath.Ext(pth) == ".ipa" {
			ipaFiles = append(ipaFiles, pth)
		}

		return nil
	}); walkErr != nil {
		return fmt.Errorf("failed to search for .ipa file, error: %s", walkErr)
	}

	if len(ipaFiles) == 0 {
		log.Errorf("No .ipa file found at export dir: %s", opts.IPAExportDir)
		log.Printf("File list in the export dir:")
		for _, pth := range fileList {
			log.Printf("- %s", pth)
		}
		return fmt.Errorf("")
	}

	if err := command.CopyFile(ipaFiles[0], opts.IPAPath); err != nil {
		return fmt.Errorf("failed to copy (%s) -> (%s), error: %s", ipaFiles[0], opts.IPAPath, err)
	}

	if len(ipaFiles) > 1 {
		log.Warnf("More than 1 .ipa file found, exporting first one: %s", ipaFiles[0])
		log.Warnf("Moving every ipa to the BITRISE_DEPLOY_DIR")

		for i, pth := range ipaFiles {
			if i == 0 {
				continue
			}

			base := filepath.Base(pth)
			deployPth := filepath.Join(opts.OutputDir, base)

			if err := command.CopyFile(pth, deployPth); err != nil {
				return fmt.Errorf("failed to copy (%s) -> (%s), error: %s", pth, opts.IPAPath, err)
			}
		}
	}

	log.Infof("Exporting outputs...")

	//
	// Export outputs

	// Export .xcarchive
	fmt.Println()

	if err := utils.ExportOutputDir(opts.ArchivePath, opts.ArchivePath, bitriseXCArchivePthEnvKey); err != nil {
		return fmt.Errorf("failed to export %s, error: %s", bitriseXCArchivePthEnvKey, err)
	}

	log.Donef("The xcarchive path is now available in the Environment Variable: %s (value: %s)", bitriseXCArchivePthEnvKey, opts.ArchivePath)

	if err := utils.ExportOutputDirAsZip(opts.ArchivePath, opts.ArchiveZipPath, bitriseXCArchiveZipPthEnvKey); err != nil {
		return fmt.Errorf("failed to export %s, error: %s", bitriseXCArchiveZipPthEnvKey, err)
	}

	log.Donef("The xcarchive zip path is now available in the Environment Variable: %s (value: %s)", bitriseXCArchiveZipPthEnvKey, opts.ArchiveZipPath)

	// Export .app
	fmt.Println()

	archive, err := xcarchive.NewIosArchive(opts.ArchivePath)
	if err != nil {
		return fmt.Errorf("failed to parse archive, error: %s", err)
	}

	mainApplication := archive.Application
	exportedApp := mainApplication.Path

	if err := utils.ExportOutputDir(exportedApp, exportedApp, bitriseAppDirPthEnvKey); err != nil {
		return fmt.Errorf("failed to export %s, error: %s", bitriseAppDirPthEnvKey, err)
	}

	log.Donef("The app directory is now available in the Environment Variable: %s (value: %s)", bitriseAppDirPthEnvKey, opts.AppPath)

	// Export .ipa
	fmt.Println()

	if err := utils.ExportOutputFile(opts.IPAPath, opts.IPAPath, bitriseIPAPthEnvKey); err != nil {
		return fmt.Errorf("failed to export %s, error: %s", bitriseIPAPthEnvKey, err)
	}

	log.Donef("The ipa path is now available in the Environment Variable: %s (value: %s)", bitriseIPAPthEnvKey, opts.IPAPath)

	// Export .dSYMs
	fmt.Println()

	appDSYM, frameworkDSYMs, err := archive.FindDSYMs()
	if err != nil {
		return fmt.Errorf("failed to export dsyms, error: %s", err)
	}

	if err == nil {
		dsymDir, err := pathutil.NormalizedOSTempDirPath("__dsyms__")
		if err != nil {
			return fmt.Errorf("failed to create tmp dir, error: %s", err)
		}

		if len(appDSYM) > 0 {
			if err := exportDSYMs(dsymDir, appDSYM); err != nil {
				return fmt.Errorf("failed to export dSYMs: %v", err)
			}
		} else {
			log.Warnf("no app dsyms found")
		}

		if opts.ExportAllDsyms {
			if err := exportDSYMs(dsymDir, frameworkDSYMs); err != nil {
				return fmt.Errorf("failed to export dSYMs: %v", err)
			}
		}

		if err := utils.ExportOutputDir(dsymDir, dsymDir, bitriseDSYMDirPthEnvKey); err != nil {
			return fmt.Errorf("failed to export %s, error: %s", bitriseDSYMDirPthEnvKey, err)
		}

		log.Donef("The dSYM dir path is now available in the Environment Variable: %s (value: %s)", bitriseDSYMDirPthEnvKey, dsymDir)

		if err := utils.ExportOutputDirAsZip(dsymDir, opts.DSYMZipPath, bitriseDSYMPthEnvKey); err != nil {
			return fmt.Errorf("failed to export %s, error: %s", bitriseDSYMPthEnvKey, err)
		}

		log.Donef("The dSYM zip path is now available in the Environment Variable: %s (value: %s)", bitriseDSYMPthEnvKey, opts.DSYMZipPath)
	}

	return nil
}

// RunStep ...
func RunStep() error {
	step := NewXcodeArchiveStep()

	config, err := step.ProcessInputs()
	if err != nil {
		return err
	}

	installDepsOpts := InstallDepsOpts{
		InstallXcpretty: config.OutputTool == "xcpretty",
	}
	if err := step.InstallDeps(installDepsOpts); err != nil {
		log.Warnf(err.Error())
		log.Warnf("Switching to xcodebuild for output tool")
		config.OutputTool = "xcodebuild"
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
	if err := step.Run(runOpts); err != nil {
		return err
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
		return err
	}

	return nil
}

func main() {
	if err := RunStep(); err != nil {
		log.Errorf(err.Error())
		os.Exit(1)
	}
}
