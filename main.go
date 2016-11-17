package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/steps-xcode-archive/utils"
	"github.com/bitrise-io/steps-xcode-archive/xcodebuild"
	"github.com/bitrise-io/steps-xcode-archive/xcpretty"
	"github.com/bitrise-tools/go-xcode/exportoptions"
	"github.com/bitrise-tools/go-xcode/provisioningprofile"
	"github.com/bitrise-tools/go-xcode/xcarchive"
	"github.com/kballard/go-shellquote"
)

const (
	minSupportedXcodeMajorVersion = 6
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
		return fmt.Errorf("Invalid OutputTool specified (%s), valid options: [xcpretty xcodebuild]", configs.OutputTool)
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
		return fmt.Errorf("Invalid IsExportXcarchiveZip specified (%s), valid options: [yes no]", configs.IsExportXcarchiveZip)
	}

	if configs.UseDeprecatedExport == "" {
		return errors.New("no UseDeprecatedExport parameter specified")
	}
	if configs.UseDeprecatedExport != "yes" && configs.UseDeprecatedExport != "no" {
		return fmt.Errorf("Invalid UseDeprecatedExport specified (%s), valid options: [yes no]", configs.UseDeprecatedExport)
	}

	if configs.ExportAllDsyms == "" {
		return errors.New("no ExportAllDsyms parameter specified")
	}
	if configs.ExportAllDsyms != "yes" && configs.ExportAllDsyms != "no" {
		return fmt.Errorf("invalid ExportAllDsyms specified (%s), valid options: [yes no]", configs.ExportAllDsyms)
	}

	return nil
}

func exportEnvironmentWithEnvman(keyStr, valueStr string) error {
	cmd := cmdex.NewCommand("envman", "add", "--key", keyStr)
	cmd.SetStdin(strings.NewReader(valueStr))
	return cmd.Run()
}

func fail(format string, v ...interface{}) {
	log.Error(format, v...)
	os.Exit(1)
}

func zip(sourceDir, destinationZipPth string) error {
	parentDir := filepath.Dir(sourceDir)
	dirName := filepath.Base(sourceDir)
	cmd := cmdex.NewCommand("/usr/bin/zip", "-rTy", destinationZipPth, dirName)
	cmd.SetDir(parentDir)
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to zip dir: %s, output: %s, error: %s", sourceDir, out, err)
	}

	return nil
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
		xcodeMajorVersion == 6 {
		log.Warn("xcodeMajorVersion = 6, CustomExportOptionsPlistContent only used if xcodeMajorVersion > 6")
		configs.CustomExportOptionsPlistContent = ""
	}

	if configs.ForceProvisioningProfileSpecifier != "" &&
		xcodeMajorVersion < 8 {
		log.Warn("ForceProvisioningProfileSpecifier is set but, ForceProvisioningProfileSpecifier only used if xcodeMajorVersion > 7")
		configs.ForceProvisioningProfileSpecifier = ""
	}

	if configs.ForceTeamID == "" &&
		xcodeMajorVersion < 8 {
		log.Warn("force_team_id is set but, force_team_id only used if xcodeMajorVersion > 7")
		configs.ForceTeamID = ""
	}

	if configs.ForceProvisioningProfileSpecifier != "" &&
		configs.ForceProvisioningProfile != "" {
		log.Warn("both ForceProvisioningProfileSpecifier and ForceProvisioningProfile are set, using ForceProvisioningProfileSpecifier")
		configs.ForceProvisioningProfile = ""
	}

	// project or workspace flag
	projectAction := ""
	ext := filepath.Ext(configs.ProjectPath)
	if ext == ".xcodeproj" {
		projectAction = "-project"
	} else if ext == ".xcworkspace" {
		projectAction = "-workspace"
	} else {
		fail("Project file extension should .xcodeproj or .xcworkspace, but got: %s", ext)
	}
	log.Detail("- projectAction: %s", projectAction)

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
	archiveZipPath := filepath.Join(configs.OutputDir, configs.ArtifactName+".xcarchive.zip")
	dsymZipPath := filepath.Join(configs.OutputDir, configs.ArtifactName+".dSYM.zip")
	rawXcodebuildOutputLogPath := filepath.Join(configs.OutputDir, "raw-xcodebuild-output.log")
	exportOptionsPath := filepath.Join(configs.OutputDir, "export_options.plist")

	// cleanup
	filesToCleanup := []string{
		appPath,
		ipaPath,
		archiveZipPath,
		dsymZipPath,
		rawXcodebuildOutputLogPath,
		exportOptionsPath,
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

	xcodebuildCmd := xcodebuild.New()
	xcodebuildCmd.SetProjectAction(projectAction)
	xcodebuildCmd.SetProjectPath(configs.ProjectPath)
	xcodebuildCmd.SetScheme(configs.Scheme)
	xcodebuildCmd.SetConfiguration(configs.Configuration)
	xcodebuildCmd.SetIsCleanBuild(configs.IsCleanBuild == "yes")
	xcodebuildCmd.SetArchivePath(tmpArchivePath)

	if configs.ForceTeamID != "" {
		log.Detail("Forcing Development Team: %s", configs.ForceTeamID)
		xcodebuildCmd.SetForceDevelopmentTeam(configs.ForceTeamID)
	}

	if configs.ForceProvisioningProfileSpecifier != "" {
		log.Detail("Forcing Provisioning Profile Specifier: %s", configs.ForceProvisioningProfileSpecifier)
		xcodebuildCmd.SetForceProvisioningProfileSpecifier(configs.ForceProvisioningProfileSpecifier)
	}

	if configs.ForceProvisioningProfile != "" {
		log.Detail("Forcing Provisioning Profile: %s", configs.ForceProvisioningProfile)
		xcodebuildCmd.SetForceProvisioningProfile(configs.ForceProvisioningProfile)
	}

	if configs.ForceCodeSignIdentity != "" {
		log.Detail("Forcing Code Signing Identity: %s", configs.ForceCodeSignIdentity)
		xcodebuildCmd.SetForceCodeSignIdentity(configs.ForceCodeSignIdentity)
	}

	if configs.XcodebuildOptions != "" {
		options, err := shellquote.Split(configs.XcodebuildOptions)
		if err != nil {
			fail("Failed to shell split XcodebuildOptions (%s), error: %s", configs.XcodebuildOptions)
		}
		xcodebuildCmd.SetCustomOptions(options)
	}

	if configs.OutputTool == "xcpretty" {
		xcprettyCmd := xcpretty.New()

		archiveCmd, err := xcodebuildCmd.ArchiveCmd()
		if err != nil {
			fail("Failed to create archive command, error: %s", err)
		}

		xcprettyCmd.SetCmdToPretty(archiveCmd)

		log.Done("$ %s", xcprettyCmd.PrintableCmd())
		fmt.Println()

		rawXcodebuildOut, err := xcprettyCmd.Run()
		if err != nil {
			if err := fileutil.WriteStringToFile(rawXcodebuildOutputLogPath, rawXcodebuildOut); err != nil {
				log.Warn("Failed to write raw xcodebuild log, error: %s", err)
			} else if err := exportEnvironmentWithEnvman("BITRISE_XCODE_RAW_RESULT_TEXT_PATH", rawXcodebuildOutputLogPath); err != nil {
				log.Warn("Failed to export xcodebuild raw log path, error: %s", err)
			} else {
				log.Warn(`If you can't find the reason of the error in the log, please check the raw-xcodebuild-output.log
The log file is stored in \$BITRISE_DEPLOY_DIR, and its full path
is available in the \$BITRISE_XCODE_RAW_RESULT_TEXT_PATH environment variable`)
			}

			fail("Archive failed, error: %s", err)
		}
	} else {
		log.Done("$ %s", xcodebuildCmd.PrintableArchiveCmd())
		fmt.Println()

		if err := xcodebuildCmd.Archive(); err != nil {
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

		xcodebuildCmd := xcodebuild.New()
		xcodebuildCmd.SetExportFormat("ipa")
		xcodebuildCmd.SetArchivePath(tmpArchivePath)
		xcodebuildCmd.SetExportPath(ipaPath)
		xcodebuildCmd.SetExportProvisioningProfile(*provProfile.Name)

		if configs.OutputTool == "xcpretty" {
			exportCmd, err := xcodebuildCmd.LegacyExportCmd()
			if err != nil {
				fail("Failed to create export command, error: %s", err)
			}

			xcprettyCmd := xcpretty.New()
			xcprettyCmd.SetCmdToPretty(exportCmd)

			log.Done("$ %s", xcprettyCmd.PrintableCmd())
			fmt.Println()

			rawXcodebuildOut, err := xcprettyCmd.Run()
			if err != nil {
				if err := fileutil.WriteStringToFile(rawXcodebuildOutputLogPath, rawXcodebuildOut); err != nil {
					log.Warn("Failed to write raw xcodebuild log, error: %s", err)
				} else if err := exportEnvironmentWithEnvman("BITRISE_XCODE_RAW_RESULT_TEXT_PATH", rawXcodebuildOutputLogPath); err != nil {
					log.Warn("Failed to export xcodebuild raw log path, error: %s", err)
				} else {
					log.Warn(`If you can't find the reason of the error in the log, please check the raw-xcodebuild-output.log
The log file is stored in \$BITRISE_DEPLOY_DIR, and its full path
is available in the \$BITRISE_XCODE_RAW_RESULT_TEXT_PATH environment variable`)
				}

				fail("Export failed, error: %s", err)
			}
		} else {
			log.Done("$ %s", xcodebuildCmd.PrintableLegacyExportCmd())
			fmt.Println()

			if err := xcodebuildCmd.LegacyExport(); err != nil {
				fail("Export failed, error: %s", err)
			}
		}
	} else {
		log.Detail("Using export options")

		if configs.CustomExportOptionsPlistContent != "" {
			log.Detail("Custom export options content provided:")
			fmt.Println(configs.CustomExportOptionsPlistContent)

			if err := fileutil.WriteStringToFile(exportOptionsPath, configs.CustomExportOptionsPlistContent); err != nil {
				log.Error("Failed to write export options to file, error: %s", err)
				os.Exit(1)
			}
		} else {
			log.Detail("Generating export options")

			/*
			   Because of an RVM issue which conflicts with `xcodebuild`'s new
			   `-exportOptionsPlist` option
			   link: https://github.com/bitrise-io/steps-xcode-archive/issues/13
			*/
			if err := applyRVMFix(); err != nil {
				fail("rvm fix failed, error: %s", err)
			}

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

		xcodebuildCmd := xcodebuild.New()
		xcodebuildCmd.SetArchivePath(tmpArchivePath)
		xcodebuildCmd.SetExportPath(tmpDir)
		xcodebuildCmd.SetExportOptionsPlist(exportOptionsPath)

		if configs.OutputTool == "xcpretty" {
			xcprettyCmd := xcpretty.New()

			exportCmd, err := xcodebuildCmd.ExportCmd()
			if err != nil {
				fail("Failed to create export command, error: %s", err)
			}

			xcprettyCmd.SetCmdToPretty(exportCmd)

			log.Done("$ %s", xcprettyCmd.PrintableCmd())
			fmt.Println()

			xcodebuildOut, xcprettyErr := xcprettyCmd.Run()
			logPth, err := findIDEDistrubutionLogsPath(xcodebuildOut)
			if err != nil {
				log.Warn("Failed to find xcdistributionlogs, error: %s", err)
			}

			if err := exportEnvironmentWithEnvman("BITRISE_IDEDISTRIBUTION_LOGS_PATH", logPth); err != nil {
				fail("Failed to export xcdistributionlogs path, error: %s", err)
			}

			if xcprettyErr != nil {
				if err := exportEnvironmentWithEnvman("BITRISE_XCODE_RAW_RESULT_TEXT_PATH", xcodebuildOut); err != nil {
					fail("Failed to export xcodebuild raw log path, error: %s", err)
				}

				fail("Export failed, error: %s", err)
			}
		} else {
			log.Done("$ %s", xcodebuildCmd.PrintableExportCmd())
			fmt.Println()

			xcodebuildOut, xcodebuildErr := xcodebuildCmd.Export()
			logPth, err := findIDEDistrubutionLogsPath(xcodebuildOut)
			if err != nil {
				log.Warn("Failed to find xcdistributionlogs, error: %s", err)
			}

			if err := exportEnvironmentWithEnvman("BITRISE_IDEDISTRIBUTION_LOGS_PATH", logPth); err != nil {
				fail("Failed to export xcdistributionlogs path, error: %s", err)
			}

			if xcodebuildErr != nil {
				fail("Export failed, error: %s", xcodebuildErr)
			}
		}

		// Search for ipa
		exportedIPA := ""

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
				ipaPath = exportedIPA
			}
		}
	}

	log.Info("Exporting outputs...")

	//
	// Export outputs

	// Export .xcarchive
	fmt.Println()

	if err := exportEnvironmentWithEnvman("BITRISE_XCARCHIVE_PATH", tmpArchiveDir); err != nil {
		fail("Failed to export xcarchivepath, error: %s", err)
	}

	log.Done("The xcarchive path is now available in the Environment Variable: $BITRISE_XCARCHIVE_PATH (value: %s)", archiveZipPath)

	if configs.IsExportXcarchiveZip == "yes" {
		if err := zip(tmpArchiveDir, archiveZipPath); err != nil {
			fail("zip failed, error: %s", err)
		}

		if err := exportEnvironmentWithEnvman("BITRISE_XCARCHIVE_ZIP_PATH", archiveZipPath); err != nil {
			fail("Failed to export xcarchive zip path, error: %s", err)
		}

		log.Done("The xcarchive zip path is now available in the Environment Variable: $BITRISE_XCARCHIVE_ZIP_PATH (value: %s)", archiveZipPath)
	}

	// Export .app
	fmt.Println()

	exportedApp := ""

	pattern := filepath.Join(tmpArchivePath, "Products/Applications", "*.app")
	apps, err := filepath.Glob(pattern)
	if err != nil {
		fail("Failed to find .app directories, error: %s", err)
	}

	if len(apps) == 0 {
		log.Warn("No app found with pattern (%s)", pattern)
	} else if len(apps) == 1 {
		if err := cmdex.CopyDir(apps[0], appPath, true); err != nil {
			fail("Failed to copy (%s) -> (%s), error: %s", apps[0], appPath, err)
		}
		exportedApp = appPath
	} else {
		log.Warn("More than 1 .app directory found")

		for _, app := range apps {
			base := filepath.Base(app)
			deployPth := filepath.Join(configs.OutputDir, base)

			if err := cmdex.CopyDir(app, deployPth, true); err != nil {
				fail("Failed to copy (%s) -> (%s), error: %s", app, deployPth, err)
			}

			exportedApp = deployPth
		}
	}

	if exportedApp != "" {
		if err := exportEnvironmentWithEnvman("BITRISE_APP_DIR_PATH", exportedApp); err != nil {
			fail("Failed to export .app path, error: %s", err)
		}

		log.Done("The app directory is now available in the Environment Variable: $BITRISE_APP_DIR_PATH (value: %s)", exportedApp)
	}

	// Export .ipa
	fmt.Println()

	if err := exportEnvironmentWithEnvman("BITRISE_IPA_PATH", ipaPath); err != nil {
		fail("Failed to export ipa path, error: %s", err)
	}

	log.Done("The ipa path is now available in the Environment Variable: $BITRISE_IPA_PATH (value: %s)", ipaPath)

	// Export .dSYMs
	fmt.Println()

	appDSYM, frameworkDSYMs, err := xcarchive.ExportDSYMs(tmpArchivePath)
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

	if err := exportEnvironmentWithEnvman("BITRISE_DSYM_DIR_PATH", dsymDir); err != nil {
		fail("Failed to export dsym path, error: %s", err)
	}

	log.Done("The dSYM dir path is now available in the Environment Variable: $BITRISE_DSYM_DIR_PATH (value: %s)", dsymDir)

	if err := zip(dsymDir, dsymZipPath); err != nil {
		fail("zip failed, error: %s", err)
	}

	if err := exportEnvironmentWithEnvman("BITRISE_DSYM_DIR_ZIP_PATH", dsymZipPath); err != nil {
		fail("Failed to export dsym path, error: %s", err)
	}

	log.Done("The dSYM zip path is now available in the Environment Variable: $BITRISE_DSYM_DIR_ZIP_PATH (value: %s)", dsymZipPath)

	if err := exportEnvironmentWithEnvman("BITRISE_DSYM_PATH", dsymZipPath); err != nil {
		fail("Failed to export dsym path, error: %s", err)
	}

	log.Done("The dSYM zip path is now available in the Environment Variable: $BITRISE_DSYM_PATH (value: %s)", dsymZipPath)
}
