package xcodebuild

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/bitrise-io/go-utils/command"
)

const (
	// XCWorkspaceExtension ...
	XCWorkspaceExtension = ".xcworkspace"
	// XCProjExtension ...
	XCProjExtension = ".xcodeproj"
)

/*
xcodebuild [-project <projectname>] \
	-scheme <schemeName> \
	[-destination <destinationspecifier>]... \
	[-configuration <configurationname>] \
	[-arch <architecture>]... \
	[-sdk [<sdkname>|<sdkpath>]] \
	[-showBuildSettings] \
	[<buildsetting>=<value>]... \
	[<buildaction>]...
xcodebuild -workspace <workspacename> \
	-scheme <schemeName> \
	[-destination <destinationspecifier>]... \
	[-configuration <configurationname>] \
	[-arch <architecture>]... \
	[-sdk [<sdkname>|<sdkpath>]] \
	[-showBuildSettings] \
	[<buildsetting>=<value>]... \
	[<buildaction>]...
*/

// CommandBuilder ...
type CommandBuilder struct {
	actions []string

	// Options
	projectPath      string
	scheme           string
	configuration    string
	destination      string
	xcconfigPath     string
	authentication   *AuthenticationParams
	archivePath      string
	customOptions    []string
	sdk              string
	resultBundlePath string
	testPlan         string

	// buildsetting
	disableCodesign bool
}

// NewCommandBuilder ...
func NewCommandBuilder(projectPath string, actions ...string) *CommandBuilder {
	return &CommandBuilder{
		projectPath: projectPath,
		actions:     actions,
	}
}

// SetScheme ...
func (c *CommandBuilder) SetScheme(scheme string) *CommandBuilder {
	c.scheme = scheme
	return c
}

// SetConfiguration ...
func (c *CommandBuilder) SetConfiguration(configuration string) *CommandBuilder {
	c.configuration = configuration
	return c
}

// SetDestination ...
func (c *CommandBuilder) SetDestination(destination string) *CommandBuilder {
	c.destination = destination
	return c
}

// SetXCConfigPath ...
func (c *CommandBuilder) SetXCConfigPath(xcconfigPath string) *CommandBuilder {
	c.xcconfigPath = xcconfigPath
	return c
}

// SetAuthentication ...
func (c *CommandBuilder) SetAuthentication(authenticationParams AuthenticationParams) *CommandBuilder {
	c.authentication = &authenticationParams
	return c
}

// SetArchivePath ...
func (c *CommandBuilder) SetArchivePath(archivePath string) *CommandBuilder {
	c.archivePath = archivePath
	return c
}

// SetResultBundlePath ...
func (c *CommandBuilder) SetResultBundlePath(resultBundlePath string) *CommandBuilder {
	c.resultBundlePath = resultBundlePath
	return c
}

// SetCustomOptions ...
func (c *CommandBuilder) SetCustomOptions(customOptions []string) *CommandBuilder {
	c.customOptions = customOptions
	return c
}

// SetSDK ...
func (c *CommandBuilder) SetSDK(sdk string) *CommandBuilder {
	c.sdk = sdk
	return c
}

// SetDisableCodesign ...
func (c *CommandBuilder) SetDisableCodesign(disable bool) *CommandBuilder {
	c.disableCodesign = disable
	return c
}

// SetTestPlan ...
func (c *CommandBuilder) SetTestPlan(testPlan string) *CommandBuilder {
	c.testPlan = testPlan
	return c
}

func (c *CommandBuilder) cmdSlice() []string {
	slice := []string{toolName}
	slice = append(slice, c.actions...)

	if c.projectPath != "" {
		if filepath.Ext(c.projectPath) == XCWorkspaceExtension {
			slice = append(slice, "-workspace", c.projectPath)
		} else {
			slice = append(slice, "-project", c.projectPath)
		}
	}

	if c.scheme != "" {
		slice = append(slice, "-scheme", c.scheme)
	}

	if c.configuration != "" {
		slice = append(slice, "-configuration", c.configuration)
	}

	if c.destination != "" {
		// "-destination" "id=07933176-D03B-48D3-A853-0800707579E6" => (need the plus `"` marks between the `destination` and the `id`)
		slice = append(slice, "-destination", c.destination)
	}

	if c.xcconfigPath != "" {
		slice = append(slice, "-xcconfig", c.xcconfigPath)
	}

	if c.archivePath != "" {
		slice = append(slice, "-archivePath", c.archivePath)
	}

	if c.sdk != "" {
		slice = append(slice, "-sdk", c.sdk)
	}

	if c.resultBundlePath != "" {
		slice = append(slice, "-resultBundlePath", c.resultBundlePath)
	}

	if c.authentication != nil {
		slice = append(slice, c.authentication.args()...)
	}

	if c.testPlan != "" {
		slice = append(slice, "-testPlan", c.testPlan)
	}

	if c.disableCodesign {
		slice = append(slice, "CODE_SIGNING_ALLOWED=NO")
	}

	slice = append(slice, c.customOptions...)

	return slice
}

// PrintableCmd ...
func (c CommandBuilder) PrintableCmd() string {
	cmdSlice := c.cmdSlice()
	return command.PrintableCommandArgs(false, cmdSlice)
}

// Command ...
func (c CommandBuilder) Command() *command.Model {
	cmdSlice := c.cmdSlice()
	return command.New(cmdSlice[0], cmdSlice[1:]...)
}

// ExecCommand ...
func (c CommandBuilder) ExecCommand() *exec.Cmd {
	command := c.Command()
	return command.GetCmd()
}

// Run ...
func (c CommandBuilder) Run() error {
	command := c.Command()

	command.SetStdout(os.Stdout)
	command.SetStderr(os.Stderr)

	return command.Run()
}
