package xcarchive

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-tools/go-xcode/exportoptions"
	"github.com/bitrise-tools/go-xcode/provisioningprofile"
)

// CommandCallback ...
type CommandCallback func(printableCommand string)

// ExportFormat ...
type ExportFormat string

const (
	// ExportFormatUnknown ...
	ExportFormatUnknown ExportFormat = "unkown"
	// ExportFormatIPA ...
	ExportFormatIPA ExportFormat = "ipa"
	// ExportFormatAPP ...
	ExportFormatAPP ExportFormat = "app"
	// ExportFormatPKG ...
	ExportFormatPKG ExportFormat = "pkg"
)

// ParseExportFormat ...
func ParseExportFormat(format string) (ExportFormat, error) {
	switch format {
	case "ipa":
		return ExportFormatIPA, nil
	case "app":
		return ExportFormatAPP, nil
	case "pkg":
		return ExportFormatPKG, nil
	default:
		return ExportFormatUnknown, fmt.Errorf("Unknown export format (%s)", format)
	}
}

// Ext ...
func (exportFormat ExportFormat) Ext() string {
	switch exportFormat {
	case ExportFormatIPA:
		return ".ipa"
	case ExportFormatAPP:
		return ".app"
	case ExportFormatPKG:
		return ".pkg"
	default:
		return ""
	}
}

// String ...
func (exportFormat ExportFormat) String() string {
	switch exportFormat {
	case ExportFormatIPA:
		return "ipa"
	case ExportFormatAPP:
		return "app"
	case ExportFormatPKG:
		return "pkg"
	default:
		return ""
	}
}

// EmbeddedMobileProvisionPth ...
func EmbeddedMobileProvisionPth(archivePth string) (string, error) {
	applicationPth := filepath.Join(archivePth, "/Products/Applications")
	mobileProvisionPthPattern := filepath.Join(applicationPth, "*.app/embedded.mobileprovision")
	mobileProvisionPths, err := filepath.Glob(mobileProvisionPthPattern)
	if err != nil {
		return "", fmt.Errorf("failed to find embedded.mobileprovision with pattern: %s, error: %s", mobileProvisionPthPattern, err)
	}
	if len(mobileProvisionPths) == 0 {
		return "", fmt.Errorf("no embedded.mobileprovision with pattern: %s", mobileProvisionPthPattern)
	}
	return mobileProvisionPths[0], nil
}

// DefaultExportOptions ...
func DefaultExportOptions(provProfile provisioningprofile.Model) (exportoptions.ExportOptions, error) {
	method := provProfile.GetExportMethod()
	developerTeamID := provProfile.GetDeveloperTeam()

	if method == exportoptions.MethodAppStore {
		options := exportoptions.NewAppStoreOptions()
		options.TeamID = developerTeamID
		return options, nil
	}

	options := exportoptions.NewNonAppStoreOptions(method)
	options.TeamID = developerTeamID
	return options, nil
}

// Export ...
func Export(archivePth, exportOptionsPth string, callback CommandCallback) (string, error) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("output")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir, error: %s", err)
	}

	cmdSlice := []string{
		"xcodebuild", "-exportArchive",
		"-archivePath", archivePth,
		"-exportOptionsPlist", exportOptionsPth,
		"-exportPath", tmpDir,
	}

	if callback != nil {
		callback(cmdex.PrintableCommandArgs(false, cmdSlice))
	}

	cmd, err := cmdex.NewCommandFromSlice(cmdSlice)
	if err != nil {
		return "", fmt.Errorf("failed to create command from (%s)", strings.Join(cmdSlice, " "))
	}

	cmd.SetStdin(os.Stdin)
	cmd.SetStderr(os.Stderr)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("export command failed, error: %s", err)
	}

	pattern := filepath.Join(tmpDir, "*")
	matches, err := filepath.Glob(pattern)
	validOutputs := []string{}
	for _, pth := range matches {
		ext := filepath.Ext(pth)
		if ext == ".ipa" || ext == ".app" || ext == ".pkg" {
			validOutputs = append(validOutputs, pth)
		}
	}
	if len(validOutputs) == 0 {
		return "", errors.New("no output (.ipa/.app/.pkg) found")
	}

	return validOutputs[0], nil
}

// LegacyExport ...
func LegacyExport(archivePth, provisioningProfileName string, exportFormat ExportFormat, callback CommandCallback) (string, error) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("output")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir, error: %s", err)
	}

	outputName := strings.TrimSuffix(filepath.Base(archivePth), filepath.Ext(archivePth))
	outputExt := exportFormat.Ext()
	outputPth := filepath.Join(tmpDir, outputName+outputExt)

	cmdSlice := []string{
		"xcodebuild", "-exportArchive",
		"-archivePath", archivePth,
		"-exportFormat", exportFormat.String(),
		"-exportPath", outputPth,
	}

	if provisioningProfileName != "" {
		cmdSlice = append(cmdSlice, "-exportProvisioningProfile", provisioningProfileName)
	}

	if callback != nil {
		callback(cmdex.PrintableCommandArgs(false, cmdSlice))
	}

	cmd, err := cmdex.NewCommandFromSlice(cmdSlice)
	if err != nil {
		return "", fmt.Errorf("failed to create command from (%s)", strings.Join(cmdSlice, " "))
	}

	cmd.SetStdin(os.Stdin)
	cmd.SetStderr(os.Stderr)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("export command failed, error: %s", err)
	}

	return outputPth, nil
}

// ExportDSYMs ...
func ExportDSYMs(archivePth string) (string, []string, error) {
	dsymsPattern := filepath.Join(archivePth, "dSYMs", "*.dSYM")
	dsyms, err := filepath.Glob(dsymsPattern)
	if err != nil {
		return "", []string{}, fmt.Errorf("failed to find dSYM with pattern: %s, error: %s", dsymsPattern, err)
	}
	appDSYM := ""
	frameworkDSYMs := []string{}
	for _, dsym := range dsyms {
		if strings.HasSuffix(dsym, ".app.dSYM") {
			appDSYM = dsym
		} else {
			frameworkDSYMs = append(frameworkDSYMs, dsym)
		}
	}
	return appDSYM, frameworkDSYMs, nil
}
