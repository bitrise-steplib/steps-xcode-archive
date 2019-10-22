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
	"github.com/bitrise-io/go-xcode/certificateutil"
	"github.com/bitrise-io/go-xcode/export"
	"github.com/bitrise-io/go-xcode/exportoptions"
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

// configs ...
type configs struct {
	ExportMethod               string `env:"export_method,opt[auto-detect,app-store,ad-hoc,enterprise,development]"`
	UploadBitcode              bool   `env:"upload_bitcode,opt[yes,no]"`
	CompileBitcode             bool   `env:"compile_bitcode,opt[yes,no]"`
	ICloudContainerEnvironment string `env:"icloud_container_environment"`
	TeamID                     string `env:"team_id"`

	UseDeprecatedExport               bool   `env:"use_deprecated_export,opt[yes,no]"`
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

func main() {
	var cfg configs
	if err := stepconf.Parse(&cfg); err != nil {
		fail("Issue with input: %s", err)
	}

	stepconf.Print(cfg)
	fmt.Println()
	log.SetEnableDebugLog(cfg.VerboseLog)

	if cfg.ExportMethod == "auto-detect" {
		exportMethods := []exportoptions.Method{exportoptions.MethodAppStore, exportoptions.MethodAdHoc, exportoptions.MethodEnterprise, exportoptions.MethodDevelopment}
		log.Warnf("Export method: auto-detect is DEPRECATED, use a direct export method %s", exportMethods)
		fmt.Println()
	}

	if cfg.Workdir != "" {
		if err := input.ValidateIfDirExists(cfg.Workdir); err != nil {
			fail("issue with input Workdir: " + err.Error())
		}
	}

	if cfg.CustomExportOptionsPlistContent != "" {
		var options map[string]interface{}
		if _, err := plist.Unmarshal([]byte(cfg.CustomExportOptionsPlistContent), &options); err != nil {
			fail("issue with input CustomExportOptionsPlistContent: " + err.Error())
		}
	}

	log.Infof("step determined configs:")

	// Detect Xcode major version
	xcodebuildVersion, err := utility.GetXcodeVersion()
	if err != nil {
		fail("Failed to determin xcode version, error: %s", err)
	}
	log.Printf("- xcodebuildVersion: %s (%s)", xcodebuildVersion.Version, xcodebuildVersion.BuildVersion)

	xcodeMajorVersion := xcodebuildVersion.MajorVersion
	if xcodeMajorVersion < minSupportedXcodeMajorVersion {
		fail("Invalid xcode major version (%d), should not be less then min supported: %d", xcodeMajorVersion, minSupportedXcodeMajorVersion)
	}

	// Detect xcpretty version
	outputTool := cfg.OutputTool
	if outputTool == "xcpretty" {
		fmt.Println()
		log.Infof("Checking if output tool (xcpretty) is installed")

		installed, err := xcpretty.IsInstalled()
		if err != nil {
			log.Warnf("Failed to check if xcpretty is installed, error: %s", err)
			log.Printf("Switching to xcodebuild for output tool")
			outputTool = "xcodebuild"
		} else if !installed {
			log.Warnf(`xcpretty is not installed`)
			fmt.Println()
			log.Printf("Installing xcpretty")

			if cmds, err := xcpretty.Install(); err != nil {
				log.Warnf("Failed to create xcpretty install command: %s", err)
				log.Warnf("Switching to xcodebuild for output tool")
				outputTool = "xcodebuild"
			} else {
				for _, cmd := range cmds {
					if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
						if errorutil.IsExitStatusError(err) {
							log.Warnf("%s failed: %s", out)
						} else {
							log.Warnf("%s failed: %s", err)
						}
						log.Warnf("Switching to xcodebuild for output tool")
						outputTool = "xcodebuild"
					}
				}
			}
		}
	}

	if outputTool == "xcpretty" {
		xcprettyVersion, err := xcpretty.Version()
		if err != nil {
			log.Warnf("Failed to determin xcpretty version, error: %s", err)
			log.Printf("Switching to xcodebuild for output tool")
			outputTool = "xcodebuild"
		}
		log.Printf("- xcprettyVersion: %s", xcprettyVersion.String())
	}

	// Validation CustomExportOptionsPlistContent
	customExportOptionsPlistContent := strings.TrimSpace(cfg.CustomExportOptionsPlistContent)
	if customExportOptionsPlistContent != cfg.CustomExportOptionsPlistContent {
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
			log.Printf("- ExportMethod: %s", cfg.ExportMethod)
			log.Printf("- UploadBitcode: %s", cfg.UploadBitcode)
			log.Printf("- CompileBitcode: %s", cfg.CompileBitcode)
			log.Printf("- TeamID: %s", cfg.TeamID)
			fmt.Println()
		}
	}

	if cfg.ForceProvisioningProfileSpecifier != "" &&
		xcodeMajorVersion < 8 {
		fmt.Println()
		log.Warnf("ForceProvisioningProfileSpecifier is set, but ForceProvisioningProfileSpecifier only used if xcodeMajorVersion > 7")
		cfg.ForceProvisioningProfileSpecifier = ""
	}

	if cfg.ForceTeamID != "" &&
		xcodeMajorVersion < 8 {
		fmt.Println()
		log.Warnf("ForceTeamID is set, but ForceTeamID only used if xcodeMajorVersion > 7")
		cfg.ForceTeamID = ""
	}

	if cfg.ForceProvisioningProfileSpecifier != "" &&
		cfg.ForceProvisioningProfile != "" {
		fmt.Println()
		log.Warnf("both ForceProvisioningProfileSpecifier and ForceProvisioningProfile are set, using ForceProvisioningProfileSpecifier")
		cfg.ForceProvisioningProfile = ""
	}

	fmt.Println()

	absProjectPath, err := filepath.Abs(cfg.ProjectPath)
	if err != nil {
		fail("Failed to get absolute project path, error: %s", err)
	}

	// abs out dir pth
	absOutputDir, err := pathutil.AbsPath(cfg.OutputDir)
	if err != nil {
		fail("Failed to expand OutputDir (%s), error: %s", cfg.OutputDir, err)
	}
	cfg.OutputDir = absOutputDir

	if exist, err := pathutil.IsPathExists(cfg.OutputDir); err != nil {
		fail("Failed to check if OutputDir exist, error: %s", err)
	} else if !exist {
		if err := os.MkdirAll(cfg.OutputDir, 0777); err != nil {
			fail("Failed to create OutputDir (%s), error: %s", cfg.OutputDir, err)
		}
	}

	// output files
	tmpArchiveDir, err := pathutil.NormalizedOSTempDirPath("__archive__")
	if err != nil {
		fail("Failed to create temp dir for archives, error: %s", err)
	}
	tmpArchivePath := filepath.Join(tmpArchiveDir, cfg.ArtifactName+".xcarchive")

	appPath := filepath.Join(cfg.OutputDir, cfg.ArtifactName+".app")
	ipaPath := filepath.Join(cfg.OutputDir, cfg.ArtifactName+".ipa")
	exportOptionsPath := filepath.Join(cfg.OutputDir, "export_options.plist")
	rawXcodebuildOutputLogPath := filepath.Join(cfg.OutputDir, "raw-xcodebuild-output.log")

	dsymZipPath := filepath.Join(cfg.OutputDir, cfg.ArtifactName+".dSYM.zip")
	archiveZipPath := filepath.Join(cfg.OutputDir, cfg.ArtifactName+".xcarchive.zip")
	ideDistributionLogsZipPath := filepath.Join(cfg.OutputDir, "xcodebuild.xcdistributionlogs.zip")

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
	ext := filepath.Ext(absProjectPath)
	if ext == ".xcodeproj" {
		isWorkspace = false
	} else if ext == ".xcworkspace" {
		isWorkspace = true
	} else {
		fail("Project file extension should be .xcodeproj or .xcworkspace, but got: %s", ext)
	}

	archiveCmd := xcodebuild.NewCommandBuilder(absProjectPath, isWorkspace, xcodebuild.ArchiveAction)
	archiveCmd.SetScheme(cfg.Scheme)
	archiveCmd.SetConfiguration(cfg.Configuration)

	if cfg.ForceTeamID != "" {
		log.Printf("Forcing Development Team: %s", cfg.ForceTeamID)
		archiveCmd.SetForceDevelopmentTeam(cfg.ForceTeamID)
	}
	if cfg.ForceProvisioningProfileSpecifier != "" {
		log.Printf("Forcing Provisioning Profile Specifier: %s", cfg.ForceProvisioningProfileSpecifier)
		archiveCmd.SetForceProvisioningProfileSpecifier(cfg.ForceProvisioningProfileSpecifier)
	}
	if cfg.ForceProvisioningProfile != "" {
		log.Printf("Forcing Provisioning Profile: %s", cfg.ForceProvisioningProfile)
		archiveCmd.SetForceProvisioningProfile(cfg.ForceProvisioningProfile)
	}
	if cfg.ForceCodeSignIdentity != "" {
		log.Printf("Forcing Code Signing Identity: %s", cfg.ForceCodeSignIdentity)
		archiveCmd.SetForceCodeSignIdentity(cfg.ForceCodeSignIdentity)
	}

	if cfg.IsCleanBuild {
		archiveCmd.SetCustomBuildAction("clean")
	}

	archiveCmd.SetDisableIndexWhileBuilding(cfg.DisableIndexWhileBuilding)
	archiveCmd.SetArchivePath(tmpArchivePath)

	if cfg.XcodebuildOptions != "" {
		options, err := shellquote.Split(cfg.XcodebuildOptions)
		if err != nil {
			fail("Failed to shell split XcodebuildOptions (%s), error: %s", cfg.XcodebuildOptions)
		}
		archiveCmd.SetCustomOptions(options)
	}

	var swiftPackagesPath string
	if xcodeMajorVersion >= 11 {
		var err error
		if swiftPackagesPath, err = cache.SwiftPackagesPath(absProjectPath); err != nil {
			fail("Failed to get Swift Packages path, error: %s", err)
		}
	}

	rawXcodebuildOut, err := runArchiveCommandWithRetry(archiveCmd, outputTool == "xcpretty", swiftPackagesPath)
	if err != nil {
		if outputTool == "xcpretty" {
			log.Errorf("\nLast lines of the Xcode's build log:")
			fmt.Println(stringutil.LastNLines(rawXcodebuildOut, 10))

			if err := utils.ExportOutputFileContent(rawXcodebuildOut, rawXcodebuildOutputLogPath, bitriseXcodeRawResultTextEnvKey); err != nil {
				log.Warnf("Failed to export %s, error: %s", bitriseXcodeRawResultTextEnvKey, err)
			} else {
				log.Warnf(`You can find the last couple of lines of Xcode's build log above, but the full log is also available in the raw-xcodebuild-output.log
	The log file is stored in $BITRISE_DEPLOY_DIR, and its full path is available in the $BITRISE_XCODE_RAW_RESULT_TEXT_PATH environment variable
	(value: %s)`, rawXcodebuildOutputLogPath)
			}
		}
		fail("Archive failed, error: %s", err)
	}

	fmt.Println()

	// Ensure xcarchive exists
	if exist, err := pathutil.IsPathExists(tmpArchivePath); err != nil {
		fail("Failed to check if archive exist, error: %s", err)
	} else if !exist {
		fail("No archive generated at: %s", tmpArchivePath)
	}

	// Cache swift PM
	if xcodeMajorVersion >= 11 && cfg.CacheLevel == "swift_packages" {
		if err := cache.CollectSwiftPackages(absProjectPath); err != nil {
			log.Warnf("Failed to mark swift packages for caching, error: %s", err)
		}
	}

	if xcodeMajorVersion >= 9 && cfg.UseDeprecatedExport {
		fail("Legacy export method (using '-exportFormat ipa' flag) is not supported from Xcode version 9")
	}

	envsToUnset := []string{"GEM_HOME", "GEM_PATH", "RUBYLIB", "RUBYOPT", "BUNDLE_BIN_PATH", "_ORIGINAL_GEM_PATH", "BUNDLE_GEMFILE"}
	for _, key := range envsToUnset {
		if err := os.Unsetenv(key); err != nil {
			fail("Failed to unset (%s), error: %s", key, err)
		}
	}

	archive, err := xcarchive.NewIosArchive(tmpArchivePath)
	if err != nil {
		fail("Failed to parse archive, error: %s", err)
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

	if xcodeMajorVersion <= 6 || cfg.UseDeprecatedExport {
		log.Printf("Using legacy export")
		/*
			Get the name of the profile which was used for creating the archive
			--> Search for embedded.mobileprovision in the xcarchive.
			It should contain a .app folder in the xcarchive folder
			under the Products/Applications folder
		*/

		legacyExportCmd := xcodebuild.NewLegacyExportCommand()
		legacyExportCmd.SetExportFormat("ipa")
		legacyExportCmd.SetArchivePath(tmpArchivePath)
		legacyExportCmd.SetExportPath(ipaPath)
		legacyExportCmd.SetExportProvisioningProfileName(mainApplication.ProvisioningProfile.Name)

		if outputTool == "xcpretty" {
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

			if cfg.ExportMethod == "auto-detect" {
				log.Printf("auto-detect export method specified")
				exportMethod = archiveExportMethod

				log.Printf("using the archive profile's (%s) export method: %s", mainApplication.ProvisioningProfile.Name, exportMethod)
			} else {
				parsedMethod, err := exportoptions.ParseMethod(cfg.ExportMethod)
				if err != nil {
					fail("Failed to parse export options, error: %s", err)
				}
				exportMethod = parsedMethod
				log.Printf("export-method specified: %s", cfg.ExportMethod)
			}

			bundleIDEntitlementsMap, err := utils.ProjectEntitlementsByBundleID(absProjectPath, cfg.Scheme, cfg.Configuration)
			if err != nil {
				fail(err.Error())
			}

			// iCloudContainerEnvironment: If the app is using CloudKit, this configures the "com.apple.developer.icloud-container-environment" entitlement.
			// Available options vary depending on the type of provisioning profile used, but may include: Development and Production.
			usesCloudKit := false
			for _, entitlements := range bundleIDEntitlementsMap {
				if entitlements == nil {
					continue
				}

				services, ok := entitlements.GetStringArray("com.apple.developer.icloud-services")
				if ok {
					usesCloudKit = sliceutil.IsStringInSlice("CloudKit", services) || sliceutil.IsStringInSlice("CloudDocuments", services)
					if usesCloudKit {
						break
					}
				}
			}

			// From Xcode 9 iCloudContainerEnvironment is required for every export method, before that version only for non app-store exports.
			var iCloudContainerEnvironment string
			if usesCloudKit && (xcodeMajorVersion >= 9 || exportMethod != exportoptions.MethodAppStore) {
				if exportMethod == exportoptions.MethodAppStore {
					iCloudContainerEnvironment = "Production"
				} else if cfg.ICloudContainerEnvironment == "" {
					fail("project uses CloudKit, but iCloud container environment input not specified")
				} else {
					iCloudContainerEnvironment = cfg.ICloudContainerEnvironment
				}
			}

			if xcodeMajorVersion >= 9 {
				log.Printf("xcode major version > 9, generating provisioningProfiles node")

				fmt.Println()
				log.Printf("Target Bundle ID - Entitlements map")
				var bundleIDs []string
				for bundleID, entitlements := range bundleIDEntitlementsMap {
					bundleIDs = append(bundleIDs, bundleID)

					entitlementKeys := []string{}
					for key := range entitlements {
						entitlementKeys = append(entitlementKeys, key)
					}
					log.Printf("%s: %s", bundleID, entitlementKeys)
				}

				fmt.Println()
				log.Printf("Resolving CodeSignGroups...")

				certs, err := certificateutil.InstalledCodesigningCertificateInfos()
				if err != nil {
					fail("Failed to get installed certificates, error: %s", err)
				}
				certs = certificateutil.FilterValidCertificateInfos(certs).ValidCertificates

				log.Debugf("Installed certificates:")
				for _, certInfo := range certs {
					log.Debugf(certInfo.String())
				}

				profs, err := profileutil.InstalledProvisioningProfileInfos(profileutil.ProfileTypeIos)
				if err != nil {
					fail("Failed to get installed provisioning profiles, error: %s", err)
				}

				log.Debugf("Installed profiles:")
				for _, profileInfo := range profs {
					log.Debugf(profileInfo.String(certs...))
				}

				log.Printf("Resolving CodeSignGroups...")
				codeSignGroups := export.CreateSelectableCodeSignGroups(certs, profs, bundleIDs)
				if len(codeSignGroups) == 0 {
					log.Errorf("Failed to find code signing groups for specified export method (%s)", exportMethod)
				}

				log.Debugf("\nGroups:")
				for _, group := range codeSignGroups {
					log.Debugf(group.String())
				}

				if len(bundleIDEntitlementsMap) > 0 {
					log.Warnf("Filtering CodeSignInfo groups for target capabilities")

					codeSignGroups = export.FilterSelectableCodeSignGroups(codeSignGroups, export.CreateEntitlementsSelectableCodeSignGroupFilter(bundleIDEntitlementsMap))

					log.Debugf("\nGroups after filtering for target capabilities:")
					for _, group := range codeSignGroups {
						log.Debugf(group.String())
					}
				}

				log.Warnf("Filtering CodeSignInfo groups for export method")

				codeSignGroups = export.FilterSelectableCodeSignGroups(codeSignGroups, export.CreateExportMethodSelectableCodeSignGroupFilter(exportMethod))

				log.Debugf("\nGroups after filtering for export method:")
				for _, group := range codeSignGroups {
					log.Debugf(group.String())
				}

				if cfg.TeamID != "" {
					log.Warnf("Export TeamID specified: %s, filtering CodeSignInfo groups...", cfg.TeamID)

					codeSignGroups = export.FilterSelectableCodeSignGroups(codeSignGroups, export.CreateTeamSelectableCodeSignGroupFilter(cfg.TeamID))

					log.Debugf("\nGroups after filtering for team ID:")
					for _, group := range codeSignGroups {
						log.Debugf(group.String())
					}
				}

				if !archiveCodeSignIsXcodeManaged {
					log.Warnf("App was signed with NON xcode managed profile when archiving,\n" +
						"only NOT xcode managed profiles are allowed to sign when exporting the archive.\n" +
						"Removing xcode managed CodeSignInfo groups")

					codeSignGroups = export.FilterSelectableCodeSignGroups(codeSignGroups, export.CreateNotXcodeManagedSelectableCodeSignGroupFilter())

					log.Debugf("\nGroups after filtering for NOT Xcode managed profiles:")
					for _, group := range codeSignGroups {
						log.Debugf(group.String())
					}
				}

				defaultProfileURL := os.Getenv("BITRISE_DEFAULT_PROVISION_URL")
				if cfg.TeamID == "" && defaultProfileURL != "" {
					if defaultProfile, err := utils.GetDefaultProvisioningProfile(); err == nil {
						log.Debugf("\ndefault profile: %v\n", defaultProfile)
						filteredCodeSignGroups := export.FilterSelectableCodeSignGroups(codeSignGroups,
							export.CreateExcludeProfileNameSelectableCodeSignGroupFilter(defaultProfile.Name))
						if len(filteredCodeSignGroups) > 0 {
							codeSignGroups = filteredCodeSignGroups

							log.Debugf("\nGroups after removing default profile:")
							for _, group := range codeSignGroups {
								log.Debugf(group.String())
							}
						}
					}
				}

				var iosCodeSignGroups []export.IosCodeSignGroup

				for _, selectable := range codeSignGroups {
					bundleIDProfileMap := map[string]profileutil.ProvisioningProfileInfoModel{}
					for bundleID, profiles := range selectable.BundleIDProfilesMap {
						if len(profiles) > 0 {
							bundleIDProfileMap[bundleID] = profiles[0]
						} else {
							log.Warnf("No profile available to sign (%s) target!", bundleID)
						}
					}

					iosCodeSignGroups = append(iosCodeSignGroups, *export.NewIOSGroup(selectable.Certificate, bundleIDProfileMap))
				}

				log.Debugf("\nFiltered groups:")
				for i, group := range iosCodeSignGroups {
					log.Debugf("Group #%d:", i)
					for bundleID, profile := range group.BundleIDProfileMap() {
						log.Debugf(" - %s: %s (%s)", bundleID, profile.Name, profile.UUID)
					}
				}

				if len(iosCodeSignGroups) > 0 {
					codeSignGroup := export.IosCodeSignGroup{}

					if len(iosCodeSignGroups) >= 1 {
						codeSignGroup = iosCodeSignGroups[0]
					}
					if len(iosCodeSignGroups) > 1 {
						log.Warnf("Multiple code signing groups found! Using the first code signing group")
					}

					exportTeamID = codeSignGroup.Certificate().TeamID
					exportCodeSignIdentity = codeSignGroup.Certificate().CommonName

					for bundleID, profileInfo := range codeSignGroup.BundleIDProfileMap() {
						exportProfileMapping[bundleID] = profileInfo.Name

						isXcodeManaged := profileutil.IsXcodeManaged(profileInfo.Name)
						if isXcodeManaged {
							if exportCodeSignStyle != "" && exportCodeSignStyle != "automatic" {
								log.Errorf("Both xcode managed and NON xcode managed profiles in code signing group")
							}
							exportCodeSignStyle = "automatic"
						} else {
							if exportCodeSignStyle != "" && exportCodeSignStyle != "manual" {
								log.Errorf("Both xcode managed and NON xcode managed profiles in code signing group")
							}
							exportCodeSignStyle = "manual"
						}
					}
				} else {
					log.Errorf("Failed to find Codesign Groups")
				}
			}

			var exportOpts exportoptions.ExportOptions
			if exportMethod == exportoptions.MethodAppStore {
				options := exportoptions.NewAppStoreOptions()
				options.UploadBitcode = cfg.UploadBitcode

				if xcodeMajorVersion >= 9 {
					options.BundleIDProvisioningProfileMapping = exportProfileMapping
					options.SigningCertificate = exportCodeSignIdentity
					options.TeamID = exportTeamID

					if archiveCodeSignIsXcodeManaged && exportCodeSignStyle == "manual" {
						log.Warnf("App was signed with xcode managed profile when archiving,")
						log.Warnf("ipa export uses manual code signing.")
						log.Warnf(`Setting "signingStyle" to "manual"`)

						options.SigningStyle = "manual"
					}
				}

				if iCloudContainerEnvironment != "" {
					options.ICloudContainerEnvironment = exportoptions.ICloudContainerEnvironment(iCloudContainerEnvironment)
				}

				exportOpts = options
			} else {
				options := exportoptions.NewNonAppStoreOptions(exportMethod)
				options.CompileBitcode = cfg.CompileBitcode

				if xcodeMajorVersion >= 9 {
					options.BundleIDProvisioningProfileMapping = exportProfileMapping
					options.SigningCertificate = exportCodeSignIdentity
					options.TeamID = exportTeamID

					if archiveCodeSignIsXcodeManaged && exportCodeSignStyle == "manual" {
						log.Warnf("App was signed with xcode managed profile when archiving,")
						log.Warnf("ipa export uses manual code signing.")
						log.Warnf(`Setting "signingStyle" to "manual"`)

						options.SigningStyle = "manual"
					}
				}

				if iCloudContainerEnvironment != "" {
					options.ICloudContainerEnvironment = exportoptions.ICloudContainerEnvironment(iCloudContainerEnvironment)
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

		if outputTool == "xcpretty" {
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
					deployPth := filepath.Join(cfg.OutputDir, base)

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

	if err := utils.ExportOutputDirAsZip(tmpArchivePath, archiveZipPath, bitriseXCArchiveZipPthEnvKey); err != nil {
		fail("Failed to export %s, error: %s", bitriseXCArchiveZipPthEnvKey, err)
	}

	log.Donef("The xcarchive zip path is now available in the Environment Variable: %s (value: %s)", bitriseXCArchiveZipPthEnvKey, archiveZipPath)

	// Export .app
	fmt.Println()

	exportedApp := mainApplication.Path

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

	appDSYM, frameworkDSYMs, err := archive.FindDSYMs()
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

		if cfg.ExportAllDsyms {
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
