package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/steps-xcode-archive/utils"
	"github.com/bitrise-tools/go-xcode/exportoptions"
	"github.com/bitrise-tools/go-xcode/provisioningprofile"
	"github.com/bitrise-tools/go-xcode/xcarchive"
	"github.com/bitrise-tools/go-xcode/xcodebuild"
	"github.com/bitrise-tools/go-xcode/xcpretty"
	"github.com/kballard/go-shellquote"
)

const (
	minSupportedXcodeMajorVersion = 6
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

// ConfigsModel ...
type ConfigsModel struct {
	ExportMethod   string
	UploadBitcode  string
	CompileBitcode string
	TeamID         string

	UseDeprecatedExport               string
	ForceTeamID                       string
	ForceProvisioningProfileSpecifier string
	ForceProvisioningProfile          string
	ForceCodeSignIdentity             string
	CustomExportOptionsPlistContent   string

	OutputTool        string
	Workdir           string
	ProjectPath       string
	Scheme            string
	Configuration     string
	OutputDir         string
	IsCleanBuild      string
	XcodebuildOptions string

	IsExportXcarchiveZip string
	ExportAllDsyms       string
	ArtifactName         string
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		ExportMethod:   os.Getenv("export_method"),
		UploadBitcode:  os.Getenv("upload_bitcode"),
		CompileBitcode: os.Getenv("compile_bitcode"),
		TeamID:         os.Getenv("team_id"),

		UseDeprecatedExport:               os.Getenv("use_deprecated_export"),
		ForceTeamID:                       os.Getenv("force_team_id"),
		ForceProvisioningProfileSpecifier: os.Getenv("force_provisioning_profile_specifier"),
		ForceProvisioningProfile:          os.Getenv("force_provisioning_profile"),
		ForceCodeSignIdentity:             os.Getenv("force_code_sign_identity"),
		CustomExportOptionsPlistContent:   os.Getenv("custom_export_options_plist_content"),

		OutputTool:        os.Getenv("output_tool"),
		Workdir:           os.Getenv("workdir"),
		ProjectPath:       os.Getenv("project_path"),
		Scheme:            os.Getenv("scheme"),
		Configuration:     os.Getenv("configuration"),
		OutputDir:         os.Getenv("output_dir"),
		IsCleanBuild:      os.Getenv("is_clean_build"),
		XcodebuildOptions: os.Getenv("xcodebuild_options"),

		IsExportXcarchiveZip: os.Getenv("is_export_xcarchive_zip"),
		ExportAllDsyms:       os.Getenv("export_all_dsyms"),
		ArtifactName:         os.Getenv("artifact_name"),
	}
}

func (configs ConfigsModel) print() {
	log.Info("ipa export configs:")

	useCustomExportOptions := (configs.CustomExportOptionsPlistContent != "")
	if useCustomExportOptions {
		fmt.Println()
		log.Warn("Ignoring the following options because CustomExportOptionsPlistContent provided:")
	}

	log.Detail("- ExportMethod: %s", configs.ExportMethod)
	log.Detail("- UploadBitcode: %s", configs.UploadBitcode)
	log.Detail("- CompileBitcode: %s", configs.CompileBitcode)
	log.Detail("- TeamID: %s", configs.TeamID)

	if useCustomExportOptions {
		log.Warn("----------")
	}

	log.Detail("- UseDeprecatedExport: %s", configs.UseDeprecatedExport)
	log.Detail("- ForceTeamID: %s", configs.ForceTeamID)
	log.Detail("- ForceProvisioningProfileSpecifier: %s", configs.ForceProvisioningProfileSpecifier)
	log.Detail("- ForceProvisioningProfile: %s", configs.ForceProvisioningProfile)
	log.Detail("- ForceCodeSignIdentity: %s", configs.ForceCodeSignIdentity)
	log.Detail("- CustomExportOptionsPlistContent:")
	if configs.CustomExportOptionsPlistContent != "" {
		log.Detail(configs.CustomExportOptionsPlistContent)
	}
	fmt.Println()

	log.Info("xcodebuild configs:")
	log.Detail("- OutputTool: %s", configs.OutputTool)
	log.Detail("- Workdir: %s", configs.Workdir)
	log.Detail("- ProjectPath: %s", configs.ProjectPath)
	log.Detail("- Scheme: %s", configs.Scheme)
	log.Detail("- Configuration: %s", configs.Configuration)
	log.Detail("- OutputDir: %s", configs.OutputDir)
	log.Detail("- IsCleanBuild: %s", configs.IsCleanBuild)
	log.Detail("- XcodebuildOptions: %s", configs.XcodebuildOptions)
	fmt.Println()

	log.Info("step output configs:")
	log.Detail("- IsExportXcarchiveZip: %s", configs.IsExportXcarchiveZip)
	log.Detail("- ExportAllDsyms: %s", configs.ExportAllDsyms)
	log.Detail("- ArtifactName: %s", configs.ArtifactName)
	fmt.Println()
}

func (configs ConfigsModel) validate() error {
	if configs.ProjectPath == "" {
		return errors.New("no ProjectPath parameter specified")
	}
	if exist, err := pathutil.IsPathExists(configs.ProjectPath); err != nil {
		return fmt.Errorf("failed to check if ProjectPath exist at: %s, error: %s", configs.ProjectPath, err)
	} else if !exist {
		return fmt.Errorf("projectPath not exist at: %s", configs.ProjectPath)
	}

	if configs.Scheme == "" {
		return errors.New("no Scheme parameter specified")
	}

	if configs.OutputDir == "" {
		return errors.New("no OutputDir parameter specified")
	}

	if configs.OutputTool == "" {
		return errors.New("no OutputTool parameter specified")
	}
	if configs.OutputTool != "xcpretty" && configs.OutputTool != "xcodebuild" {
		return fmt.Errorf("invalid OutputTool specified (%s), valid options: [xcpretty xcodebuild]", configs.OutputTool)
	}

	if configs.IsCleanBuild == "" {
		return errors.New("no IsCleanBuild parameter specified")
	}
	if configs.IsCleanBuild != "yes" && configs.IsCleanBuild != "no" {
		return fmt.Errorf("invalid IsCleanBuild specified (%s), valid options: [yes no]", configs.IsCleanBuild)
	}

	if configs.IsExportXcarchiveZip == "" {
		return errors.New("no IsExportXcarchiveZip parameter specified")
	}
	if configs.IsExportXcarchiveZip != "yes" && configs.IsExportXcarchiveZip != "no" {
		return fmt.Errorf("invalid IsExportXcarchiveZip specified (%s), valid options: [yes no]", configs.IsExportXcarchiveZip)
	}

	if configs.UseDeprecatedExport == "" {
		return errors.New("no UseDeprecatedExport parameter specified")
	}
	if configs.UseDeprecatedExport != "yes" && configs.UseDeprecatedExport != "no" {
		return fmt.Errorf("invalid UseDeprecatedExport specified (%s), valid options: [yes no]", configs.UseDeprecatedExport)
	}

	if configs.ExportAllDsyms == "" {
		return errors.New("no ExportAllDsyms parameter specified")
	}
	if configs.ExportAllDsyms != "yes" && configs.ExportAllDsyms != "no" {
		return fmt.Errorf("invalid ExportAllDsyms specified (%s), valid options: [yes no]", configs.ExportAllDsyms)
	}

	return nil
}

func fail(format string, v ...interface{}) {
	log.Error(format, v...)
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

func applyRVMFix() error {
	if !utils.IsToolInstalled("rvm") {
		return nil
	}
	log.Warn(`Applying RVM 'fix'`)

	homeDir := pathutil.UserHomeDir()
	rvmScriptPth := filepath.Join(homeDir, ".rvm/scripts/rvm")
	if exist, err := pathutil.IsPathExists(rvmScriptPth); err != nil {
		return err
	} else if !exist {
		return nil
	}

	if err := cmdex.NewCommand("source", rvmScriptPth).Run(); err != nil {
		return err
	}

	if err := cmdex.NewCommand("rvm", "use", "system").Run(); err != nil {
		return err
	}

	return nil
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

func main() {
	configs := createConfigsModelFromEnvs()

	fmt.Println()
	configs.print()

	if err := configs.validate(); err != nil {
		fail("Issue with input: %s", err)
	}

	log.Info("step determined configs:")

	// Detect Xcode major version
	xcodebuildVersion, err := utils.XcodeBuildVersion()
	if err != nil {
		fail("Failed to determin xcode version, error: %s", err)
	}
	log.Detail("- xcodebuildVersion: %s (%s)", xcodebuildVersion.XcodeVersion.String(), xcodebuildVersion.BuildVersion)

	xcodeMajorVersion := xcodebuildVersion.XcodeVersion.Segments()[0]
	if xcodeMajorVersion < minSupportedXcodeMajorVersion {
		fail("Invalid xcode major version (%s), should not be less then min supported: %d", xcodeMajorVersion, minSupportedXcodeMajorVersion)
	}

	// Detect xcpretty version
	if configs.OutputTool == "xcpretty" {
		if !utils.IsXcprettyInstalled() {
			fail(`xcpretty is not installed
For xcpretty installation see: 'https://github.com/supermarin/xcpretty',
or use 'xcodebuild' as 'output_tool'.`)
		}

		xcprettyVersion, err := utils.XcprettyVersion()
		if err != nil {
			fail("Failed to determin xcpretty version, error: %s", err)
		}
		log.Detail("- xcprettyVersion: %s", xcprettyVersion.String())
	}

	// Validation CustomExportOptionsPlistContent
	if configs.CustomExportOptionsPlistContent != "" &&
		xcodeMajorVersion < 7 {
		log.Warn("CustomExportOptionsPlistContent is set, but CustomExportOptionsPlistContent only used if xcodeMajorVersion > 6")
		configs.CustomExportOptionsPlistContent = ""
	}

	if configs.ForceProvisioningProfileSpecifier != "" &&
		xcodeMajorVersion < 8 {
		log.Warn("ForceProvisioningProfileSpecifier is set, but ForceProvisioningProfileSpecifier only used if xcodeMajorVersion > 7")
		configs.ForceProvisioningProfileSpecifier = ""
	}

	if configs.ForceTeamID == "" &&
		xcodeMajorVersion < 8 {
		log.Warn("ForceTeamID is set, but ForceTeamID only used if xcodeMajorVersion > 7")
		configs.ForceTeamID = ""
	}

	if configs.ForceProvisioningProfileSpecifier != "" &&
		configs.ForceProvisioningProfile != "" {
		log.Warn("both ForceProvisioningProfileSpecifier and ForceProvisioningProfile are set, using ForceProvisioningProfileSpecifier")
		configs.ForceProvisioningProfile = ""
	}

	fmt.Println()

	// abs out dir pth
	absOutputDir, err := pathutil.AbsPath(configs.OutputDir)
	if err != nil {
		fail("Failed to expand OutputDir (%s), error: %s", configs.OutputDir, err)
	}
	configs.OutputDir = absOutputDir

	if exist, err := pathutil.IsPathExists(configs.OutputDir); err != nil {
		fail("Failed to check if OutputDir exist, error: %s", err)
	} else if !exist {
		if err := os.MkdirAll(configs.OutputDir, 0777); err != nil {
			fail("Failed to create OutputDir (%s), error: %s", configs.OutputDir, err)
		}
	}

	// output files
	tmpArchiveDir, err := pathutil.NormalizedOSTempDirPath("__archive__")
	if err != nil {
		fail("Failed to create temp dir for archives, error: %s", err)
	}
	tmpArchivePath := filepath.Join(tmpArchiveDir, configs.ArtifactName+".xcarchive")

	appPath := filepath.Join(configs.OutputDir, configs.ArtifactName+".app")
	ipaPath := filepath.Join(configs.OutputDir, configs.ArtifactName+".ipa")
	exportOptionsPath := filepath.Join(configs.OutputDir, "export_options.plist")
	rawXcodebuildOutputLogPath := filepath.Join(configs.OutputDir, "raw-xcodebuild-output.log")

	dsymZipPath := filepath.Join(configs.OutputDir, configs.ArtifactName+".dSYM.zip")
	archiveZipPath := filepath.Join(configs.OutputDir, configs.ArtifactName+".xcarchive.zip")
	ideDistributionLogsZipPath := filepath.Join(configs.OutputDir, "xcodebuild.xcdistributionlogs.zip")

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
			fail("Failed to check if path (%s) exist, error: %s", pth, err)
		} else if exist {
			if err := os.RemoveAll(pth); err != nil {
				fail("Failed to remove path (%s), error: %s", pth, err)
			}
		}
	}

	//
	// Create the Archive with Xcode Command Line tools
	log.Info("Create the Archive ...")
	fmt.Println()

	isWorkspace := false
	ext := filepath.Ext(configs.ProjectPath)
	if ext == ".xcodeproj" {
		isWorkspace = false
	} else if ext == ".xcworkspace" {
		isWorkspace = true
	} else {
		fail("Project file extension should be .xcodeproj or .xcworkspace, but got: %s", ext)
	}

	archiveCmd := xcodebuild.NewArchiveCommand(configs.ProjectPath, isWorkspace)
	archiveCmd.SetScheme(configs.Scheme)
	archiveCmd.SetConfiguration(configs.Configuration)

	if configs.ForceTeamID != "" {
		log.Detail("Forcing Development Team: %s", configs.ForceTeamID)
		archiveCmd.SetForceDevelopmentTeam(configs.ForceTeamID)
	}
	if configs.ForceProvisioningProfileSpecifier != "" {
		log.Detail("Forcing Provisioning Profile Specifier: %s", configs.ForceProvisioningProfileSpecifier)
		archiveCmd.SetForceProvisioningProfileSpecifier(configs.ForceProvisioningProfileSpecifier)
	}
	if configs.ForceProvisioningProfile != "" {
		log.Detail("Forcing Provisioning Profile: %s", configs.ForceProvisioningProfile)
		archiveCmd.SetForceProvisioningProfile(configs.ForceProvisioningProfile)
	}
	if configs.ForceCodeSignIdentity != "" {
		log.Detail("Forcing Code Signing Identity: %s", configs.ForceCodeSignIdentity)
		archiveCmd.SetForceCodeSignIdentity(configs.ForceCodeSignIdentity)
	}

	if configs.IsCleanBuild == "yes" {
		archiveCmd.SetCustomBuildAction("clean")
	}

	archiveCmd.SetArchivePath(tmpArchivePath)

	if configs.XcodebuildOptions != "" {
		options, err := shellquote.Split(configs.XcodebuildOptions)
		if err != nil {
			fail("Failed to shell split XcodebuildOptions (%s), error: %s", configs.XcodebuildOptions)
		}
		archiveCmd.SetCustomOptions(options)
	}

	if configs.OutputTool == "xcpretty" {
		xcprettyCmd := xcpretty.New(archiveCmd)

		logWithTimestamp(colorstring.Green, "$ %s", xcprettyCmd.PrintableCmd())
		fmt.Println()

		if rawXcodebuildOut, err := xcprettyCmd.Run(); err != nil {
			if err := utils.ExportOutputFileContent(rawXcodebuildOut, rawXcodebuildOutputLogPath, bitriseXcodeRawResultTextEnvKey); err != nil {
				log.Warn("Failed to export %s, error: %s", bitriseXcodeRawResultTextEnvKey, err)
			} else {
				log.Warn(`If you can't find the reason of the error in the log, please check the raw-xcodebuild-output.log
The log file is stored in $BITRISE_DEPLOY_DIR, and its full path
is available in the $BITRISE_XCODE_RAW_RESULT_TEXT_PATH environment variable`)
			}

			fail("Archive failed, error: %s", err)
		}
	} else {
		logWithTimestamp(colorstring.Green, "$ %s", archiveCmd.PrintableCmd())
		fmt.Println()

		if err := archiveCmd.Run(); err != nil {
			fail("Archive failed, error: %s", err)
		}
	}

	fmt.Println()

	// Ensure xcarchive exists
	if exist, err := pathutil.IsPathExists(tmpArchivePath); err != nil {
		fail("Failed to check if archive exist, error: %s", err)
	} else if !exist {
		fail("No archive generated at: %s", tmpArchivePath)
	}

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
	log.Info("Exporting ipa from the archive...")
	fmt.Println()

	envsToUnset := []string{"GEM_HOME", "GEM_PATH", "RUBYLIB", "RUBYOPT", "BUNDLE_BIN_PATH", "_ORIGINAL_GEM_PATH", "BUNDLE_GEMFILE"}
	for _, key := range envsToUnset {
		if err := os.Unsetenv(key); err != nil {
			fail("Failed to unset (%s), error: %s", key, err)
		}
	}

	if xcodeMajorVersion == 6 || configs.UseDeprecatedExport == "yes" {
		log.Detail("Using legacy export")
		/*
			Get the name of the profile which was used for creating the archive
			--> Search for embedded.mobileprovision in the xcarchive.
			It should contain a .app folder in the xcarchive folder
			under the Products/Applications folder
		*/

		embeddedProfilePth, err := xcarchive.EmbeddedMobileProvisionPth(tmpArchivePath)
		if err != nil {
			fail("Failed to get embedded profile path, error: %s", err)
		}

		provProfile, err := provisioningprofile.NewFromFile(embeddedProfilePth)
		if err != nil {
			fail("Failed to create provisioning profile model, error: %s", err)
		}

		if provProfile.Name == nil {
			fail("Profile name empty")
		}

		legacyExportCmd := xcodebuild.NewLegacyExportCommand()
		legacyExportCmd.SetExportFormat("ipa")
		legacyExportCmd.SetArchivePath(tmpArchivePath)
		legacyExportCmd.SetExportPath(ipaPath)
		legacyExportCmd.SetExportProvisioningProfileName(*provProfile.Name)

		if configs.OutputTool == "xcpretty" {
			xcprettyCmd := xcpretty.New(legacyExportCmd)

			logWithTimestamp(colorstring.Green, xcprettyCmd.PrintableCmd())
			fmt.Println()

			if rawXcodebuildOut, err := xcprettyCmd.Run(); err != nil {
				if err := utils.ExportOutputFileContent(rawXcodebuildOut, rawXcodebuildOutputLogPath, bitriseXcodeRawResultTextEnvKey); err != nil {
					log.Warn("Failed to export %s, error: %s", bitriseXcodeRawResultTextEnvKey, err)
				} else {
					log.Warn(`If you can't find the reason of the error in the log, please check the raw-xcodebuild-output.log
The log file is stored in $BITRISE_DEPLOY_DIR, and its full path
is available in the $BITRISE_XCODE_RAW_RESULT_TEXT_PATH environment variable`)
				}

				fail("Export failed, error: %s", err)
			}
		} else {
			logWithTimestamp(colorstring.Green, legacyExportCmd.PrintableCmd())
			fmt.Println()

			if err := legacyExportCmd.Run(); err != nil {
				fail("Export failed, error: %s", err)
			}
		}
	} else {
		log.Detail("Using export options")

		/*
		   Because of an RVM issue which conflicts with `xcodebuild`'s new
		   `-exportOptionsPlist` option
		   link: https://github.com/bitrise-io/steps-xcode-archive/issues/13
		*/
		if err := applyRVMFix(); err != nil {
			fail("rvm fix failed, error: %s", err)
		}

		if configs.CustomExportOptionsPlistContent != "" {
			log.Detail("Custom export options content provided:")
			fmt.Println(configs.CustomExportOptionsPlistContent)

			if err := fileutil.WriteStringToFile(exportOptionsPath, configs.CustomExportOptionsPlistContent); err != nil {
				fail("Failed to write export options to file, error: %s", err)
			}
		} else {
			log.Detail("Generating export options")

			var method exportoptions.Method
			if configs.ExportMethod == "auto-detect" {
				log.Detail("auto-detect export method, based on embedded profile")

				embeddedProfilePth, err := xcarchive.EmbeddedMobileProvisionPth(tmpArchivePath)
				if err != nil {
					fail("Failed to get embedded profile path, error: %s", err)
				}

				provProfile, err := provisioningprofile.NewFromFile(embeddedProfilePth)
				if err != nil {
					fail("Failed to create provisioning profile model, error: %s", err)
				}

				method = provProfile.GetExportMethod()
				log.Detail("detected export method: %s", method)
			} else {
				log.Detail("using export-method input: %s", configs.ExportMethod)
				parsedMethod, err := exportoptions.ParseMethod(configs.ExportMethod)
				if err != nil {
					fail("Failed to parse export options, error: %s", err)
				}
				method = parsedMethod
			}

			var exportOpts exportoptions.ExportOptions
			if method == exportoptions.MethodAppStore {
				options := exportoptions.NewAppStoreOptions()
				options.UploadBitcode = (configs.UploadBitcode == "yes")
				options.TeamID = configs.TeamID

				exportOpts = options
			} else {
				options := exportoptions.NewNonAppStoreOptions(method)
				options.CompileBitcode = (configs.CompileBitcode == "yes")
				options.TeamID = configs.TeamID

				exportOpts = options
			}

			log.Detail("generated export options content:")
			fmt.Println()
			fmt.Println(exportOpts.String())

			if err = exportOpts.WriteToFile(exportOptionsPath); err != nil {
				fail("Failed to write export options to file, error: %s", err)
			}
		}

		fmt.Println()

		tmpDir, err := pathutil.NormalizedOSTempDirPath("__export__")
		if err != nil {
			fail("Failed to create tmp dir, error: %s", err)
		}

		exportCmd := xcodebuild.NewExportCommand()
		exportCmd.SetArchivePath(tmpArchivePath)
		exportCmd.SetExportDir(tmpDir)
		exportCmd.SetExportOptionsPlist(exportOptionsPath)

		if configs.OutputTool == "xcpretty" {
			xcprettyCmd := xcpretty.New(exportCmd)

			logWithTimestamp(colorstring.Green, xcprettyCmd.PrintableCmd())
			fmt.Println()

			if xcodebuildOut, err := xcprettyCmd.Run(); err != nil {
				// xcodebuild raw output
				if err := utils.ExportOutputFileContent(xcodebuildOut, rawXcodebuildOutputLogPath, bitriseXcodeRawResultTextEnvKey); err != nil {
					log.Warn("Failed to export %s, error: %s", bitriseXcodeRawResultTextEnvKey, err)
				} else {
					log.Warn(`If you can't find the reason of the error in the log, please check the raw-xcodebuild-output.log
The log file is stored in $BITRISE_DEPLOY_DIR, and its full path
is available in the $BITRISE_XCODE_RAW_RESULT_TEXT_PATH environment variable`)
				}

				// xcdistributionlogs
				if logsDirPth, err := findIDEDistrubutionLogsPath(xcodebuildOut); err != nil {
					log.Warn("Failed to find xcdistributionlogs, error: %s", err)
				} else if err := utils.ExportOutputDirAsZip(logsDirPth, ideDistributionLogsZipPath, bitriseIDEDistributionLogsPthEnvKey); err != nil {
					log.Warn("Failed to export %s, error: %s", bitriseIDEDistributionLogsPthEnvKey, err)
				} else {
					log.Warn(`Also please check the xcdistributionlogs
The logs directory is stored in $BITRISE_DEPLOY_DIR, and its full path
is available in the $BITRISE_IDEDISTRIBUTION_LOGS_PATH environment variable`)
				}

				fail("Export failed, error: %s", err)
			}
		} else {
			logWithTimestamp(colorstring.Green, exportCmd.PrintableCmd())
			fmt.Println()

			if xcodebuildOut, err := exportCmd.RunAndReturnOutput(); err != nil {
				// xcdistributionlogs
				if logsDirPth, err := findIDEDistrubutionLogsPath(xcodebuildOut); err != nil {
					log.Warn("Failed to find xcdistributionlogs, error: %s", err)
				} else if err := utils.ExportOutputDirAsZip(logsDirPth, ideDistributionLogsZipPath, bitriseIDEDistributionLogsPthEnvKey); err != nil {
					log.Warn("Failed to export %s, error: %s", bitriseIDEDistributionLogsPthEnvKey, err)
				} else {
					log.Warn(`If you can't find the reason of the error in the log, please check the xcdistributionlogs
The logs directory is stored in $BITRISE_DEPLOY_DIR, and its full path
is available in the $BITRISE_IDEDISTRIBUTION_LOGS_PATH environment variable`)
				}

				fail("Export failed, error: %s", err)
			}
		}

		// Search for ipa
		pattern := filepath.Join(tmpDir, "*.ipa")
		ipas, err := filepath.Glob(pattern)
		if err != nil {
			fail("Failed to collect ipa files, error: %s", err)
		}

		if len(ipas) == 0 {
			fail("No ipa found with pattern: %s", pattern)
		} else if len(ipas) == 1 {
			if err := cmdex.CopyFile(ipas[0], ipaPath); err != nil {
				fail("Failed to copy (%s) -> (%s), error: %s", ipas[0], ipaPath, err)
			}
		} else {
			log.Warn("More than 1 .ipa file found")

			for _, ipa := range ipas {
				base := filepath.Base(ipa)
				deployPth := filepath.Join(configs.OutputDir, base)

				if err := cmdex.CopyFile(ipa, deployPth); err != nil {
					fail("Failed to copy (%s) -> (%s), error: %s", ipas[0], ipaPath, err)
				}
				ipaPath = ipa
			}
		}
	}

	log.Info("Exporting outputs...")

	//
	// Export outputs

	// Export .xcarchive
	fmt.Println()

	if err := utils.ExportOutputDir(tmpArchivePath, tmpArchivePath, bitriseXCArchivePthEnvKey); err != nil {
		fail("Failed to export %s, error: %s", bitriseXCArchivePthEnvKey, err)
	}

	log.Done("The xcarchive path is now available in the Environment Variable: $BITRISE_XCARCHIVE_PATH (value: %s)", tmpArchivePath)

	if configs.IsExportXcarchiveZip == "yes" {
		if err := utils.ExportOutputDirAsZip(tmpArchivePath, archiveZipPath, bitriseXCArchiveZipPthEnvKey); err != nil {
			fail("Failed to export %s, error: %s", bitriseXCArchiveZipPthEnvKey, err)
		}

		log.Done("The xcarchive zip path is now available in the Environment Variable: $BITRISE_XCARCHIVE_ZIP_PATH (value: %s)", archiveZipPath)
	}

	// Export .app
	fmt.Println()

	exportedApp, err := xcarchive.FindApp(tmpArchivePath)
	if err != nil {
		fail("Failed to find app, error: %s", err)
	}

	if err := utils.ExportOutputDir(exportedApp, exportedApp, bitriseAppDirPthEnvKey); err != nil {
		fail("Failed to export %s, error: %s", bitriseAppDirPthEnvKey, err)
	}

	log.Done("The app directory is now available in the Environment Variable: $BITRISE_APP_DIR_PATH (value: %s)", appPath)

	// Export .ipa
	fmt.Println()

	if err := utils.ExportOutputFile(ipaPath, ipaPath, bitriseIPAPthEnvKey); err != nil {
		fail("Failed to export %s, error: %s", bitriseIPAPthEnvKey, err)
	}

	log.Done("The ipa path is now available in the Environment Variable: $BITRISE_IPA_PATH (value: %s)", ipaPath)

	// Export .dSYMs
	fmt.Println()

	appDSYM, frameworkDSYMs, err := xcarchive.FindDSYMs(tmpArchivePath)
	if err != nil {
		fail("Failed to export dsyms, error: %s", err)
	}

	dsymDir, err := pathutil.NormalizedOSTempDirPath("__dsyms__")
	if err != nil {
		fail("Failed to create tmp dir, error: %s", err)
	}

	if err := cmdex.CopyDir(appDSYM, dsymDir, false); err != nil {
		fail("Failed to copy (%s) -> (%s), error: %s", appDSYM, dsymDir, err)
	}

	if configs.ExportAllDsyms == "yes" {
		for _, dsym := range frameworkDSYMs {
			if err := cmdex.CopyDir(dsym, dsymDir, false); err != nil {
				fail("Failed to copy (%s) -> (%s), error: %s", dsym, dsymDir, err)
			}
		}
	}

	if err := utils.ExportOutputDir(dsymDir, dsymDir, bitriseDSYMDirPthEnvKey); err != nil {
		fail("Failed to export %s, error: %s", bitriseDSYMDirPthEnvKey, err)
	}

	log.Done("The dSYM dir path is now available in the Environment Variable: $BITRISE_DSYM_DIR_PATH (value: %s)", dsymDir)

	if err := utils.ExportOutputDirAsZip(dsymDir, dsymZipPath, bitriseDSYMPthEnvKey); err != nil {
		fail("Failed to export %s, error: %s", bitriseDSYMPthEnvKey, err)
	}

	log.Done("The dSYM zip path is now available in the Environment Variable: $BITRISE_DSYM_PATH (value: %s)", dsymZipPath)
}
