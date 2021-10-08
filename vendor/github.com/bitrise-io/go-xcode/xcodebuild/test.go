package xcodebuild

import (
	"os"

	"github.com/bitrise-io/go-utils/command"
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

// TestCommandModel ...
type TestCommandModel struct {
	commandFactory command.Factory

	projectPath string
	isWorkspace bool
	scheme      string
	destination string

	// buildsetting
	generateCodeCoverage      bool
	disableIndexWhileBuilding bool

	// buildaction
	customBuildActions []string // clean, build

	// Options
	customOptions []string
}

// NewTestCommand ...
func NewTestCommand(projectPath string, isWorkspace bool, commandFactory command.Factory) *TestCommandModel {
	return &TestCommandModel{
		commandFactory: commandFactory,
		projectPath:    projectPath,
		isWorkspace:    isWorkspace,
	}
}

// SetScheme ...
func (c *TestCommandModel) SetScheme(scheme string) *TestCommandModel {
	c.scheme = scheme
	return c
}

// SetDestination ...
func (c *TestCommandModel) SetDestination(destination string) *TestCommandModel {
	c.destination = destination
	return c
}

// SetGenerateCodeCoverage ...
func (c *TestCommandModel) SetGenerateCodeCoverage(generateCodeCoverage bool) *TestCommandModel {
	c.generateCodeCoverage = generateCodeCoverage
	return c
}

// SetCustomBuildAction ...
func (c *TestCommandModel) SetCustomBuildAction(buildAction ...string) *TestCommandModel {
	c.customBuildActions = buildAction
	return c
}

// SetCustomOptions ...
func (c *TestCommandModel) SetCustomOptions(customOptions []string) *TestCommandModel {
	c.customOptions = customOptions
	return c
}

// SetDisableIndexWhileBuilding ...
func (c *TestCommandModel) SetDisableIndexWhileBuilding(disable bool) *TestCommandModel {
	c.disableIndexWhileBuilding = disable
	return c
}

func (c *TestCommandModel) args() []string {
	var slice []string

	if c.projectPath != "" {
		if c.isWorkspace {
			slice = append(slice, "-workspace", c.projectPath)
		} else {
			slice = append(slice, "-project", c.projectPath)
		}
	}

	if c.scheme != "" {
		slice = append(slice, "-scheme", c.scheme)
	}

	if c.generateCodeCoverage {
		slice = append(slice, "GCC_INSTRUMENT_PROGRAM_FLOW_ARCS=YES", "GCC_GENERATE_TEST_COVERAGE_FILES=YES")
	}

	slice = append(slice, c.customBuildActions...)
	slice = append(slice, "test")
	if c.destination != "" {
		slice = append(slice, "-destination", c.destination)
	}

	if c.disableIndexWhileBuilding {
		slice = append(slice, "COMPILER_INDEX_STORE_ENABLE=NO")
	}

	slice = append(slice, c.customOptions...)

	return slice
}

// Command ...
func (c TestCommandModel) Command(opts *command.Opts) command.Command {
	return c.commandFactory.Create(toolName, c.args(), opts)
}

// PrintableCmd ...
func (c TestCommandModel) PrintableCmd() string {
	return c.Command(nil).PrintableCommandArgs()
}

// Run ...
func (c TestCommandModel) Run() error {
	command := c.Command(&command.Opts{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
	return command.Run()
}
