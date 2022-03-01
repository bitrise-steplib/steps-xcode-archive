package xcodebuild

import (
	"path/filepath"

	"github.com/bitrise-io/go-utils/command"
)

type ResolvePackagesCommandModel struct {
	projectPath string

	customOptions []string
}

func NewResolvePackagesCommandModel(projectPath string) *ResolvePackagesCommandModel {
	return &ResolvePackagesCommandModel{
		projectPath: projectPath,
	}
}

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

func (m *ResolvePackagesCommandModel) Command() command.Model {
	cmdSlice := m.cmdSlice()
	return *command.NewWithStandardOuts(cmdSlice[0], cmdSlice[1:]...)
}
