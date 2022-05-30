package xcodebuild

import (
	"fmt"
	"path/filepath"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/log"
)

// ResolvePackagesCommandModel is a command builder
// used to create `xcodebuild -resolvePackageDependencies` command
type ResolvePackagesCommandModel struct {
	projectPath   string
	scheme        string
	configuration string

	customOptions []string
}

// NewResolvePackagesCommandModel returns a new ResolvePackagesCommandModel
func NewResolvePackagesCommandModel(projectPath, scheme, configuration string) *ResolvePackagesCommandModel {
	return &ResolvePackagesCommandModel{
		projectPath:   projectPath,
		scheme:        scheme,
		configuration: configuration,
	}
}

// SetCustomOptions sets custom options
func (m *ResolvePackagesCommandModel) SetCustomOptions(customOptions []string) *ResolvePackagesCommandModel {
	m.customOptions = customOptions
	return m
}

func (m *ResolvePackagesCommandModel) cmdSlice() []string {
	slice := []string{toolName}

	if m.projectPath != "" {
		if filepath.Ext(m.projectPath) == ".xcworkspace" {
			slice = append(slice, "-workspace", m.projectPath)
		} else {
			slice = append(slice, "-project", m.projectPath)
		}
	}

	if m.scheme != "" {
		slice = append(slice, "-scheme", m.scheme)
	}

	if m.configuration != "" {
		slice = append(slice, "-configuration", m.configuration)
	}

	slice = append(slice, "-resolvePackageDependencies")
	slice = append(slice, m.customOptions...)

	return slice
}

// Command returns the executable command
func (m *ResolvePackagesCommandModel) command() command.Model {
	cmdSlice := m.cmdSlice()
	return *command.NewWithStandardOuts(cmdSlice[0], cmdSlice[1:]...)
}

// Run runs the command and logs elapsed time
func (m *ResolvePackagesCommandModel) Run() error {
	var cmd = m.command()

	log.TPrintf("Resolving package dependencies...")

	log.TDonef("$ %s", cmd.PrintableCommandArgs())
	if err := cmd.Run(); err != nil {
		if errorutil.IsExitStatusError(err) {
			return fmt.Errorf("failed to resolve package dependencies")
		}
		return fmt.Errorf("failed to run command: %s", err)
	}

	log.TPrintf("Resolved package dependencies.")

	return nil
}
