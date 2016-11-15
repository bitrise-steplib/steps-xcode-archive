package xcodebuild

import (
	"io"
	"os"
	"os/exec"

	"bytes"

	"github.com/bitrise-io/go-utils/cmdex"
)

// SetExportFormat ...
func (xb *Model) SetExportFormat(exportFormat string) *Model {
	xb.exportFormat = exportFormat
	return xb
}

// SetExportPath ...
func (xb *Model) SetExportPath(exportPath string) *Model {
	xb.exportPath = exportPath
	return xb
}

// SetExportProvisioningProfile ...
func (xb *Model) SetExportProvisioningProfile(exportProvisioningProfile string) *Model {
	xb.exportProvisioningProfile = exportProvisioningProfile
	return xb
}

// SetExportOptionsPlist ...
func (xb *Model) SetExportOptionsPlist(exportOptionsPlist string) *Model {
	xb.exportOptionsPlist = exportOptionsPlist
	return xb
}

func (xb Model) legacyExportCmdSlice() []string {
	slice := []string{toolName}
	slice = append(slice, "-exportArchive")
	if xb.exportFormat != "" {
		slice = append(slice, "-exportFormat", xb.exportFormat)
	}
	if xb.archivePath != "" {
		slice = append(slice, "-archivePath", xb.archivePath)
	}
	if xb.exportPath != "" {
		slice = append(slice, "-exportPath", xb.exportPath)
	}
	if xb.exportProvisioningProfile != "" {
		slice = append(slice, "-exportProvisioningProfile", xb.exportProvisioningProfile)
	}
	return slice
}

func (xb Model) exportCmdSlice() []string {
	slice := []string{toolName}
	slice = append(slice, "-exportArchive")
	if xb.archivePath != "" {
		slice = append(slice, "-archivePath", xb.archivePath)
	}
	if xb.exportPath != "" {
		slice = append(slice, "-exportPath", xb.exportPath)
	}
	if xb.exportOptionsPlist != "" {
		slice = append(slice, "-exportOptionsPlist", xb.exportOptionsPlist)
	}
	return slice
}

// PrintableLegacyExportCmd ...
func (xb Model) PrintableLegacyExportCmd() string {
	cmdSlice := xb.legacyExportCmdSlice()
	return cmdex.PrintableCommandArgs(false, cmdSlice)
}

// PrintableExportCmd ...
func (xb Model) PrintableExportCmd() string {
	cmdSlice := xb.exportCmdSlice()
	return cmdex.PrintableCommandArgs(false, cmdSlice)
}

// LegacyExportCmd ...
func (xb Model) LegacyExportCmd() (*exec.Cmd, error) {
	cmdSlice := xb.legacyExportCmdSlice()
	cmd, err := cmdex.NewCommandFromSlice(cmdSlice)
	if err != nil {
		return nil, err
	}
	return cmd.GetCmd(), nil
}

// ExportCmd ...
func (xb Model) ExportCmd() (*exec.Cmd, error) {
	cmdSlice := xb.exportCmdSlice()
	cmd, err := cmdex.NewCommandFromSlice(cmdSlice)
	if err != nil {
		return nil, err
	}
	return cmd.GetCmd(), nil
}

// LegacyExport ...
func (xb Model) LegacyExport() error {
	cmdSlice := xb.legacyExportCmdSlice()
	cmd, err := cmdex.NewCommandFromSlice(cmdSlice)
	if err != nil {
		return err
	}

	cmd.SetStdout(os.Stdout)
	cmd.SetStderr(os.Stderr)

	return cmd.Run()
}

// Export ...
func (xb Model) Export() (string, error) {
	cmdSlice := xb.exportCmdSlice()
	cmd, err := cmdex.NewCommandFromSlice(cmdSlice)
	if err != nil {
		return "", err
	}

	var outBuffer bytes.Buffer
	outWriter := io.MultiWriter(&outBuffer, os.Stdout)

	cmd.SetStdout(outWriter)
	cmd.SetStderr(outWriter)

	return outBuffer.String(), cmd.Run()
}
