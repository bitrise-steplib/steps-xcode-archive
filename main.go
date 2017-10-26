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

	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/steps-xcode-archive/utils"
	"github.com/bitrise-tools/go-xcode/certificateutil"
	"github.com/bitrise-tools/go-xcode/exportoptions"
	"github.com/bitrise-tools/go-xcode/plistutil"
	"github.com/bitrise-tools/go-xcode/profileutil"
	"github.com/bitrise-tools/go-xcode/xcarchive"
	"github.com/bitrise-tools/go-xcode/xcodebuild"
	"github.com/bitrise-tools/go-xcode/xcodeproj"
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
	log.Infof("ipa export configs:")
	log.Printf("- ExportMethod: %s", configs.ExportMethod)
	log.Printf("- UploadBitcode: %s", configs.UploadBitcode)
	log.Printf("- CompileBitcode: %s", configs.CompileBitcode)
	log.Printf("- TeamID: %s", configs.TeamID)
	log.Printf("- UseDeprecatedExport: %s", configs.UseDeprecatedExport)
	log.Printf("- CustomExportOptionsPlistContent:")
	if configs.CustomExportOptionsPlistContent != "" {
		log.Printf(configs.CustomExportOptionsPlistContent)
	}
	fmt.Println()

	log.Infof("xcodebuild configs:")
	log.Printf("- OutputTool: %s", configs.OutputTool)
	log.Printf("- Workdir: %s", configs.Workdir)
	log.Printf("- ProjectPath: %s", configs.ProjectPath)
	log.Printf("- Scheme: %s", configs.Scheme)
	log.Printf("- Configuration: %s", configs.Configuration)
	log.Printf("- OutputDir: %s", configs.OutputDir)
	log.Printf("- IsCleanBuild: %s", configs.IsCleanBuild)
	log.Printf("- XcodebuildOptions: %s", configs.XcodebuildOptions)
	log.Printf("- ForceTeamID: %s", configs.ForceTeamID)
	log.Printf("- ForceProvisioningProfileSpecifier: %s", configs.ForceProvisioningProfileSpecifier)
	log.Printf("- ForceProvisioningProfile: %s", configs.ForceProvisioningProfile)
	log.Printf("- ForceCodeSignIdentity: %s", configs.ForceCodeSignIdentity)
	fmt.Println()

	log.Infof("step output configs:")
	log.Printf("- IsExportXcarchiveZip: %s", configs.IsExportXcarchiveZip)
	log.Printf("- ExportAllDsyms: %s", configs.ExportAllDsyms)
	log.Printf("- ArtifactName: %s", configs.ArtifactName)
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

func printCertificateInfo(info certificateutil.CertificateInfoModel) {
	log.Printf(info.CommonName)
	log.Printf("serial: %s", info.Serial)
	log.Printf("team: %s (%s)", info.TeamName, info.TeamID)
	log.Printf("expire: %s", info.EndDate)

	if err := info.CheckValidity(); err != nil {
		log.Errorf("[X] %s", err)
	}
}

func printProfileInfo(info profileutil.ProvisioningProfileInfoModel, installedCertificates []certificateutil.CertificateInfoModel) {
	log.Printf("%s (%s)", info.Name, info.UUID)
	log.Printf("exportType: %s", string(info.ExportType))
	log.Printf("team: %s (%s)", info.TeamName, info.TeamID)
	log.Printf("bundleID: %s", info.BundleID)

	log.Printf("certificates:")
	for _, certificateInfo := range info.DeveloperCertificates {
		log.Printf("- %s", certificateInfo.CommonName)
		log.Printf("  serial: %s", certificateInfo.Serial)
		log.Printf("  teamID: %s", certificateInfo.TeamID)
	}

	log.Printf("devices:")
	for _, deviceID := range info.ProvisionedDevices {
		log.Printf("- %s", deviceID)
	}

	log.Printf("expire: %s", info.ExpirationDate)

	if !info.HasInstalledCertificate(installedCertificates) {
		log.Errorf("[X] none of the profile's certificates are installed")
	}

	if err := info.CheckValidity(); err != nil {
		log.Errorf("[X] %s", err)
	}

	if info.IsXcodeManaged() {
		log.Warnf("[!] xcode managed profile")
	}
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

func main() {
	configs := createConfigsModelFromEnvs()

	fmt.Println()
	configs.print()

	if err := configs.validate(); err != nil {
		fail("Issue with input: %s", err)
	}

	log.Infof("step determined configs:")

	// Detect Xcode major version
	xcodebuildVersion, err := utils.XcodeBuildVersion()
	if err != nil {
		fail("Failed to determin xcode version, error: %s", err)
	}
	log.Printf("- xcodebuildVersion: %s (%s)", xcodebuildVersion.XcodeVersion.String(), xcodebuildVersion.BuildVersion)

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
		log.Printf("- xcprettyVersion: %s", xcprettyVersion.String())
	}

	// Validation CustomExportOptionsPlistContent
	customExportOptionsPlistContent := strings.TrimSpace(configs.CustomExportOptionsPlistContent)
	if customExportOptionsPlistContent != configs.CustomExportOptionsPlistContent {
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
			log.Printf("- ExportMethod: %s", configs.ExportMethod)
			log.Printf("- UploadBitcode: %s", configs.UploadBitcode)
			log.Printf("- CompileBitcode: %s", configs.CompileBitcode)
			log.Printf("- TeamID: %s", configs.TeamID)
			fmt.Println()
		}
	}

	if configs.ForceProvisioningProfileSpecifier != "" &&
		xcodeMajorVersion < 8 {
		fmt.Println()
		log.Warnf("ForceProvisioningProfileSpecifier is set, but ForceProvisioningProfileSpecifier only used if xcodeMajorVersion > 7")
		configs.ForceProvisioningProfileSpecifier = ""
	}

	if configs.ForceTeamID != "" &&
		xcodeMajorVersion < 8 {
		fmt.Println()
		log.Warnf("ForceTeamID is set, but ForceTeamID only used if xcodeMajorVersion > 7")
		configs.ForceTeamID = ""
	}

	if configs.ForceProvisioningProfileSpecifier != "" &&
		configs.ForceProvisioningProfile != "" {
		fmt.Println()
		log.Warnf("both ForceProvisioningProfileSpecifier and ForceProvisioningProfile are set, using ForceProvisioningProfileSpecifier")
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
	log.Infof("Create the Archive ...")
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
		log.Printf("Forcing Development Team: %s", configs.ForceTeamID)
		archiveCmd.SetForceDevelopmentTeam(configs.ForceTeamID)
	}
	if configs.ForceProvisioningProfileSpecifier != "" {
		log.Printf("Forcing Provisioning Profile Specifier: %s", configs.ForceProvisioningProfileSpecifier)
		archiveCmd.SetForceProvisioningProfileSpecifier(configs.ForceProvisioningProfileSpecifier)
	}
	if configs.ForceProvisioningProfile != "" {
		log.Printf("Forcing Provisioning Profile: %s", configs.ForceProvisioningProfile)
		archiveCmd.SetForceProvisioningProfile(configs.ForceProvisioningProfile)
	}
	if configs.ForceCodeSignIdentity != "" {
		log.Printf("Forcing Code Signing Identity: %s", configs.ForceCodeSignIdentity)
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
				log.Warnf("Failed to export %s, error: %s", bitriseXcodeRawResultTextEnvKey, err)
			} else {
				log.Warnf(`If you can't find the reason of the error in the log, please check the raw-xcodebuild-output.log
The log file is stored in $BITRISE_DEPLOY_DIR, and its full path
is available in the $BITRISE_XCODE_RAW_RESULT_TEXT_PATH environment variable`)
			}

			fail("Archive failed, error: %s", err)
		}
	} else {
		logWithTimestamp(colorstring.Green, "$ %s", archiveCmd.PrintableCmd())
		fmt.Println()

		archiveRootCmd := archiveCmd.Command()
		archiveRootCmd.SetStdout(os.Stdout)
		archiveRootCmd.SetStderr(os.Stderr)

		if err := archiveRootCmd.Run(); err != nil {
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

	archiveTeamID := ""
	archiveProfileName := ""
	var archiveExportMethod exportoptions.Method
	archiveCodeSignIsXcodeManaged := false
	{
		embeddedProfilePth, err := xcarchive.FindEmbeddedMobileProvision(tmpArchivePath)
		if err != nil {
			fail("Failed to get embedded profile path, error: %s", err)
		}

		profile, err := profileutil.NewPlistDataFromFile(embeddedProfilePth)
		if err != nil {
			fail("Failed to create provisioning profile model, error: %s", err)
		}

		archiveTeamID = profile.GetTeamID()
		archiveProfileName = profile.GetName()
		archiveExportMethod = profile.GetExportMethod()
		archiveCodeSignIsXcodeManaged = profileutil.IsXcodeManaged(profile.GetName())
	}

	log.Infof("Archive infos:")
	log.Printf("archiveTeamID: %s", archiveTeamID)
	log.Printf("archiveProfileName: %s", archiveProfileName)
	log.Printf("archiveExportMethod: %s", archiveExportMethod)
	log.Printf("archiveCodeSignIsXcodeManaged: %v", archiveCodeSignIsXcodeManaged)
	fmt.Println()

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

	envsToUnset := []string{"GEM_HOME", "GEM_PATH", "RUBYLIB", "RUBYOPT", "BUNDLE_BIN_PATH", "_ORIGINAL_GEM_PATH", "BUNDLE_GEMFILE"}
	for _, key := range envsToUnset {
		if err := os.Unsetenv(key); err != nil {
			fail("Failed to unset (%s), error: %s", key, err)
		}
	}

	if xcodeMajorVersion == 6 || configs.UseDeprecatedExport == "yes" {
		log.Printf("Using legacy export")
		/*
			Get the name of the profile which was used for creating the archive
			--> Search for embedded.mobileprovision in the xcarchive.
			It should contain a .app folder in the xcarchive folder
			under the Products/Applications folder
		*/

		embeddedProfilePth, err := xcarchive.FindEmbeddedMobileProvision(tmpArchivePath)
		if err != nil {
			fail("Failed to get embedded profile path, error: %s", err)
		}

		profile, err := profileutil.NewPlistDataFromFile(embeddedProfilePth)
		if err != nil {
			fail("Failed to create provisioning profile model, error: %s", err)
		}

		name := profile.GetName()

		legacyExportCmd := xcodebuild.NewLegacyExportCommand()
		legacyExportCmd.SetExportFormat("ipa")
		legacyExportCmd.SetArchivePath(tmpArchivePath)
		legacyExportCmd.SetExportPath(ipaPath)
		legacyExportCmd.SetExportProvisioningProfileName(name)

		if configs.OutputTool == "xcpretty" {
			xcprettyCmd := xcpretty.New(legacyExportCmd)

			logWithTimestamp(colorstring.Green, xcprettyCmd.PrintableCmd())
			fmt.Println()

			if rawXcodebuildOut, err := xcprettyCmd.Run(); err != nil {
				if err := utils.ExportOutputFileContent(rawXcodebuildOut, rawXcodebuildOutputLogPath, bitriseXcodeRawResultTextEnvKey); err != nil {
					log.Warnf("Failed to export %s, error: %s", bitriseXcodeRawResultTextEnvKey, err)
				} else {
					log.Warnf(`If you can't find the reason of the error in the log, please check the raw-xcodebuild-output.log
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
		log.Printf("Exporting ipa with ExportOptions.plist")

		if customExportOptionsPlistContent != "" {
			log.Printf("Custom export options content provided, using it:")
			fmt.Println(customExportOptionsPlistContent)

			if err := fileutil.WriteStringToFile(exportOptionsPath, customExportOptionsPlistContent); err != nil {
				fail("Failed to write export options to file, error: %s", err)
			}
		} else {
			log.Printf("No custom export options content provided, generating export options...")

			var exportMethod exportoptions.Method
			exportTeamID := ""
			exportCodeSignIdentity := ""
			exportCodeSignStyle := ""
			exportProfileMapping := map[string]string{}

			if configs.ExportMethod == "auto-detect" {
				log.Printf("auto-detect export method specified")
				exportMethod = archiveExportMethod

				log.Printf("using the archive profile's (%s) export method: %s", archiveProfileName, exportMethod)
			} else {
				parsedMethod, err := exportoptions.ParseMethod(configs.ExportMethod)
				if err != nil {
					fail("Failed to parse export options, error: %s", err)
				}
				exportMethod = parsedMethod
				log.Printf("export-method specified: %s", configs.ExportMethod)
			}

			if xcodeMajorVersion >= 9 {
				log.Printf("xcode major version > 9, generating provisioningProfiles node")

				user := os.Getenv("USER")
				targetCodeSignInfoMap, err := xcodeproj.ResolveCodeSignInfo(configs.ProjectPath, configs.Scheme, user)
				if err != nil {
					log.Errorf("Failed to create scheme - target mapping, error: %s", err)
					log.Errorf("Please contact us on bitrise on-site-chat and")
					log.Errorf("attach your failed build url and your project file(s) (.xcworkspace and/or .xcodeproj file(s)),")
					log.Errorf("to let us identify the issue.")
					os.Exit(1)
				}

				bundleIDs := []string{}
				bundleIDTargetMap := map[string]string{}
				bundleIDEntitlemnstMap := map[string]plistutil.PlistData{}

				fmt.Println()
				log.Printf("Target - CodeSignInfo mapping:")
				for target, info := range targetCodeSignInfoMap {
					log.Printf(target)
					log.Printf("  BundleIdentifier: %s", info.BundleIdentifier)
					log.Printf("  DevelopmentTeam: %s", info.DevelopmentTeam)
					log.Printf("  CodeSignIdentity: %s", info.CodeSignIdentity)

					profile := info.ProvisioningProfileSpecifier
					if profile == "" {
						profile = info.ProvisioningProfile
					}
					log.Printf("  Profile: %s", profile)

					bundleIDs = append(bundleIDs, info.BundleIdentifier)
					bundleIDTargetMap[info.BundleIdentifier] = target

					if info.CodeSignEntitlementsPath == "" {
						continue
					}

					if exist, err := pathutil.IsPathExists(info.CodeSignEntitlementsPath); err != nil {
						log.Warnf("Failed to check if target (%s) entitlements file exist, error: %s", target, err)
						continue
					} else if !exist {
						log.Warnf("Target (%s) entitlements file not exist at: %s", target, info.CodeSignEntitlementsPath)
						continue
					}

					entitlements, err := plistutil.NewPlistDataFromFile(info.CodeSignEntitlementsPath)
					if err != nil {
						log.Warnf("Failed to parse target (%s) entitlements", target)
						continue
					}

					bundleIDEntitlemnstMap[info.BundleIdentifier] = entitlements

					log.Printf("  Entitlements:")
					for key := range entitlements {
						log.Printf("  - %s", key)
					}
				}
				fmt.Println()

				certs, err := certificateutil.InstalledCodesigningCertificateInfos()
				if err != nil {
					fail("Failed to get installed certificates, error: %s", err)
				}

				log.Printf("Installed certificates:")
				for _, certInfo := range certs {
					printCertificateInfo(certInfo)
					fmt.Println()
				}

				profs, err := profileutil.InstalledProvisioningProfileInfos(profileutil.ProfileTypeIos)
				if err != nil {
					fail("Failed to get installed provisioning profiles, error: %s", err)
				}

				log.Printf("Installed profiles:")
				for _, profileInfo := range profs {
					printProfileInfo(profileInfo, certs)
					fmt.Println()
				}

				log.Printf("Resolving CodeSignGroups...")
				codeSignGroups := utils.ResolveCodeSignGroupItems(bundleIDs, exportoptions.Method(exportMethod), profs, certs)
				if err != nil {
					log.Errorf("Failed to get matching provisioning profiles, error: %s", err)
				}

				log.Printf("Resolved CodeSignGroups:")
				for _, group := range codeSignGroups {
					log.Printf("codeSignIdentity: %s", group.Certificate.CommonName)
					for bundleID, prof := range group.BundleIDProfileMap {
						log.Printf("bundle ID: %s is provisioned by: %s", bundleID, prof.Name)
					}
					fmt.Println()
				}

				if len(codeSignGroups) == 0 {
					log.Errorf("Failed to find code singing groups for specified export method (%s)", exportMethod)
				}

				// Handle if archive used NON xcode managed profile
				if len(codeSignGroups) > 0 && !archiveCodeSignIsXcodeManaged {
					log.Warnf("App was signed with NON xcode managed profile when archiving,")
					log.Warnf("only NOT xcode managed profiles are allowed to sign when exporting the archive.")
					log.Warnf("Removing xcode managed CodeSignInfo groups")

					filteredGroups := []utils.CodeSignGroupItem{}
					for _, group := range codeSignGroups {
						xcodeManagedGroup := false
						for _, profile := range group.BundleIDProfileMap {
							isXcodeManaged := profileutil.IsXcodeManaged(profile.Name)
							if isXcodeManaged {
								xcodeManagedGroup = true
								break
							}
						}
						if !xcodeManagedGroup {
							filteredGroups = append(filteredGroups, group)
						} else {
							log.Warnf("removing CodeSignGroup: %s", group.Certificate.CommonName)
						}
					}

					codeSignGroups = filteredGroups

					if len(codeSignGroups) == 0 {
						log.Errorf("Failed to find code singing groups for specified export method (%s) and WITHOUT xcode managed profiles", exportMethod)
					}
				}

				// Filter for specified export team
				if len(codeSignGroups) > 0 && configs.TeamID != "" {
					log.Warnf("Export TeamID specified: %s, filtering CodeSignInfo groups...", configs.TeamID)

					filteredGroups := []utils.CodeSignGroupItem{}
					for _, group := range codeSignGroups {
						if group.Certificate.TeamID == configs.TeamID {
							filteredGroups = append(filteredGroups, group)
						} else {
							log.Warnf("removing CodeSignGroup: %s", group.Certificate.CommonName)
							fmt.Println()
						}
					}

					codeSignGroups = filteredGroups

					if len(codeSignGroups) == 0 {
						log.Errorf("Failed to find code singing groups for specified export method (%s) and team (%s)", exportMethod, configs.TeamID)
					}
				}

				// Filter for capabilities
				if len(codeSignGroups) > 0 && len(bundleIDEntitlemnstMap) > 0 {
					log.Warnf("Filtering CodeSignInfo groups for target capabilities")

					filteredGroups := []utils.CodeSignGroupItem{}
					for _, group := range codeSignGroups {
						validGroup := true
						for bundleID, profile := range group.BundleIDProfileMap {
							targetEntitlements := bundleIDEntitlemnstMap[bundleID]
							profileEntitlements := profile.Entitlements

							missingEntitlements := profileutil.MatchTargetAndProfileEntitlements(targetEntitlements, profileEntitlements)
							if len(missingEntitlements) > 0 {
								target := bundleIDTargetMap[bundleID]
								log.Printf("Entitlements missing from profile (%s) for target (%s)", profile.Name, target)
								for _, entitlemntKey := range missingEntitlements {
									log.Printf("- %s", entitlemntKey)
								}

								validGroup = false
								continue
							}
						}

						if validGroup {
							filteredGroups = append(filteredGroups, group)
						} else {
							log.Warnf("removing CodeSignGroup: %s", group.Certificate.CommonName)
							fmt.Println()
						}
					}

					codeSignGroups = filteredGroups

					if len(codeSignGroups) == 0 {
						log.Errorf("Failed to find code singing groups for specified export method (%s) and project capabilities", exportMethod)
					}
				}

				// Filter out default code sign files
				if len(codeSignGroups) > 0 && configs.TeamID == "" {
					if defaultProfile, err := utils.GetDefaultProvisioningProfile(); err == nil && defaultProfile.TeamID != "" {
						if exportTeamID != defaultProfile.TeamID {
							filteredGroups := []utils.CodeSignGroupItem{}
							for _, group := range codeSignGroups {
								if group.Certificate.TeamID != defaultProfile.TeamID {
									filteredGroups = append(filteredGroups, group)
								}
							}

							if len(filteredGroups) > 0 {
								codeSignGroups = filteredGroups
							}
						}
					}
				}

				fmt.Println()
				log.Printf("Filtered CodeSignGroups:")
				for _, group := range codeSignGroups {
					log.Printf("codeSignIdentity: %s", group.Certificate.CommonName)
					for bundleID, prof := range group.BundleIDProfileMap {
						log.Printf("bundle ID: %s is provisioned by: %s", bundleID, prof.Name)
					}
					fmt.Println()
				}

				if len(codeSignGroups) > 0 {
					codeSignGroup := utils.CodeSignGroupItem{}

					if len(codeSignGroups) == 1 {
						codeSignGroup = codeSignGroups[0]
					} else if len(codeSignGroups) > 1 {
						log.Warnf("Multiple code singing groups found")

						codeSignGroup = codeSignGroups[0]

						found := false
						if archiveTeamID != "" {
							for i, group := range codeSignGroups {
								if group.Certificate.TeamID == exportTeamID && i != 0 {
									log.Warnf("Prefering code singing group with the team used for the archive")
									codeSignGroup = group
									found = true
									break
								}
							}
						}
						if !found {
							log.Warnf("Using first group")
						}
					}

					exportTeamID = codeSignGroup.Certificate.TeamID
					exportCodeSignIdentity = codeSignGroup.Certificate.CommonName

					for bundleID, profileInfo := range codeSignGroup.BundleIDProfileMap {
						exportProfileMapping[bundleID] = profileInfo.Name

						isXcodeManaged := profileutil.IsXcodeManaged(profileInfo.Name)
						if isXcodeManaged {
							if exportCodeSignStyle != "" && exportCodeSignStyle != "automatic" {
								log.Errorf("Both xcode managed and NON xcode managed profiles in code singing group")
							}
							exportCodeSignStyle = "automatic"
						} else {
							if exportCodeSignStyle != "" && exportCodeSignStyle != "manual" {
								log.Errorf("Both xcode managed and NON xcode managed profiles in code singing group")
							}
							exportCodeSignStyle = "manual"
						}
					}
				}
			}

			var exportOpts exportoptions.ExportOptions
			if exportMethod == exportoptions.MethodAppStore {
				options := exportoptions.NewAppStoreOptions()
				options.UploadBitcode = (configs.UploadBitcode == "yes")

				if xcodeMajorVersion >= 9 {
					options.BundleIDProvisioningProfileMapping = exportProfileMapping
					options.SigningCertificate = exportCodeSignIdentity
					options.TeamID = exportTeamID

					if archiveCodeSignIsXcodeManaged && exportCodeSignStyle == "manual" {
						log.Warnf("App was signed with xcode managed profile when archiving,")
						log.Warnf("ipa export uses manual code singing.")
						log.Warnf(`Setting "signingStyle" to "manual"`)

						options.SigningStyle = "manual"
					}
				}

				exportOpts = options
			} else {
				options := exportoptions.NewNonAppStoreOptions(exportMethod)
				options.CompileBitcode = (configs.CompileBitcode == "yes")

				if xcodeMajorVersion >= 9 {
					options.BundleIDProvisioningProfileMapping = exportProfileMapping
					options.SigningCertificate = exportCodeSignIdentity
					options.TeamID = exportTeamID

					if archiveCodeSignIsXcodeManaged && exportCodeSignStyle == "manual" {
						log.Warnf("App was signed with xcode managed profile when archiving,")
						log.Warnf("ipa export uses manual code singing.")
						log.Warnf(`Setting "signingStyle" to "manual"`)

						options.SigningStyle = "manual"
					}
				}

				exportOpts = options
			}

			fmt.Println()
			log.Printf("generated export options content:")
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
					log.Warnf("Failed to export %s, error: %s", bitriseXcodeRawResultTextEnvKey, err)
				} else {
					log.Warnf(`If you can't find the reason of the error in the log, please check the raw-xcodebuild-output.log
The log file is stored in $BITRISE_DEPLOY_DIR, and its full path
is available in the $BITRISE_XCODE_RAW_RESULT_TEXT_PATH environment variable`)
				}

				// xcdistributionlogs
				if logsDirPth, err := findIDEDistrubutionLogsPath(xcodebuildOut); err != nil {
					log.Warnf("Failed to find xcdistributionlogs, error: %s", err)
				} else if err := utils.ExportOutputDirAsZip(logsDirPth, ideDistributionLogsZipPath, bitriseIDEDistributionLogsPthEnvKey); err != nil {
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

				fail("Export failed, error: %s", err)
			}
		} else {
			logWithTimestamp(colorstring.Green, exportCmd.PrintableCmd())
			fmt.Println()

			if xcodebuildOut, err := exportCmd.RunAndReturnOutput(); err != nil {
				// xcdistributionlogs
				if logsDirPth, err := findIDEDistrubutionLogsPath(xcodebuildOut); err != nil {
					log.Warnf("Failed to find xcdistributionlogs, error: %s", err)
				} else if err := utils.ExportOutputDirAsZip(logsDirPth, ideDistributionLogsZipPath, bitriseIDEDistributionLogsPthEnvKey); err != nil {
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

				fail("Export failed, error: %s", err)
			}
		}

		// Search for ipa
		fileList := []string{}
		ipaFiles := []string{}
		if walkErr := filepath.Walk(tmpDir, func(pth string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			fileList = append(fileList, pth)

			if filepath.Ext(pth) == ".ipa" {
				ipaFiles = append(ipaFiles, pth)
			}

			return nil
		}); walkErr != nil {
			fail("Failed to search for .ipa file, error: %s", err)
		}

		if len(ipaFiles) == 0 {
			log.Errorf("No .ipa file found at export dir: %s", tmpDir)
			log.Printf("File list in the export dir:")
			for _, pth := range fileList {
				log.Printf("- %s", pth)
			}
			fail("")
		} else {
			if err := command.CopyFile(ipaFiles[0], ipaPath); err != nil {
				fail("Failed to copy (%s) -> (%s), error: %s", ipaFiles[0], ipaPath, err)
			}

			if len(ipaFiles) > 1 {
				log.Warnf("More than 1 .ipa file found, exporting first one: %s", ipaFiles[0])
				log.Warnf("Moving every ipa to the BITRISE_DEPLOY_DIR")

				for i, pth := range ipaFiles {
					if i == 0 {
						continue
					}

					base := filepath.Base(pth)
					deployPth := filepath.Join(configs.OutputDir, base)

					if err := command.CopyFile(pth, deployPth); err != nil {
						fail("Failed to copy (%s) -> (%s), error: %s", pth, ipaPath, err)
					}
				}
			}
		}
	}

	log.Infof("Exporting outputs...")

	//
	// Export outputs

	// Export .xcarchive
	fmt.Println()

	if err := utils.ExportOutputDir(tmpArchivePath, tmpArchivePath, bitriseXCArchivePthEnvKey); err != nil {
		fail("Failed to export %s, error: %s", bitriseXCArchivePthEnvKey, err)
	}

	log.Donef("The xcarchive path is now available in the Environment Variable: %s (value: %s)", bitriseXCArchivePthEnvKey, tmpArchivePath)

	if configs.IsExportXcarchiveZip == "yes" {
		if err := utils.ExportOutputDirAsZip(tmpArchivePath, archiveZipPath, bitriseXCArchiveZipPthEnvKey); err != nil {
			fail("Failed to export %s, error: %s", bitriseXCArchiveZipPthEnvKey, err)
		}

		log.Donef("The xcarchive zip path is now available in the Environment Variable: %s (value: %s)", bitriseXCArchiveZipPthEnvKey, archiveZipPath)
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

	log.Donef("The app directory is now available in the Environment Variable: %s (value: %s)", bitriseAppDirPthEnvKey, appPath)

	// Export .ipa
	fmt.Println()

	if err := utils.ExportOutputFile(ipaPath, ipaPath, bitriseIPAPthEnvKey); err != nil {
		fail("Failed to export %s, error: %s", bitriseIPAPthEnvKey, err)
	}

	log.Donef("The ipa path is now available in the Environment Variable: %s (value: %s)", bitriseIPAPthEnvKey, ipaPath)

	// Export .dSYMs
	fmt.Println()

	appDSYM, frameworkDSYMs, err := xcarchive.FindDSYMs(tmpArchivePath)
	if err != nil {
		if err.Error() == "no dsym found" {
			log.Warnf("no app nor framework dsyms found")
		} else {
			fail("Failed to export dsyms, error: %s", err)
		}
	}
	if err == nil {
		dsymDir, err := pathutil.NormalizedOSTempDirPath("__dsyms__")
		if err != nil {
			fail("Failed to create tmp dir, error: %s", err)
		}

		if err := command.CopyDir(appDSYM, dsymDir, false); err != nil {
			fail("Failed to copy (%s) -> (%s), error: %s", appDSYM, dsymDir, err)
		}

		if configs.ExportAllDsyms == "yes" {
			for _, dsym := range frameworkDSYMs {
				if err := command.CopyDir(dsym, dsymDir, false); err != nil {
					fail("Failed to copy (%s) -> (%s), error: %s", dsym, dsymDir, err)
				}
			}
		}

		if err := utils.ExportOutputDir(dsymDir, dsymDir, bitriseDSYMDirPthEnvKey); err != nil {
			fail("Failed to export %s, error: %s", bitriseDSYMDirPthEnvKey, err)
		}

		log.Donef("The dSYM dir path is now available in the Environment Variable: %s (value: %s)", bitriseDSYMDirPthEnvKey, dsymDir)

		if err := utils.ExportOutputDirAsZip(dsymDir, dsymZipPath, bitriseDSYMPthEnvKey); err != nil {
			fail("Failed to export %s, error: %s", bitriseDSYMPthEnvKey, err)
		}

		log.Donef("The dSYM zip path is now available in the Environment Variable: %s (value: %s)", bitriseDSYMPthEnvKey, dsymZipPath)
	}
}
