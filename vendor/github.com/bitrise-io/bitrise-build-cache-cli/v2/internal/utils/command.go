package utils

import (
	"context"
	"os"
	"os/exec"
	"syscall"
)

//go:generate moq -stub -out mocks/command_mock.go -pkg mocks . Command
type Command interface {
	Start() error
	Wait() error
	Err() error
	CombinedOutput() ([]byte, error)
	SetStdout(file *os.File)
	SetStderr(file *os.File)
	SetStdin(file *os.File)
	SetSysProcAttr(sysProcAttr *syscall.SysProcAttr)
	PID() int
}

type CommandWrapper struct {
	Wrapped *exec.Cmd
}

func (cmd CommandWrapper) SetStdout(file *os.File) {
	cmd.Wrapped.Stdout = file
}

func (cmd CommandWrapper) SetStderr(file *os.File) {
	cmd.Wrapped.Stderr = file
}

func (cmd CommandWrapper) SetStdin(file *os.File) {
	cmd.Wrapped.Stdin = file
}

func (cmd CommandWrapper) SetSysProcAttr(sysProcAttr *syscall.SysProcAttr) {
	cmd.Wrapped.SysProcAttr = sysProcAttr
}

func (cmd CommandWrapper) CombinedOutput() ([]byte, error) {
	return cmd.Wrapped.CombinedOutput() //nolint:wrapcheck
}

func (cmd CommandWrapper) Wait() error {
	return cmd.Wrapped.Wait() //nolint:wrapcheck
}

func (cmd CommandWrapper) Err() error {
	return cmd.Wrapped.Err //nolint:wrapcheck
}

func (cmd CommandWrapper) PID() int {
	if cmd.Wrapped.Process == nil {
		return 0
	}

	return cmd.Wrapped.Process.Pid
}

func (cmd CommandWrapper) Start() error {
	return cmd.Wrapped.Start() //nolint:wrapcheck
}

type CommandFunc func(ctx context.Context, command string, args ...string) Command

func DefaultCommandFunc() CommandFunc {
	return func(ctx context.Context, command string, args ...string) Command {
		return CommandWrapper{Wrapped: exec.CommandContext(ctx, command, args...)}
	}
}
