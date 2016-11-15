package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	shellquote "github.com/kballard/go-shellquote"
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
		fail("Invalid xcode major version (%s), should not be less then %d", xcodeMajorVersion, minSupportedXcodeMajorVersion)
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
	log.Info("output paths")
	archiveTmpDir, err := pathutil.NormalizedOSTempDirPath("__archive__")
	if err != nil {
		fail("Failed to create temp dir for archives, error: %s", err)
	}
	archivePath := filepath.Join(archiveTmpDir, configs.ArtifactName+".xcarchive")
	log.Detail("- archivePath: %s", archivePath)

	ipaPath := filepath.Join(configs.OutputDir, configs.ArtifactName+".ipa")
	log.Detail("- ipaPath: %s", ipaPath)

	dsymZipPath := filepath.Join(configs.OutputDir, configs.ArtifactName+".dSYM.zip")
	log.Detail("- dsymZipPath: %s", dsymZipPath)

	// cleanup
	if exist, err := pathutil.IsPathExists(ipaPath); err != nil {
		fail("Failed to check if ipa (%s) exist, error: %s", ipaPath, err)
	} else if exist {
		if err := os.Remove(ipaPath); err != nil {
			fail("Failed to remove ipa (%s), error: %s", ipaPath, err)
		}
	}

	if exist, err := pathutil.IsPathExists(dsymZipPath); err != nil {
		fail("Failed to check if dsym.zip (%s) exist, error: %s", dsymZipPath, err)
	} else if exist {
		if err := os.Remove(dsymZipPath); err != nil {
			fail("Failed to remove dsym.zip (%s), error: %s", dsymZipPath, err)
		}
	}
	fmt.Println()

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
	xcodebuildCmd.SetArchivePath(archivePath)

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
			if err := exportEnvironmentWithEnvman("BITRISE_XCODE_RAW_RESULT_TEXT_PATH", rawXcodebuildOut); err != nil {
				fail("Failed to export xcodebuild raw log path, error: %s", err)
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
	if exist, err := pathutil.IsPathExists(archivePath); err != nil {
		fail("Failed to check if archive exist, error: %s", err)
	} else if !exist {
		fail("No archive generated at: %s", archivePath)
	}

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

	//
	// Export ipa from the archive
	log.Info("Exporting ipa from the archive...")
	fmt.Println()

	envsToUnset := []string{"GEM_HOME", "GEM_PATH", "RUBYLIB", "RUBYOPT", "BUNDLE_BIN_PATH", "_ORIGINAL_GEM_PATH", "BUNDLE_GEMFILE"}
	for _, key := range envsToUnset {
		if err := os.Unsetenv(key); err != nil {
			fail("Failed to unset (%s), error: %s", key, err)
		}
	}

	if xcodeMajorVersion == 6 || configs.UseDeprecatedExport == "yes" {
		log.Warn("Using legacy export")
		/*
			Get the name of the profile which was used for creating the archive
			--> Search for embedded.mobileprovision in the xcarchive.
			It should contain a .app folder in the xcarchive folder
			under the Products/Applications folder
		*/

		embeddedProfilePth, err := xcarchive.EmbeddedMobileProvisionPth(archivePath)
		if err != nil {
			log.Error("Failed to get embedded profile path, error: %s", err)
			os.Exit(1)
		}

		provProfile, err := provisioningprofile.NewFromFile(embeddedProfilePth)
		if err != nil {
			log.Error("Failed to create provisioning profile model, error: %s", err)
			os.Exit(1)
		}

		if provProfile.Name == nil {
			log.Error("Profile name empty")
			os.Exit(1)
		}

		xcodebuildCmd := xcodebuild.New()
		xcodebuildCmd.SetExportFormat("ipa")
		xcodebuildCmd.SetArchivePath(archivePath)
		xcodebuildCmd.SetExportPath(ipaPath)
		xcodebuildCmd.SetExportProvisioningProfile(*provProfile.Name)

		if configs.OutputTool == "xcpretty" {
			xcprettyCmd := xcpretty.New()

			exportCmd, err := xcodebuildCmd.LegacyExportCmd()
			if err != nil {
				fail("Failed to create export command, error: %s", err)
			}

			xcprettyCmd.SetCmdToPretty(exportCmd)

			log.Done("$ %s", xcprettyCmd.PrintableCmd())
			fmt.Println()

			rawXcodebuildOut, err := xcprettyCmd.Run()
			if err != nil {
				if err := exportEnvironmentWithEnvman("BITRISE_XCODE_RAW_RESULT_TEXT_PATH", rawXcodebuildOut); err != nil {
					fail("Failed to export xcodebuild raw log path, error: %s", err)
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
		exportOptionsPth := ""

		if configs.CustomExportOptionsPlistContent != "" {
			log.Detail("Custom export options content provided:")
			fmt.Println(configs.CustomExportOptionsPlistContent)

			tmpDir, err := pathutil.NormalizedOSTempDirPath("export")
			if err != nil {
				log.Error("Failed to create tmp dir, error: %s", err)
				os.Exit(1)
			}
			exportOptionsPth = filepath.Join(tmpDir, "export-options.plist")

			if err := fileutil.WriteStringToFile(exportOptionsPth, configs.CustomExportOptionsPlistContent); err != nil {
				log.Error("Failed to write export options to file, error: %s", err)
				os.Exit(1)
			}
		} else {
			log.Detail("Generating export options")

			var exportOpts exportoptions.ExportOptions

			if configs.ExportMethod == "auto-detect" {
				log.Detail("creating default export options based on embedded profile")

				embeddedProfilePth, err := xcarchive.EmbeddedMobileProvisionPth(archivePath)
				if err != nil {
					log.Error("Failed to get embedded profile path, error: %s", err)
					os.Exit(1)
				}

				provProfile, err := provisioningprofile.NewFromFile(embeddedProfilePth)
				if err != nil {
					log.Error("Failed to create provisioning profile model, error: %s", err)
					os.Exit(1)
				}

				if provProfile.Name != nil {
					log.Detail("embedded profile name: %s", *provProfile.Name)
				}

				options, err := xcarchive.DefaultExportOptions(provProfile)
				if err != nil {
					log.Error("Failed to create default export options, error: %s", err)
					os.Exit(1)
				}

				exportOpts = options
			} else {
				method, err := exportoptions.ParseMethod(configs.ExportMethod)
				if err != nil {
					log.Error("Failed to parse export options, error: %s", err)
					os.Exit(1)
				}

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
			}

			log.Detail("generated export options content:")
			fmt.Println(exportOpts.String())

			var err error
			exportOptionsPth, err = exportOpts.WriteToTmpFile()
			if err != nil {
				log.Error("Failed to write export options to file, error: %s", err)
				os.Exit(1)
			}
		}

		fmt.Println()

		tmpDir, err := pathutil.NormalizedOSTempDirPath("__export__")
		if err != nil {
			fail("Failed to create tmp dir, error: %s", err)
		}

		xcodebuildCmd := xcodebuild.New()
		xcodebuildCmd.SetArchivePath(archivePath)
		xcodebuildCmd.SetExportPath(tmpDir)
		xcodebuildCmd.SetExportOptionsPlist(exportOptionsPth)

		if configs.OutputTool == "xcpretty" {
			xcprettyCmd := xcpretty.New()

			exportCmd, err := xcodebuildCmd.ExportCmd()
			if err != nil {
				fail("Failed to create export command, error: %s", err)
			}

			xcprettyCmd.SetCmdToPretty(exportCmd)

			log.Done("$ %s", xcprettyCmd.PrintableCmd())
			fmt.Println()

			rawXcodebuildOut, err := xcprettyCmd.Run()
			if err != nil {
				if err := exportEnvironmentWithEnvman("BITRISE_XCODE_RAW_RESULT_TEXT_PATH", rawXcodebuildOut); err != nil {
					fail("Failed to export xcodebuild raw log path, error: %s", err)
				}

				fail("Export failed, error: %s", err)
			}
		} else {
			log.Done("$ %s", xcodebuildCmd.PrintableExportCmd())
			fmt.Println()

			if err := xcodebuildCmd.Export(); err != nil {
				fail("Export failed, error: %s", err)
			}
		}

		// Search for ipa
		exportedIPA := ""

		pattern := filepath.Join(tmpDir, "*")
		files, err := filepath.Glob(pattern)
		if err != nil {
			fail("Failed to collect output files, error: %s", err)
		}

		for _, file := range files {
			base := filepath.Base(file)
			ext := filepath.Ext(file)
			deployPth := filepath.Join(configs.OutputDir, base+ext)

			if err := os.Rename(file, deployPth); err != nil {
				fail("Failed to move (%s) -> (%s), error: %s", file, deployPth, err)
			}

			if ext == ".ipa" {
				if exportedIPA != "" {
					log.Warn("More than 1 ipa file found")
				}
				exportedIPA = deployPth
			}
		}

		ipaPath = exportedIPA
	}

	fmt.Println()

	log.Info("Exporting outputs...")

	//
	// Export outputs

	// Export *.ipa path
	if err := exportEnvironmentWithEnvman("BITRISE_IPA_PATH", ipaPath); err != nil {
		fail("Failed to export ipa path, error: %s", err)
	}
	fmt.Println()
	log.Done("The IPA path is now available in the Environment Variable: $BITRISE_IPA_PATH (value: %s)", ipaPath)

	// Export app directory
	exportedApp := ""

	pattern := filepath.Join(archivePath, "Products/Applications", "*.app")
	files, err := filepath.Glob(pattern)
	if err != nil {
		fail("Failed to find .app directories, error: %s", err)
	}

	for _, file := range files {
		base := filepath.Base(file)
		deployPth := filepath.Join(configs.OutputDir, base)

		if err := os.Rename(file, deployPth); err != nil {
			fail("Failed to move (%s) -> (%s), error: %s", file, deployPth, err)
		}

		if exportedApp != "" {
			log.Warn("More than 1 .app directory found")
		}
		exportedApp = deployPth
	}

	if err := exportEnvironmentWithEnvman("BITRISE_APP_DIR_PATH", exportedApp); err != nil {
		fail("Failed to export .app path, error: %s", err)
	}
	fmt.Println()
	log.Done("The .app directory is now available in the Environment Variable: $BITRISE_APP_DIR_PATH (value: %s)", exportedApp)

	// dSYM handling
	appDSYM, frameworkDSYMs, err := xcarchive.ExportDSYMs(archivePath)
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

	if err := zip(dsymDir, dsymZipPath); err != nil {
		fail("zip failed, error: %s", err)
	}
	if err := exportEnvironmentWithEnvman("BITRISE_DSYM_PATH", dsymZipPath); err != nil {
		fail("Failed to export dsym path, error: %s", err)
	}
	fmt.Println()
	log.Done("The dSYM path is now available in the Environment Variable: $BITRISE_DSYM_PATH (value: %s)", dsymZipPath)
}
