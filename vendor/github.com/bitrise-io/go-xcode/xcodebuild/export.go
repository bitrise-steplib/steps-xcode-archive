package xcodebuild

import (
	"bytes"
	"io"
	"os"

	"github.com/bitrise-io/go-utils/command"
)

/*
xcodebuild -exportArchive \
	-archivePath <xcarchivepath> \
	-exportPath <destinationpath> \
	-exportOptionsPlist <plistpath>
*/

// ExportCommandModel ...
type ExportCommandModel struct {
	commandFactory command.Factory

	archivePath        string
	exportDir          string
	exportOptionsPlist string
	authentication     *AuthenticationParams
}

// NewExportCommand ...
func NewExportCommand(commandFactory command.Factory) *ExportCommandModel {
	return &ExportCommandModel{
		commandFactory: commandFactory,
	}
}

// SetArchivePath ...
func (c *ExportCommandModel) SetArchivePath(archivePath string) *ExportCommandModel {
	c.archivePath = archivePath
	return c
}

// SetExportDir ...
func (c *ExportCommandModel) SetExportDir(exportDir string) *ExportCommandModel {
	c.exportDir = exportDir
	return c
}

// SetExportOptionsPlist ...
func (c *ExportCommandModel) SetExportOptionsPlist(exportOptionsPlist string) *ExportCommandModel {
	c.exportOptionsPlist = exportOptionsPlist
	return c
}

// SetAuthentication ...
func (c *ExportCommandModel) SetAuthentication(authenticationParams AuthenticationParams) *ExportCommandModel {
	c.authentication = &authenticationParams
	return c
}

func (c ExportCommandModel) args() []string {
	slice := []string{"-exportArchive"}
	if c.archivePath != "" {
		slice = append(slice, "-archivePath", c.archivePath)
	}

	if c.exportDir != "" {
		slice = append(slice, "-exportPath", c.exportDir)
	}

	if c.exportOptionsPlist != "" {
		slice = append(slice, "-exportOptionsPlist", c.exportOptionsPlist)
	}

	if c.authentication != nil {
		slice = append(slice, c.authentication.args()...)
	}

	return slice
}

// Command ...
func (c ExportCommandModel) Command(opts *command.Opts) command.Command {
	return c.commandFactory.Create(toolName, c.args(), opts)
}

// PrintableCmd ...
func (c ExportCommandModel) PrintableCmd() string {
	return c.Command(nil).PrintableCommandArgs()
}

// Run ...
func (c ExportCommandModel) Run() error {
	command := c.Command(&command.Opts{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})

	return command.Run()
}

// RunAndReturnOutput ...
func (c ExportCommandModel) RunAndReturnOutput() (string, error) {
	var outBuffer bytes.Buffer
	outWriter := io.MultiWriter(&outBuffer, os.Stdout)

	command := c.Command(&command.Opts{
		Stdout: outWriter,
		Stderr: outWriter,
	})

	err := command.Run()
	out := outBuffer.String()

	return out, err
}
