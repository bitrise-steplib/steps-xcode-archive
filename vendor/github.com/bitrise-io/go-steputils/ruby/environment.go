package ruby

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/env"
	"github.com/bitrise-io/go-utils/pathutil"
)

const (
	systemRubyPth  = "/usr/bin/ruby"
	brewRubyPth    = "/usr/local/bin/ruby"
	brewRubyPthAlt = "/usr/local/opt/ruby/bin/ruby"
)

// InstallType ...
type InstallType int8

const (
	// Unknown ...
	Unknown InstallType = iota
	// SystemRuby ...
	SystemRuby
	// BrewRuby ...
	BrewRuby
	// RVMRuby ...
	RVMRuby
	// RbenvRuby ...
	RbenvRuby
)

// Environment ...
type Environment interface {
	RubyInstallType() InstallType
	IsGemInstalled(gem, version string) (bool, error)
	IsSpecifiedRbenvRubyInstalled(workdir string) (bool, string, error)
}

type environment struct {
	factory    CommandFactory
	cmdLocator env.CommandLocator
}

// NewEnvironment ...
func NewEnvironment(factory CommandFactory, cmdLocator env.CommandLocator) Environment {
	return environment{
		factory:    factory,
		cmdLocator: cmdLocator,
	}
}

// RubyInstallType returns which version manager was used for the ruby install
func (m environment) RubyInstallType() InstallType {
	return rubyInstallType(m.cmdLocator)
}

func (m environment) IsGemInstalled(gem, version string) (bool, error) {
	cmd := m.factory.Create("gem", []string{"list"}, nil)

	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return false, fmt.Errorf("%s: error: %s", out, err)
	}

	return findGemInList(out, gem, version)
}

// IsSpecifiedRbenvRubyInstalled checks if the selected ruby version is installed via rbenv.
// Ruby version is set by
// 1. The RBENV_VERSION environment variable
// 2. The first .ruby-version file found by searching the directory of the script you are executing and each of its
// parent directories until reaching the root of your filesystem.
// 3.The first .ruby-version file found by searching the current working directory and each of its parent directories
// until reaching the root of your filesystem.
// 4. The global ~/.rbenv/version file. You can modify this file using the rbenv global command.
// src: https://github.com/rbenv/rbenv#choosing-the-ruby-version
func (m environment) IsSpecifiedRbenvRubyInstalled(workdir string) (bool, string, error) {
	absWorkdir, err := pathutil.AbsPath(workdir)
	if err != nil {
		return false, "", fmt.Errorf("failed to get absolute path for ( %s ), error: %s", workdir, err)
	}

	cmd := m.factory.Create("rbenv", []string{"version"}, &command.Opts{Dir: absWorkdir})
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return false, "", fmt.Errorf("failed to check installed ruby version, %s error: %s", out, err)
	}
	return isSpecifiedRbenvRubyInstalled(out)
}

func rubyInstallType(cmdLocator env.CommandLocator) InstallType {
	pth, err := cmdLocator.LookPath("ruby")
	if err != nil {
		return Unknown
	}

	installType := Unknown
	if pth == systemRubyPth {
		installType = SystemRuby
	} else if pth == brewRubyPth {
		installType = BrewRuby
	} else if pth == brewRubyPthAlt {
		installType = BrewRuby
	} else if _, err := cmdLocator.LookPath("rvm"); err == nil {
		installType = RVMRuby
	} else if _, err := cmdLocator.LookPath("rbenv"); err == nil {
		installType = RbenvRuby
	}

	return installType
}

func isSpecifiedRbenvRubyInstalled(message string) (bool, string, error) {
	//
	// Not installed
	reg, err := regexp.Compile("rbenv: version \x60.*' is not installed") // \x60 == ` (The go linter suggested to use the hex code instead)
	if err != nil {
		return false, "", fmt.Errorf("failed to parse regex ( %s ) on the error message, error: %s", "rbenv: version \x60.*' is not installed", err) // \x60 == ` (The go linter suggested to use the hex code instead)
	}

	var version string
	if reg.MatchString(message) {
		message := reg.FindString(message)
		version = strings.Split(strings.Split(message, "`")[1], "'")[0]
		return false, version, nil
	}

	//
	// Installed
	reg, err = regexp.Compile(".* \\(set by")
	if err != nil {
		return false, "", fmt.Errorf("failed to parse regex ( %s ) on the error message, error: %s", ".* \\(set by", err)
	}

	if reg.MatchString(message) {
		s := reg.FindString(message)
		version = strings.Split(s, " (set by")[0]
		return true, version, nil
	}
	return false, version, nil
}

func findGemInList(gemList, gem, version string) (bool, error) {
	// minitest (5.10.1, 5.9.1, 5.9.0, 5.8.3, 4.7.5)
	pattern := fmt.Sprintf(`^%s \(.*%s.*\)`, gem, version)
	re := regexp.MustCompile(pattern)

	reader := bytes.NewReader([]byte(gemList))
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		match := re.FindString(line)
		if match != "" {
			return true, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	return false, nil
}
