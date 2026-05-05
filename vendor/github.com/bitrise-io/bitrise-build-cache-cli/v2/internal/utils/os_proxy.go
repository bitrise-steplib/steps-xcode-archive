package utils

import (
	"fmt"
	"os"

	"github.com/bitrise-io/go-utils/v2/pathutil"
)

//go:generate moq -out mocks/os_proxy_mock.go -pkg mocks . OsProxy
//nolint:interfacebloat
type OsProxy interface {
	Create(name string) (*os.File, error)
	Executable() (string, error)
	FindProcess(pid int) (*os.Process, error)
	Getwd() (string, error)
	Hostname() (string, error)
	MkdirAll(name string, mode os.FileMode) error
	OpenFile(name string, flag int, perm os.FileMode) (*os.File, error)
	ReadFileIfExists(name string) (string, bool, error)
	Remove(name string) error
	Stat(pth string) (os.FileInfo, error)
	TempDir() string
	UserHomeDir() (string, error)
	WriteFile(name string, data []byte, mode os.FileMode) error
}

type DefaultOsProxy struct{}

func (d DefaultOsProxy) ReadFileIfExists(pth string) (string, bool, error) {
	if exists, err := pathutil.NewPathChecker().IsPathExists(pth); err != nil {
		return "", false, fmt.Errorf("failed to check if path (%s) exists: %w", pth, err)
	} else if !exists {
		return "", false, nil
	}

	content, err := os.ReadFile(pth)
	if err != nil {
		return "", true, fmt.Errorf("failed to read file: %s, error: %w", pth, err)
	}

	return string(content), true, nil
}

// Intentionally passing errors back unwrapped

func (d DefaultOsProxy) MkdirAll(pth string, perm os.FileMode) error {
	return os.MkdirAll(pth, perm) //nolint:wrapcheck
}

func (d DefaultOsProxy) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm) //nolint:wrapcheck
}

func (d DefaultOsProxy) UserHomeDir() (string, error) {
	return os.UserHomeDir() //nolint:wrapcheck
}

func (d DefaultOsProxy) Create(name string) (*os.File, error) {
	return os.Create(name) //nolint:wrapcheck
}

func (d DefaultOsProxy) TempDir() string {
	return os.TempDir()
}

func (d DefaultOsProxy) Remove(name string) error {
	return os.Remove(name) //nolint:wrapcheck
}

func (d DefaultOsProxy) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm) //nolint:wrapcheck
}

func (d DefaultOsProxy) Executable() (string, error) {
	return os.Executable() //nolint:wrapcheck
}

func (d DefaultOsProxy) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name) //nolint:wrapcheck
}

func (d DefaultOsProxy) Getwd() (string, error) {
	return os.Getwd() //nolint:wrapcheck
}

func (d DefaultOsProxy) Hostname() (string, error) {
	return os.Hostname() //nolint:wrapcheck
}

func (d DefaultOsProxy) FindProcess(pid int) (*os.Process, error) {
	return os.FindProcess(pid) //nolint:wrapcheck
}
