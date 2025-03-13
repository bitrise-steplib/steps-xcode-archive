package xcodecommand

import (
	"fmt"

	"github.com/bitrise-io/go-steputils/v2/ruby"
	command "github.com/bitrise-io/go-utils/v2/command"
	version "github.com/hashicorp/go-version"
)

type xcprettyManager interface {
	isDepInstalled() (bool, error)
	installDep() []command.Command
	depVersion() (*version.Version, error)
}

type xcpretty struct {
	commandFactory     command.Factory
	rubyEnv            ruby.Environment
	rubyCommandFactory ruby.CommandFactory
}

// CheckInstall checks if xcpretty is isntalled, if not installs it.
// Returns its version.
func (c *XcprettyCommandRunner) CheckInstall() (*version.Version, error) {
	c.logger.Println()
	c.logger.Infof("Checking if log formatter (xcpretty) is installed")

	installed, err := c.xcpretty.isDepInstalled()
	if err != nil {
		return nil, err
	} else if !installed {
		c.logger.Warnf(`xcpretty is not installed`)
		fmt.Println()
		c.logger.Printf("Installing xcpretty")

		cmdModelSlice := c.xcpretty.installDep()
		for _, cmd := range cmdModelSlice {
			if err := cmd.Run(); err != nil {
				return nil, fmt.Errorf("failed to run xcpretty install command (%s): %w", cmd.PrintableCommandArgs(), err)
			}
		}
	}

	xcprettyVersion, err := c.xcpretty.depVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get xcpretty version: %w", err)
	}

	return xcprettyVersion, nil
}

func (c *xcpretty) isDepInstalled() (bool, error) {
	return c.rubyEnv.IsGemInstalled("xcpretty", "")
}

func (c *xcpretty) installDep() []command.Command {
	cmds := c.rubyCommandFactory.CreateGemInstall("xcpretty", "", false, false, nil)
	return cmds
}

func (c *xcpretty) depVersion() (*version.Version, error) {
	cmd := c.commandFactory.Create("xcpretty", []string{"--version"}, nil)

	versionOut, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return nil, err
	}

	return version.NewVersion(versionOut)
}
