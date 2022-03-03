package xcodebuild

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/log"
)

// ResolvePackagesCommandModel is a command builder
// used to create `xcodebuild -resolvePackageDependencies` command
type ResolvePackagesCommandModel struct {
	projectPath string

	customOptions []string
}

// NewResolvePackagesCommandModel returns a new ResolvePackagesCommandModel
func NewResolvePackagesCommandModel(projectPath string) *ResolvePackagesCommandModel {
	return &ResolvePackagesCommandModel{
		projectPath: projectPath,
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

	slice = append(slice, "-resolvePackageDependencies")
	slice = append(slice, m.customOptions...)

	return slice
}

// Command returns the executable command
func (m *ResolvePackagesCommandModel) Command() command.Model {
	cmdSlice := m.cmdSlice()
	return *command.NewWithStandardOuts(cmdSlice[0], cmdSlice[1:]...)
}

func (m *ResolvePackagesCommandModel) Run() error {
	var (
		cmd   = m.Command()
		start = time.Now()
	)

	log.Printf("Resolving package dependencies...")

	log.TDonef("$ %s", cmd.PrintableCommandArgs())
	if err := cmd.Run(); err != nil {
		if errorutil.IsExitStatusError(err) {
			return fmt.Errorf("failed to resolve package dependencies")
		}
		return fmt.Errorf("failed to run command: %s", err)
	}

	log.Printf("Resolved package dependencies in %s.", time.Since(start).Round(time.Second))

	return nil
}
