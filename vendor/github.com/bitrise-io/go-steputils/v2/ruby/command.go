package ruby

import (
	"errors"
	"strings"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
)

// CommandFactory ...
type CommandFactory interface {
	Create(name string, args []string, opts *command.Opts) command.Command
	CreateBundleExec(name string, args []string, bundlerVersion string, opts *command.Opts) command.Command
	CreateBundleInstall(bundlerVersion string, opts *command.Opts) command.Command
	CreateGemInstall(gem, version string, enablePrerelease, force bool, opts *command.Opts) []command.Command
	CreateGemUpdate(gem string, opts *command.Opts) []command.Command
}

type commandFactory struct {
	cmdFactory  command.Factory
	installType InstallType
}

// NewCommandFactory ...
func NewCommandFactory(cmdFactory command.Factory, cmdLocator env.CommandLocator) (CommandFactory, error) {
	installType := rubyInstallType(cmdLocator)
	if installType == Unknown {
		return nil, errors.New("unknown Ruby installation")
	}

	return commandFactory{
		cmdFactory:  cmdFactory,
		installType: installType,
	}, nil
}

// Create ...
func (f commandFactory) Create(name string, args []string, opts *command.Opts) command.Command {
	s := append([]string{name}, args...)
	if sudoNeeded(f.installType, s...) {
		return f.cmdFactory.Create("sudo", s, opts)
	}
	return f.cmdFactory.Create(name, args, opts)
}

// CreateBundleExec ...
func (f commandFactory) CreateBundleExec(name string, args []string, bundlerVersion string, opts *command.Opts) command.Command {
	a := bundleCommandArgs(append([]string{"exec", name}, args...), bundlerVersion)
	return f.Create("bundle", a, opts)
}

// CreateBundleInstall returns a command to install a bundle using bundler
func (f commandFactory) CreateBundleInstall(bundlerVersion string, opts *command.Opts) command.Command {
	a := bundleCommandArgs([]string{"install", "--jobs", "20", "--retry", "5"}, bundlerVersion)
	return f.Create("bundle", a, opts)
}

// CreateGemInstall ...
func (f commandFactory) CreateGemInstall(gem, version string, enablePrerelease, force bool, opts *command.Opts) []command.Command {
	a := gemInstallCommandArgs(gem, version, enablePrerelease, force)
	cmd := f.Create("gem", a, opts)
	cmds := []command.Command{cmd}

	if f.installType == RbenvRuby {
		cmd := f.Create("rbenv", []string{"rehash"}, nil)
		cmds = append(cmds, cmd)
	} else if f.installType == ASDFRuby {
		cmd := f.Create("asdf", []string{"reshim", "ruby"}, nil)
		cmds = append(cmds, cmd)
	}

	return cmds
}

// CreateGemUpdate ...
func (f commandFactory) CreateGemUpdate(gem string, opts *command.Opts) []command.Command {
	cmd := f.Create("gem", []string{"update", gem, "--no-document"}, opts)
	cmds := []command.Command{cmd}

	if f.installType == RbenvRuby {
		cmd := f.Create("rbenv", []string{"rehash"}, nil)
		cmds = append(cmds, cmd)
	} else if f.installType == ASDFRuby {
		cmd := f.Create("asdf", []string{"reshim", "ruby"}, nil)
		cmds = append(cmds, cmd)
	}

	return cmds
}

func bundleCommandArgs(args []string, bundlerVersion string) []string {
	var a []string
	if bundlerVersion != "" {
		a = append(a, "_"+bundlerVersion+"_")
	}
	return append(a, args...)
}

func gemInstallCommandArgs(gem, version string, enablePrerelease, force bool) []string {
	slice := []string{"install", gem, "--no-document"}
	if enablePrerelease {
		slice = append(slice, "--prerelease")
	}
	if version != "" {
		slice = append(slice, "-v", version)
	}
	if force {
		slice = append(slice, "--force")
	}

	return slice
}

func sudoNeeded(installType InstallType, command ...string) bool {
	if installType != SystemRuby {
		return false
	}

	if len(command) < 2 {
		return false
	}

	name := command[0]
	if name == "bundle" {
		cmd := command[1]
		/*
			bundle command can contain version:
			`bundle _2.0.1_ install`
		*/
		const bundleVersionMarker = "_"
		if strings.HasPrefix(command[1], bundleVersionMarker) && strings.HasSuffix(command[1], bundleVersionMarker) {
			if len(command) < 3 {
				return false
			}
			cmd = command[2]
		}

		return cmd == "install" || cmd == "update"
	} else if name == "gem" {
		cmd := command[1]
		return cmd == "install" || cmd == "uninstall"
	}

	return false
}
