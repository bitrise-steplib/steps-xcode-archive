package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/colorstring"
	v1command "github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/errorutil"
	v1fileutil "github.com/bitrise-io/go-utils/fileutil"
	v1log "github.com/bitrise-io/go-utils/log"
	v1pathutil "github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-io/go-utils/stringutil"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-io/go-xcode/devportalservice"
	"github.com/bitrise-io/go-xcode/exportoptions"
	"github.com/bitrise-io/go-xcode/models"
	"github.com/bitrise-io/go-xcode/profileutil"
	"github.com/bitrise-io/go-xcode/utility"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/certdownloader"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/codesignasset"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/devportalclient"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/localcodesignasset"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/profiledownloader"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/projectmanager"
	"github.com/bitrise-io/go-xcode/v2/codesign"
	"github.com/bitrise-io/go-xcode/v2/exportoptionsgenerator"
	"github.com/bitrise-io/go-xcode/v2/xcconfig"
	cache "github.com/bitrise-io/go-xcode/v2/xcodecache"
	"github.com/bitrise-io/go-xcode/v2/xcpretty"
	"github.com/bitrise-io/go-xcode/xcarchive"
	"github.com/bitrise-io/go-xcode/xcodebuild"
	v1xcpretty "github.com/bitrise-io/go-xcode/xcpretty"
	"github.com/bitrise-steplib/steps-xcode-archive/utils"
	"github.com/kballard/go-shellquote"
	"howett.net/plist"
)

const (
	minSupportedXcodeMajorVersion = 9
)

const (
	// Deployed Outputs (moved to the OutputDir)
	bitriseXCArchiveZipPthEnvKey = "BITRISE_XCARCHIVE_ZIP_PATH"
	bitriseDSYMPthEnvKey         = "BITRISE_DSYM_PATH"
	bitriseIPAPthEnvKey          = "BITRISE_IPA_PATH"
	// Deployed logs
	xcodebuildArchiveLogPathEnvKey       = "BITRISE_XCODEBUILD_ARCHIVE_LOG_PATH"
	xcodebuildExportArchiveLogPathEnvKey = "BITRISE_XCODEBUILD_EXPORT_ARCHIVE_LOG_PATH"
	bitriseIDEDistributionLogsPthEnvKey  = "BITRISE_IDEDISTRIBUTION_LOGS_PATH"

	// Env Outputs
	bitriseAppDirPthEnvKey    = "BITRISE_APP_DIR_PATH"
	bitriseDSYMDirPthEnvKey   = "BITRISE_DSYM_DIR_PATH"
	bitriseXCArchivePthEnvKey = "BITRISE_XCARCHIVE_PATH"

	// Code Signing Authentication Source
	codeSignSourceOff     = "off"
	codeSignSourceAPIKey  = "api-key"
	codeSignSourceAppleID = "apple-id"
)

// Inputs ...
type Inputs struct {
	ExportMethod               string `env:"distribution_method,opt[app-store,ad-hoc,enterprise,development]"`
	UploadBitcode              bool   `env:"upload_bitcode,opt[yes,no]"`
	CompileBitcode             bool   `env:"compile_bitcode,opt[yes,no]"`
	ICloudContainerEnvironment string `env:"icloud_container_environment"`
	ExportDevelopmentTeam      string `env:"export_development_team"`

	ExportOptionsPlistContent string `env:"export_options_plist_content"`

	LogFormatter       string `env:"log_formatter,opt[xcpretty,xcodebuild]"`
	ProjectPath        string `env:"project_path,file"`
	Scheme             string `env:"scheme,required"`
	Configuration      string `env:"configuration"`
	OutputDir          string `env:"output_dir,required"`
	PerformCleanAction bool   `env:"perform_clean_action,opt[yes,no]"`
	XcodebuildOptions  string `env:"xcodebuild_options"`
	XcconfigContent    string `env:"xcconfig_content"`

	ExportAllDsyms bool   `env:"export_all_dsyms,opt[yes,no]"`
	ArtifactName   string `env:"artifact_name"`
	VerboseLog     bool   `env:"verbose_log,opt[yes,no]"`

	CacheLevel string `env:"cache_level,opt[none,swift_packages]"`

	CodeSigningAuthSource           string          `env:"automatic_code_signing,opt[off,api-key,apple-id]"`
	CertificateURLList              string          `env:"certificate_url_list"`
	CertificatePassphraseList       stepconf.Secret `env:"passphrase_list"`
	KeychainPath                    string          `env:"keychain_path"`
	KeychainPassword                stepconf.Secret `env:"keychain_password"`
	RegisterTestDevices             bool            `env:"register_test_devices,opt[yes,no]"`
	MinDaysProfileValid             int             `env:"min_profile_validity,required"`
	FallbackProvisioningProfileURLs string          `env:"fallback_provisioning_profile_url_list"`
	APIKeyPath                      stepconf.Secret `env:"api_key_path"`
	APIKeyID                        string          `env:"api_key_id"`
	APIKeyIssuerID                  string          `env:"api_key_issuer_id"`
	BuildURL                        string          `env:"BITRISE_BUILD_URL"`
	BuildAPIToken                   stepconf.Secret `env:"BITRISE_BUILD_API_TOKEN"`
}

// Config ...
type Config struct {
	Inputs
	XcodeMajorVersion           int
	XcodebuildAdditionalOptions []string
	CodesignManager             *codesign.Manager // nil if automatic code signing is "off"
}

var envRepository = env.NewRepository()
var cmdFactory = command.NewFactory(envRepository)
var logger = log.NewLogger()

func findIDEDistrubutionLogsPath(output string) (string, error) {
	pattern := `IDEDistribution: -\[IDEDistributionLogging _createLoggingBundleAtPath:\]: Created bundle at path '(?P<log_path>.*)'`
	re := regexp.MustCompile(pattern)

	logger.Printf("Locating IDE distrubution logs path")

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if match := re.FindStringSubmatch(line); len(match) == 2 {
			logger.Printf("Located IDE distrubution logs path")

			return match[1], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	logger.Printf("IDE distrubution logs path not found")

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
		logger.Printf("auto-detect export method specified: using the archive profile's export method: %s", archiveExportMethod)
		return archiveExportMethod, nil
	}

	exportMethod, err := exportoptions.ParseMethod(desiredExportMethod)
	if err != nil {
		return "", fmt.Errorf("failed to parse export method: %s", err)
	}
	logger.Printf("export method specified: %s", desiredExportMethod)

	return exportMethod, nil
}

func exportDSYMs(dsymDir string, dsyms []string) error {
	for _, dsym := range dsyms {
		if err := v1command.CopyDir(dsym, dsymDir, false); err != nil {
			return fmt.Errorf("could not copy (%s) to directory (%s): %s", dsym, dsymDir, err)
		}
	}
	return nil
}

type xcodeVersionProvider interface {
	GetXcodeVersion() (models.XcodebuildVersionModel, error)
}

type xcodebuildXcodeVersionProvider struct {
}

func newXcodebuildXcodeVersionProvider() xcodebuildXcodeVersionProvider {
	return xcodebuildXcodeVersionProvider{}
}

// GetXcodeVersion ...
func (p xcodebuildXcodeVersionProvider) GetXcodeVersion() (models.XcodebuildVersionModel, error) {
	return utility.GetXcodeVersion()
}

// XcodeArchiveStep ...
type XcodeArchiveStep struct {
	xcodeVersionProvider xcodeVersionProvider
	stepInputParser      stepconf.InputParser
	pathProvider         pathutil.PathProvider
	pathChecker          pathutil.PathChecker
	pathModifier         pathutil.PathModifier
	fileManager          fileutil.FileManager
}

// NewXcodeArchiveStep ...
func NewXcodeArchiveStep() XcodeArchiveStep {
	return XcodeArchiveStep{
		xcodeVersionProvider: newXcodebuildXcodeVersionProvider(),
		stepInputParser:      stepconf.NewInputParser(env.NewRepository()),
		pathProvider:         pathutil.NewPathProvider(),
		pathChecker:          pathutil.NewPathChecker(),
		pathModifier:         pathutil.NewPathModifier(),
		fileManager:          fileutil.NewFileManager(),
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
	logger.EnableDebugLog(config.VerboseLog)
	v1log.SetEnableDebugLog(config.VerboseLog) // For compatibility

	var err error
	config.XcodebuildAdditionalOptions, err = shellquote.Split(inputs.XcodebuildOptions)
	if err != nil {
		return Config{}, fmt.Errorf("provided XcodebuildOptions (%s) are not valid CLI parameters: %s", inputs.XcodebuildOptions, err)
	}

	if strings.TrimSpace(config.XcconfigContent) == "" {
		config.XcconfigContent = ""
	}
	if sliceutil.IsStringInSlice("-xcconfig", config.XcodebuildAdditionalOptions) &&
		config.XcconfigContent != "" {
		return Config{}, fmt.Errorf("`-xcconfig` option found in XcodebuildOptions (`xcodebuild_options`), please clear Build settings (xcconfig) (`xcconfig_content`) input as only one can be set")
	}

	if config.ExportOptionsPlistContent != "" {
		var options map[string]interface{}
		if _, err := plist.Unmarshal([]byte(config.ExportOptionsPlistContent), &options); err != nil {
			return Config{}, fmt.Errorf("issue with input ExportOptionsPlistContent: " + err.Error())
		}
	}

	if filepath.Ext(config.ProjectPath) != ".xcodeproj" && filepath.Ext(config.ProjectPath) != ".xcworkspace" {
		return Config{}, fmt.Errorf("issue with input ProjectPath: should be and .xcodeproj or .xcworkspace path")
	}

	logger.Infof("Xcode version:")

	// Detect Xcode major version
	xcodebuildVersion, err := s.xcodeVersionProvider.GetXcodeVersion()
	if err != nil {
		return Config{}, fmt.Errorf("failed to determine xcode version, error: %s", err)
	}
	logger.Printf("%s (%s)", xcodebuildVersion.Version, xcodebuildVersion.BuildVersion)

	xcodeMajorVersion := xcodebuildVersion.MajorVersion
	if xcodeMajorVersion < minSupportedXcodeMajorVersion {
		return Config{}, fmt.Errorf("invalid xcode major version (%d), should not be less then min supported: %d", xcodeMajorVersion, minSupportedXcodeMajorVersion)
	}
	config.XcodeMajorVersion = int(xcodeMajorVersion)

	// Validation ExportOptionsPlistContent
	exportOptionsPlistContent := strings.TrimSpace(config.ExportOptionsPlistContent)
	if exportOptionsPlistContent != config.ExportOptionsPlistContent {
		fmt.Println()
		logger.Warnf("ExportOptionsPlistContent is stripped to remove spaces and new lines:")
		logger.Printf(exportOptionsPlistContent)
	}

	if exportOptionsPlistContent != "" {
		fmt.Println()
		logger.Warnf("Ignoring the following options because ExportOptionsPlistContent provided:")
		logger.Printf("- DistributionMethod: %s", config.ExportMethod)
		logger.Printf("- UploadBitcode: %s", config.UploadBitcode)
		logger.Printf("- CompileBitcode: %s", config.CompileBitcode)
		logger.Printf("- ExportDevelopmentTeam: %s", config.ExportDevelopmentTeam)
		logger.Printf("- ICloudContainerEnvironment: %s", config.ICloudContainerEnvironment)
		fmt.Println()
	}
	config.ExportOptionsPlistContent = exportOptionsPlistContent

	absProjectPath, err := filepath.Abs(config.ProjectPath)
	if err != nil {
		return Config{}, fmt.Errorf("failed to get absolute project path, error: %s", err)
	}
	config.ProjectPath = absProjectPath

	// abs out dir pth
	absOutputDir, err := v1pathutil.AbsPath(config.OutputDir)
	if err != nil {
		return Config{}, fmt.Errorf("failed to expand OutputDir (%s), error: %s", config.OutputDir, err)
	}
	config.OutputDir = absOutputDir

	if exist, err := v1pathutil.IsPathExists(config.OutputDir); err != nil {
		return Config{}, fmt.Errorf("failed to check if OutputDir exist, error: %s", err)
	} else if !exist {
		if err := os.MkdirAll(config.OutputDir, 0777); err != nil {
			return Config{}, fmt.Errorf("failed to create OutputDir (%s), error: %s", config.OutputDir, err)
		}
	}

	if config.CodeSigningAuthSource != codeSignSourceOff {
		codesignManager, err := s.createCodesignManager(config)
		logger.Debugf("Codesignmanager created")
		if err != nil {
			return Config{}, err
		}
		config.CodesignManager = &codesignManager
	}

	logger.Debugf("Returning config")

	return config, nil
}

func (s XcodeArchiveStep) createCodesignManager(config Config) (codesign.Manager, error) {
	var authType codesign.AuthType
	switch config.CodeSigningAuthSource {
	case codeSignSourceAppleID:
		authType = codesign.AppleIDAuth
	case codeSignSourceAPIKey:
		authType = codesign.APIKeyAuth
	case codeSignSourceOff:
		return codesign.Manager{}, fmt.Errorf("automatic code signing is disabled")
	}

	codesignInputs := codesign.Input{
		AuthType:                     authType,
		DistributionMethod:           config.ExportMethod,
		CertificateURLList:           config.CertificateURLList,
		CertificatePassphraseList:    config.CertificatePassphraseList,
		KeychainPath:                 config.KeychainPath,
		KeychainPassword:             config.KeychainPassword,
		FallbackProvisioningProfiles: config.FallbackProvisioningProfileURLs,
	}

	codesignConfig, err := codesign.ParseConfig(codesignInputs, cmdFactory)
	if err != nil {
		return codesign.Manager{}, fmt.Errorf("issue with input: %s", err)
	}

	devPortalClientFactory := devportalclient.NewFactory(logger)

	var serviceConnection *devportalservice.AppleDeveloperConnection = nil
	if config.BuildURL != "" && config.BuildAPIToken != "" {
		if serviceConnection, err = devPortalClientFactory.CreateBitriseConnection(config.BuildURL, string(config.BuildAPIToken)); err != nil {
			return codesign.Manager{}, err
		}
	}

	connectionInputs := codesign.ConnectionOverrideInputs{
		APIKeyPath:     config.Inputs.APIKeyPath,
		APIKeyID:       config.Inputs.APIKeyID,
		APIKeyIssuerID: config.Inputs.APIKeyIssuerID,
	}

	appleAuthCredentials, err := codesign.SelectConnectionCredentials(authType, serviceConnection, connectionInputs, logger)
	if err != nil {
		return codesign.Manager{}, err
	}

	opts := codesign.Opts{
		AuthType:                   authType,
		ShouldConsiderXcodeSigning: true,
		TeamID:                     config.ExportDevelopmentTeam,
		ExportMethod:               codesignConfig.DistributionMethod,
		XcodeMajorVersion:          config.XcodeMajorVersion,
		RegisterTestDevices:        config.RegisterTestDevices,
		SignUITests:                false,
		MinDaysProfileValidity:     config.MinDaysProfileValid,
		IsVerboseLog:               config.VerboseLog,
	}

	project, err := projectmanager.NewProject(projectmanager.InitParams{
		ProjectOrWorkspacePath: config.ProjectPath,
		SchemeName:             config.Scheme,
		ConfigurationName:      config.Configuration,
	})
	if err != nil {
		return codesign.Manager{}, err
	}

	logger.Debugf("Project parsed")

	client := retry.NewHTTPClient().StandardClient()
	logger.Debugf("HTTP client created")
	return codesign.NewManagerWithProject(
		opts,
		appleAuthCredentials,
		serviceConnection.TestDevices,
		devPortalClientFactory,
		certdownloader.NewDownloader(codesignConfig.CertificatesAndPassphrases, client),
		profiledownloader.New(codesignConfig.FallbackProvisioningProfiles, client),
		codesignasset.NewWriter(codesignConfig.Keychain),
		localcodesignasset.NewManager(localcodesignasset.NewProvisioningProfileProvider(), localcodesignasset.NewProvisioningProfileConverter()),
		project,
		logger,
	), nil
}

// EnsureDependenciesOpts ...
type EnsureDependenciesOpts struct {
	XCPretty bool
}

// EnsureDependencies ...
func (s XcodeArchiveStep) EnsureDependencies(opts EnsureDependenciesOpts) error {
	if !opts.XCPretty {
		return nil
	}

	fmt.Println()
	logger.Infof("Checking if log formatter (xcpretty) is installed")

	var xcpretty = xcpretty.NewXcpretty(log.NewLogger())

	installed, err := xcpretty.IsInstalled()
	if err != nil {
		return fmt.Errorf("failed to check if xcpretty is installed, error: %s", err)
	} else if !installed {
		logger.Warnf(`xcpretty is not installed`)
		fmt.Println()
		logger.Printf("Installing xcpretty")

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

	xcprettyVersion, err := xcpretty.Version()
	if err != nil {
		return fmt.Errorf("failed to determine xcpretty version, error: %s", err)
	}
	logger.Printf("- xcprettyVersion: %s", xcprettyVersion.String())

	return nil
}

type xcodeArchiveOpts struct {
	ProjectPath       string
	Scheme            string
	Configuration     string
	LogFormatter      string
	XcodeMajorVersion int
	ArtifactName      string
	XcodeAuthOptions  *xcodebuild.AuthenticationParams

	PerformCleanAction bool
	XcconfigContent    string
	AdditionalOptions  []string

	CacheLevel string
}

type xcodeArchiveOutput struct {
	Archive              *xcarchive.IosArchive
	XcodebuildArchiveLog string
}

func (s XcodeArchiveStep) xcodeArchive(opts xcodeArchiveOpts) (xcodeArchiveOutput, error) {
	out := xcodeArchiveOutput{}

	// Open Xcode project
	logger.TInfof("Opening xcode project at path: %s for scheme: %s", opts.ProjectPath, opts.Scheme)

	xcodeProj, scheme, configuration, err := utils.OpenArchivableProject(opts.ProjectPath, opts.Scheme, opts.Configuration)
	if err != nil {
		return out, fmt.Errorf("failed to open project: %s: %s", opts.ProjectPath, err)
	}

	logger.TInfof("Reading xcode project")

	platform, err := utils.BuildableTargetPlatform(xcodeProj, scheme, configuration, utils.XcodeBuild{})
	if err != nil {
		return out, fmt.Errorf("failed to read project platform: %s: %s", opts.ProjectPath, err)
	}

	logger.TInfof("Reading main target")

	mainTarget, err := exportoptionsgenerator.ArchivableApplicationTarget(xcodeProj, scheme)
	if err != nil {
		return out, fmt.Errorf("failed to read main application target: %s", err)
	}
	if mainTarget.ProductType == exportoptionsgenerator.AppClipProductType {
		logger.Errorf("Selected scheme: '%s' targets an App Clip target (%s),", opts.Scheme, mainTarget.Name)
		logger.Errorf("'Xcode Archive & Export for iOS' step is intended to archive the project using a scheme targeting an Application target.")
		logger.Errorf("Please select a scheme targeting an Application target to archive and export the main Application")
		logger.Errorf("and use 'Export iOS and tvOS Xcode archive' step to export an App Clip.")
		os.Exit(1)
	}

	// Create the Archive with Xcode Command Line tools
	logger.Println()
	logger.TInfof("Creating the Archive ...")

	var actions []string
	if opts.PerformCleanAction {
		actions = []string{"clean", "archive"}
	} else {
		actions = []string{"archive"}
	}

	archiveCmd := xcodebuild.NewCommandBuilder(opts.ProjectPath, actions...)
	archiveCmd.SetScheme(opts.Scheme)
	archiveCmd.SetConfiguration(opts.Configuration)

	if opts.XcconfigContent != "" {
		xcconfigWriter := xcconfig.NewWriter(s.pathProvider, s.fileManager, s.pathChecker, s.pathModifier)
		xcconfigPath, err := xcconfigWriter.Write(opts.XcconfigContent)
		if err != nil {
			return out, fmt.Errorf("failed to write xcconfig file contents: %w", err)
		}
		archiveCmd.SetXCConfigPath(xcconfigPath)
	}

	tmpDir, err := v1pathutil.NormalizedOSTempDirPath("xcodeArchive")
	if err != nil {
		return out, fmt.Errorf("failed to create temp dir, error: %s", err)
	}
	archivePth := filepath.Join(tmpDir, opts.ArtifactName+".xcarchive")

	archiveCmd.SetArchivePath(archivePth)
	if opts.XcodeAuthOptions != nil {
		archiveCmd.SetAuthentication(*opts.XcodeAuthOptions)
	}

	additionalOptions := generateAdditionalOptions(string(platform), opts.AdditionalOptions)
	archiveCmd.SetCustomOptions(additionalOptions)

	var swiftPackagesPath string
	if opts.XcodeMajorVersion >= 11 {
		var err error
		if swiftPackagesPath, err = cache.NewSwiftPackageCache().SwiftPackagesPath(opts.ProjectPath); err != nil {
			return out, fmt.Errorf("failed to get Swift Packages path, error: %s", err)
		}
	}

	logger.Infof("Starting the Archive ...")

	xcodebuildLog, err := runArchiveCommandWithRetry(archiveCmd, opts.LogFormatter == "xcpretty", swiftPackagesPath)
	out.XcodebuildArchiveLog = xcodebuildLog
	if err != nil || opts.LogFormatter == "xcodebuild" {
		const lastLinesMsg = "\nLast lines of the Xcode's build log:"
		if err != nil {
			logger.Infof(colorstring.Red(lastLinesMsg))
		} else {
			logger.Infof(lastLinesMsg)
		}
		fmt.Println(stringutil.LastNLines(xcodebuildLog, 20))

		logger.Warnf(`You can find the last couple of lines of Xcode's build log above, but the full log will be also available in the raw-xcodebuild-output.log
The log file will be stored in $BITRISE_DEPLOY_DIR, and its full path will be available in the $BITRISE_XCODE_RAW_RESULT_TEXT_PATH environment variable.`)
	}
	if err != nil {
		return out, fmt.Errorf("archive failed, error: %s", err)
	}

	// Ensure xcarchive exists
	if exist, err := v1pathutil.IsPathExists(archivePth); err != nil {
		return out, fmt.Errorf("failed to check if archive exist, error: %s", err)
	} else if !exist {
		return out, fmt.Errorf("no archive generated at: %s", archivePth)
	}

	archive, err := xcarchive.NewIosArchive(archivePth)
	if err != nil {
		return out, fmt.Errorf("failed to parse archive, error: %s", err)
	}
	out.Archive = &archive

	mainApplication := archive.Application

	fmt.Println()
	logger.Infof("Archive info:")
	logger.Printf("team: %s (%s)", mainApplication.ProvisioningProfile.TeamName, mainApplication.ProvisioningProfile.TeamID)
	logger.Printf("profile: %s (%s)", mainApplication.ProvisioningProfile.Name, mainApplication.ProvisioningProfile.UUID)
	logger.Printf("export: %s", mainApplication.ProvisioningProfile.ExportType)
	logger.Printf("xcode managed profile: %v", profileutil.IsXcodeManaged(mainApplication.ProvisioningProfile.Name))

	// Cache swift PM
	if opts.XcodeMajorVersion >= 11 && opts.CacheLevel == "swift_packages" {
		if err := cache.NewSwiftPackageCache().CollectSwiftPackages(opts.ProjectPath); err != nil {
			logger.Warnf("Failed to mark swift packages for caching, error: %s", err)
		}
	}

	return out, nil
}

type xcodeIPAExportOpts struct {
	ProjectPath       string
	Scheme            string
	Configuration     string
	LogFormatter      string
	XcodeMajorVersion int
	XcodeAuthOptions  *xcodebuild.AuthenticationParams

	Archive                         xcarchive.IosArchive
	CustomExportOptionsPlistContent string
	ExportMethod                    string
	ICloudContainerEnvironment      string
	ExportDevelopmentTeam           string
	UploadBitcode                   bool
	CompileBitcode                  bool
}

type xcodeIPAExportOutput struct {
	ExportOptionsPath          string
	IPAExportDir               string
	XcodebuildExportArchiveLog string
	IDEDistrubutionLogsDir     string
}

func (s XcodeArchiveStep) xcodeIPAExport(opts xcodeIPAExportOpts) (xcodeIPAExportOutput, error) {
	out := xcodeIPAExportOutput{}

	// Exporting the ipa with Xcode Command Line tools

	/*
		You'll get an "Error Domain=IDEDistributionErrorDomain Code=14 "No applicable devices found."" error
		if $GEM_HOME is set and the project's directory includes a Gemfile - to fix this
		we'll unset GEM_HOME as that's not required for xcodebuild anyway.
		This probably fixes the RVM issue too, but that still should be tested.
		See also:
		- http://stackoverflow.com/questions/33041109/xcodebuild-no-applicable-devices-found-when-exporting-archive
		- https://gist.github.com/claybridges/cea5d4afd24eda268164
	*/
	envsToUnset := []string{"GEM_HOME", "GEM_PATH", "RUBYLIB", "RUBYOPT", "BUNDLE_BIN_PATH", "_ORIGINAL_GEM_PATH", "BUNDLE_GEMFILE"}
	for _, key := range envsToUnset {
		if err := os.Unsetenv(key); err != nil {
			return out, fmt.Errorf("failed to unset (%s), error: %s", key, err)
		}
	}

	fmt.Println()
	logger.Infof("Exporting ipa from the archive...")

	tmpDir, err := v1pathutil.NormalizedOSTempDirPath("xcodeIPAExport")
	if err != nil {
		return out, fmt.Errorf("failed to create temp dir, error: %s", err)
	}

	exportOptionsPath := filepath.Join(tmpDir, "export_options.plist")

	if opts.CustomExportOptionsPlistContent != "" {
		logger.Printf("Custom export options content provided, using it:")
		fmt.Println(opts.CustomExportOptionsPlistContent)

		if err := v1fileutil.WriteStringToFile(exportOptionsPath, opts.CustomExportOptionsPlistContent); err != nil {
			return out, fmt.Errorf("failed to write export options to file, error: %s", err)
		}
	} else {
		logger.Printf("No custom export options content provided, generating export options...")

		archiveExportMethod := opts.Archive.Application.ProvisioningProfile.ExportType

		exportMethod, err := determineExportMethod(opts.ExportMethod, archiveExportMethod)
		if err != nil {
			return out, err
		}

		logger.TPrintf("Opening Xcode project at path: %s.", opts.ProjectPath)

		xcodeProj, scheme, configuration, err := utils.OpenArchivableProject(opts.ProjectPath, opts.Scheme, opts.Configuration)
		if err != nil {
			return out, fmt.Errorf("failed to open project: %s: %s", opts.ProjectPath, err)
		}

		archiveCodeSignIsXcodeManaged := opts.Archive.IsXcodeManaged()

		generator := exportoptionsgenerator.New(xcodeProj, scheme, configuration, logger)
		exportOptions, err := generator.GenerateApplicationExportOptions(exportMethod, opts.ICloudContainerEnvironment, opts.ExportDevelopmentTeam,
			opts.UploadBitcode, opts.CompileBitcode, archiveCodeSignIsXcodeManaged, int64(opts.XcodeMajorVersion))
		if err != nil {
			return out, err
		}

		fmt.Println()
		logger.Printf("generated export options content:")
		fmt.Println()
		fmt.Println(exportOptions.String())

		if err := exportOptions.WriteToFile(exportOptionsPath); err != nil {
			return out, err
		}
	}

	ipaExportDir := filepath.Join(tmpDir, "exported")

	exportCmd := xcodebuild.NewExportCommand()
	exportCmd.SetArchivePath(opts.Archive.Path)
	exportCmd.SetExportDir(ipaExportDir)
	exportCmd.SetExportOptionsPlist(exportOptionsPath)
	if opts.XcodeAuthOptions != nil {
		exportCmd.SetAuthentication(*opts.XcodeAuthOptions)
	}

	if opts.LogFormatter == "xcpretty" {
		xcprettyCmd := v1xcpretty.New(exportCmd)

		fmt.Println()
		logWithTimestamp(colorstring.Green, xcprettyCmd.PrintableCmd())

		logger.Infof("Running export ipa from the archive command.")

		xcodebuildLog, exportErr := xcprettyCmd.Run()
		out.XcodebuildExportArchiveLog = xcodebuildLog
		if exportErr != nil {
			logger.Warnf(`If you can't find the reason of the error in the log, please check the raw-xcodebuild-output.log
The log file is stored in $BITRISE_DEPLOY_DIR, and its full path
is available in the $BITRISE_XCODE_RAW_RESULT_TEXT_PATH environment variable`)

			// xcdistributionlogs
			ideDistrubutionLogsDir, err := findIDEDistrubutionLogsPath(xcodebuildLog)
			if err != nil {
				logger.Warnf("Failed to find xcdistributionlogs, error: %s", err)
			} else {
				out.IDEDistrubutionLogsDir = ideDistrubutionLogsDir

				criticalDistLogFilePth := filepath.Join(ideDistrubutionLogsDir, "IDEDistribution.critical.log")
				logger.Warnf("IDEDistribution.critical.log:")
				if criticalDistLog, err := v1fileutil.ReadStringFromFile(criticalDistLogFilePth); err == nil {
					logger.Printf(criticalDistLog)
				}

				logger.Warnf(`Also please check the xcdistributionlogs
The logs directory is stored in $BITRISE_DEPLOY_DIR, and its full path
is available in the $BITRISE_IDEDISTRIBUTION_LOGS_PATH environment variable`)
			}

			return out, fmt.Errorf("export failed, error: %s", exportErr)
		}
	} else {
		fmt.Println()
		logWithTimestamp(colorstring.Green, exportCmd.PrintableCmd())

		logger.Infof("Running export ipa from the archive command.")

		xcodebuildLog, exportErr := exportCmd.RunAndReturnOutput()
		out.XcodebuildExportArchiveLog = xcodebuildLog
		if exportErr != nil {
			// xcdistributionlogs
			ideDistrubutionLogsDir, err := findIDEDistrubutionLogsPath(xcodebuildLog)
			if err != nil {
				logger.Warnf("Failed to find xcdistributionlogs, error: %s", err)
			} else {
				out.IDEDistrubutionLogsDir = ideDistrubutionLogsDir

				criticalDistLogFilePth := filepath.Join(ideDistrubutionLogsDir, "IDEDistribution.critical.log")
				logger.Warnf("IDEDistribution.critical.log:")
				if criticalDistLog, err := v1fileutil.ReadStringFromFile(criticalDistLogFilePth); err == nil {
					logger.Printf(criticalDistLog)
				}

				logger.Warnf(`If you can't find the reason of the error in the log, please check the xcdistributionlogs
The logs directory is stored in $BITRISE_DEPLOY_DIR, and its full path
is available in the $BITRISE_IDEDISTRIBUTION_LOGS_PATH environment variable`)
			}

			return out, fmt.Errorf("export failed, error: %s", exportErr)
		}
	}

	out.ExportOptionsPath = exportOptionsPath
	out.IPAExportDir = ipaExportDir

	return out, nil
}

// RunOpts ...
type RunOpts struct {
	// Shared
	ProjectPath       string
	Scheme            string
	Configuration     string
	LogFormatter      string
	XcodeMajorVersion int
	ArtifactName      string

	// Code signing, nil if automatic code signing is "off"
	CodesignManager *codesign.Manager

	// Archive
	PerformCleanAction          bool
	XcconfigContent             string
	XcodebuildAdditionalOptions []string
	CacheLevel                  string

	// IPA Export
	CustomExportOptionsPlistContent string
	ExportMethod                    string
	ICloudContainerEnvironment      string
	ExportDevelopmentTeam           string
	UploadBitcode                   bool
	CompileBitcode                  bool
}

// RunOut ...
type RunOut struct {
	Archive      *xcarchive.IosArchive
	ArtifactName string

	ExportOptionsPath string
	IPAExportDir      string

	XcodebuildArchiveLog       string
	XcodebuildExportArchiveLog string
	IDEDistrubutionLogsDir     string
}

// Run ...
func (s XcodeArchiveStep) Run(opts RunOpts) (RunOut, error) {
	var (
		out         = RunOut{}
		authOptions *xcodebuild.AuthenticationParams
	)

	logger.Println()
	if opts.XcodeMajorVersion >= 11 {
		logger.Infof("Running resolve Swift package dependencies")
		// Resolve Swift package dependencies, so running -showBuildSettings later is faster later
		// Specifying a scheme is required for workspaces
		resolveDepsCmd := xcodebuild.NewResolvePackagesCommandModel(opts.ProjectPath, opts.Scheme, opts.Configuration)
		resolveDepsCmd.SetCustomOptions(opts.XcodebuildAdditionalOptions)
		if err := resolveDepsCmd.Run(); err != nil {
			logger.Warnf("%s", err)
		}
	}

	if opts.ArtifactName == "" {
		logger.Infof("Looking for artifact name as field is empty")

		cmdModel := xcodebuild.NewShowBuildSettingsCommand(opts.ProjectPath)
		cmdModel.SetScheme(opts.Scheme)
		cmdModel.SetConfiguration(opts.Configuration)
		settings, err := cmdModel.RunAndReturnSettings()
		if err != nil {
			return out, fmt.Errorf("failed to read build settings: %w", err)
		}
		productName, err := settings.String("PRODUCT_NAME")
		if err != nil || productName == "" {
			logger.Warnf("Product name not found in build settings, using scheme (%s) as artifact name", opts.Scheme)
			productName = opts.Scheme
		}

		opts.ArtifactName = productName
	}
	out.ArtifactName = opts.ArtifactName

	if opts.CodesignManager != nil {
		logger.Infof("Preparing code signing assets (certificates, profiles) before Archive action")

		xcodebuildAuthParams, err := opts.CodesignManager.PrepareCodesigning()
		if err != nil {
			return RunOut{}, fmt.Errorf("failed to manage code signing: %s", err)
		}

		if xcodebuildAuthParams != nil {
			privateKey, err := xcodebuildAuthParams.WritePrivateKeyToFile()
			if err != nil {
				return RunOut{}, err
			}

			defer func() {
				if err := os.Remove(privateKey); err != nil {
					logger.Warnf("failed to remove private key file: %s", err)
				}
			}()

			authOptions = &xcodebuild.AuthenticationParams{
				KeyID:     xcodebuildAuthParams.KeyID,
				IsssuerID: xcodebuildAuthParams.IssuerID,
				KeyPath:   privateKey,
			}
		}
	} else {
		logger.Infof("Automatic code signing is disabled, skipped downloading code sign assets")
	}
	logger.Println()

	archiveOpts := xcodeArchiveOpts{
		ProjectPath:       opts.ProjectPath,
		Scheme:            opts.Scheme,
		Configuration:     opts.Configuration,
		LogFormatter:      opts.LogFormatter,
		XcodeMajorVersion: opts.XcodeMajorVersion,
		ArtifactName:      opts.ArtifactName,
		XcodeAuthOptions:  authOptions,

		PerformCleanAction: opts.PerformCleanAction,
		XcconfigContent:    opts.XcconfigContent,
		AdditionalOptions:  opts.XcodebuildAdditionalOptions,
		CacheLevel:         opts.CacheLevel,
	}
	archiveOut, err := s.xcodeArchive(archiveOpts)
	out.XcodebuildArchiveLog = archiveOut.XcodebuildArchiveLog
	if err != nil {
		return out, err
	}

	out.Archive = archiveOut.Archive

	IPAExportOpts := xcodeIPAExportOpts{
		ProjectPath:       opts.ProjectPath,
		Scheme:            opts.Scheme,
		Configuration:     opts.Configuration,
		LogFormatter:      opts.LogFormatter,
		XcodeMajorVersion: opts.XcodeMajorVersion,
		XcodeAuthOptions:  authOptions,

		Archive:                         *archiveOut.Archive,
		CustomExportOptionsPlistContent: opts.CustomExportOptionsPlistContent,
		ExportMethod:                    opts.ExportMethod,
		ICloudContainerEnvironment:      opts.ICloudContainerEnvironment,
		ExportDevelopmentTeam:           opts.ExportDevelopmentTeam,
		UploadBitcode:                   opts.UploadBitcode,
		CompileBitcode:                  opts.CompileBitcode,
	}
	exportOut, err := s.xcodeIPAExport(IPAExportOpts)
	out.XcodebuildExportArchiveLog = exportOut.XcodebuildExportArchiveLog
	if err != nil {
		out.IDEDistrubutionLogsDir = exportOut.IDEDistrubutionLogsDir
		return out, err
	}

	out.ExportOptionsPath = exportOut.ExportOptionsPath
	out.IPAExportDir = exportOut.IPAExportDir

	return out, nil
}

// ExportOpts ...
type ExportOpts struct {
	OutputDir      string
	ArtifactName   string
	ExportAllDsyms bool

	Archive *xcarchive.IosArchive

	ExportOptionsPath string
	IPAExportDir      string

	XcodebuildArchiveLog       string
	XcodebuildExportArchiveLog string
	IDEDistrubutionLogsDir     string
}

// ExportOutput ...
func (s XcodeArchiveStep) ExportOutput(opts ExportOpts) error {
	fmt.Println()
	logger.Infof("Exporting outputs...")

	cleanup := func(pth string) error {
		if exist, err := v1pathutil.IsPathExists(pth); err != nil {
			return fmt.Errorf("failed to check if path (%s) exist, error: %s", pth, err)
		} else if exist {
			if err := os.RemoveAll(pth); err != nil {
				return fmt.Errorf("failed to remove path (%s), error: %s", pth, err)
			}
		}
		return nil
	}

	if opts.Archive != nil {
		fmt.Println()
		archivePath := opts.Archive.Path
		if err := utils.ExportOutputDir(cmdFactory, archivePath, archivePath, bitriseXCArchivePthEnvKey); err != nil {
			return fmt.Errorf("failed to export %s, error: %s", bitriseXCArchivePthEnvKey, err)
		}
		logger.Donef("The xcarchive path is now available in the Environment Variable: %s (value: %s)", bitriseXCArchivePthEnvKey, archivePath)

		archiveZipPath := filepath.Join(opts.OutputDir, opts.ArtifactName+".xcarchive.zip")
		if err := cleanup(archiveZipPath); err != nil {
			return err
		}

		if err := utils.ExportOutputDirAsZip(cmdFactory, archivePath, archiveZipPath, bitriseXCArchiveZipPthEnvKey); err != nil {
			return fmt.Errorf("failed to export %s, error: %s", bitriseXCArchiveZipPthEnvKey, err)
		}
		logger.Donef("The xcarchive zip path is now available in the Environment Variable: %s (value: %s)", bitriseXCArchiveZipPthEnvKey, archiveZipPath)

		fmt.Println()
		appPath := filepath.Join(opts.OutputDir, opts.ArtifactName+".app")
		if err := cleanup(appPath); err != nil {
			return err
		}

		if err := utils.ExportOutputDir(cmdFactory, opts.Archive.Application.Path, appPath, bitriseAppDirPthEnvKey); err != nil {
			return fmt.Errorf("failed to export %s, error: %s", bitriseAppDirPthEnvKey, err)
		}
		logger.Donef("The app directory is now available in the Environment Variable: %s (value: %s)", bitriseAppDirPthEnvKey, appPath)

		logger.Printf("Looking for app and framework dSYMs.")

		appDSYMPaths, frameworkDSYMPaths, err := opts.Archive.FindDSYMs()
		if err != nil {
			return fmt.Errorf("failed to export dSYMs, error: %s", err)
		}

		appDSYMPathsCount := len(appDSYMPaths)
		frameworkDSYMPathsCount := len(frameworkDSYMPaths)

		logger.Printf("Found %s app dSYMs and framework dSYMs %s.", appDSYMPathsCount, frameworkDSYMPathsCount)

		if appDSYMPathsCount > 0 || frameworkDSYMPathsCount > 0 {
			fmt.Println()
			dsymDir, err := v1pathutil.NormalizedOSTempDirPath("__dsyms__")
			if err != nil {
				return fmt.Errorf("failed to create tmp dir, error: %s", err)
			}

			if appDSYMPathsCount > 0 {
				if err := exportDSYMs(dsymDir, appDSYMPaths); err != nil {
					return fmt.Errorf("failed to export dSYMs: %v", err)
				}
			} else {
				logger.Warnf("No app dSYMs found to export")
			}

			if opts.ExportAllDsyms && frameworkDSYMPathsCount > 0 {
				if err := exportDSYMs(dsymDir, frameworkDSYMPaths); err != nil {
					return fmt.Errorf("failed to export dSYMs: %v", err)
				}
			}

			if err := utils.ExportOutputDir(cmdFactory, dsymDir, dsymDir, bitriseDSYMDirPthEnvKey); err != nil {
				return fmt.Errorf("failed to export %s, error: %s", bitriseDSYMDirPthEnvKey, err)
			}
			logger.Donef("The dSYM dir path is now available in the Environment Variable: %s (value: %s)", bitriseDSYMDirPthEnvKey, dsymDir)

			dsymZipPath := filepath.Join(opts.OutputDir, opts.ArtifactName+".dSYM.zip")
			if err := cleanup(dsymZipPath); err != nil {
				return err
			}

			if err := utils.ExportOutputDirAsZip(cmdFactory, dsymDir, dsymZipPath, bitriseDSYMPthEnvKey); err != nil {
				return fmt.Errorf("failed to export %s, error: %s", bitriseDSYMPthEnvKey, err)
			}
			logger.Donef("The dSYM zip path is now available in the Environment Variable: %s (value: %s)", bitriseDSYMPthEnvKey, dsymZipPath)
		}
	}

	if opts.ExportOptionsPath != "" {
		fmt.Println()
		exportOptionsPath := filepath.Join(opts.OutputDir, "export_options.plist")
		if err := cleanup(exportOptionsPath); err != nil {
			return err
		}

		if err := v1command.CopyFile(opts.ExportOptionsPath, exportOptionsPath); err != nil {
			return err
		}
	}

	if opts.IPAExportDir != "" {
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
			logger.Errorf("No .ipa file found at export dir: %s", opts.IPAExportDir)
			logger.Printf("File list in the export dir:")
			for _, pth := range fileList {
				logger.Printf("- %s", pth)
			}
			return fmt.Errorf("")
		}

		fmt.Println()

		ipaPath := filepath.Join(opts.OutputDir, opts.ArtifactName+".ipa")
		if err := cleanup(ipaPath); err != nil {
			return err
		}

		if err := utils.ExportOutputFile(cmdFactory, ipaFiles[0], ipaPath, bitriseIPAPthEnvKey); err != nil {
			return fmt.Errorf("failed to export %s, error: %s", bitriseIPAPthEnvKey, err)
		}
		logger.Donef("The ipa path is now available in the Environment Variable: %s (value: %s)", bitriseIPAPthEnvKey, ipaPath)

		if len(ipaFiles) > 1 {
			logger.Warnf("More than 1 .ipa file found, exporting first one: %s", ipaFiles[0])
			logger.Warnf("Moving every ipa to the BITRISE_DEPLOY_DIR")

			for i, pth := range ipaFiles {
				if i == 0 {
					continue
				}

				base := filepath.Base(pth)
				deployPth := filepath.Join(opts.OutputDir, base)

				if err := v1command.CopyFile(pth, deployPth); err != nil {
					return fmt.Errorf("failed to copy (%s) -> (%s), error: %s", pth, deployPth, err)
				}
			}
		}
	}

	if opts.IDEDistrubutionLogsDir != "" {
		ideDistributionLogsZipPath := filepath.Join(opts.OutputDir, "xcodebuild.xcdistributionlogs.zip")
		if err := cleanup(ideDistributionLogsZipPath); err != nil {
			return err
		}

		if err := utils.ExportOutputDirAsZip(cmdFactory, opts.IDEDistrubutionLogsDir, ideDistributionLogsZipPath, bitriseIDEDistributionLogsPthEnvKey); err != nil {
			logger.Warnf("Failed to export %s, error: %s", bitriseIDEDistributionLogsPthEnvKey, err)
		} else {
			logger.Donef("The xcdistributionlogs zip path is now available in the Environment Variable: %s (value: %s)", bitriseIDEDistributionLogsPthEnvKey, ideDistributionLogsZipPath)
		}
	}

	if opts.XcodebuildArchiveLog != "" {
		xcodebuildArchiveLogPath := filepath.Join(opts.OutputDir, "xcodebuild-archive.log")
		if err := cleanup(xcodebuildArchiveLogPath); err != nil {
			return err
		}

		if err := utils.ExportOutputFileContent(cmdFactory, opts.XcodebuildArchiveLog, xcodebuildArchiveLogPath, xcodebuildArchiveLogPathEnvKey); err != nil {
			logger.Warnf("Failed to export %s, error: %s", xcodebuildArchiveLogPathEnvKey, err)
		} else {
			logger.Donef("The xcodebuild archive log path is now available in the Environment Variable: %s (value: %s)", xcodebuildArchiveLogPathEnvKey, xcodebuildArchiveLogPath)
		}
	}

	if opts.XcodebuildExportArchiveLog != "" {
		xcodebuildExportArchiveLogPath := filepath.Join(opts.OutputDir, "xcodebuild-export-archive.log")
		if err := cleanup(xcodebuildExportArchiveLogPath); err != nil {
			return err
		}

		if err := utils.ExportOutputFileContent(cmdFactory, opts.XcodebuildExportArchiveLog, xcodebuildExportArchiveLogPath, xcodebuildExportArchiveLogPathEnvKey); err != nil {
			logger.Warnf("Failed to export %s, error: %s", xcodebuildArchiveLogPathEnvKey, err)
		} else {
			logger.Donef("The xcodebuild -exportArchive log path is now available in the Environment Variable: %s (value: %s)", xcodebuildExportArchiveLogPathEnvKey, xcodebuildExportArchiveLogPath)
		}
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

	dependenciesOpts := EnsureDependenciesOpts{
		XCPretty: config.LogFormatter == "xcpretty",
	}
	if err := step.EnsureDependencies(dependenciesOpts); err != nil {
		logger.Warnf(err.Error())
		logger.Warnf("Switching to xcodebuild for output tool")
		config.LogFormatter = "xcodebuild"
	}

	runOpts := RunOpts{
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
	out, runErr := step.Run(runOpts)
	config.ArtifactName = out.ArtifactName

	exportOpts := ExportOpts{
		OutputDir:      config.OutputDir,
		ArtifactName:   config.ArtifactName,
		ExportAllDsyms: config.ExportAllDsyms,

		Archive: out.Archive,

		ExportOptionsPath: out.ExportOptionsPath,
		IPAExportDir:      out.IPAExportDir,

		XcodebuildArchiveLog:       out.XcodebuildArchiveLog,
		XcodebuildExportArchiveLog: out.XcodebuildExportArchiveLog,
		IDEDistrubutionLogsDir:     out.IDEDistrubutionLogsDir,
	}
	exportErr := step.ExportOutput(exportOpts)

	if runErr != nil {
		return runErr
	}
	if exportErr != nil {
		return exportErr
	}

	return nil
}

func main() {
	if err := RunStep(); err != nil {
		logger.Errorf(err.Error())
		os.Exit(1)
	}
}
