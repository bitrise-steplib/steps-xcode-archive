package command

import (
	"io"
	"os/exec"
	"strconv"
	"strings"

	"github.com/bitrise-io/go-utils/v2/env"
)

// Opts ...
type Opts struct {
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
	Env    []string
	Dir    string
}

// Factory ...
type Factory interface {
	Create(name string, args []string, opts *Opts) Command
}

type factory struct {
	envRepository env.Repository
}

// NewFactory ...
func NewFactory(envRepository env.Repository) Factory {
	return factory{envRepository: envRepository}
}

// Create ...
func (f factory) Create(name string, args []string, opts *Opts) Command {
	cmd := exec.Command(name, args...)
	if opts != nil {
		cmd.Stdout = opts.Stdout
		cmd.Stderr = opts.Stderr
		cmd.Stdin = opts.Stdin

		// If Env is nil, the new process uses the current process's
		// environment.
		// If we pass env vars we want to append them to the
		// current process's environment.
		cmd.Env = append(f.envRepository.List(), opts.Env...)
		cmd.Dir = opts.Dir
	}
	return command{cmd}
}

// Command ...
type Command interface {
	PrintableCommandArgs() string
	Run() error
	RunAndReturnExitCode() (int, error)
	RunAndReturnTrimmedOutput() (string, error)
	RunAndReturnTrimmedCombinedOutput() (string, error)
	Start() error
	Wait() error
}

type command struct {
	cmd *exec.Cmd
}

// PrintableCommandArgs ...
func (c command) PrintableCommandArgs() string {
	return printableCommandArgs(false, c.cmd.Args)
}

// Run ...
func (c command) Run() error {
	return c.cmd.Run()
}

// RunAndReturnExitCode ...
func (c command) RunAndReturnExitCode() (int, error) {
	err := c.cmd.Run()
	exitCode := c.cmd.ProcessState.ExitCode()
	return exitCode, err
}

// RunAndReturnTrimmedOutput ...
func (c command) RunAndReturnTrimmedOutput() (string, error) {
	outBytes, err := c.cmd.Output()
	outStr := string(outBytes)
	return strings.TrimSpace(outStr), err
}

// RunAndReturnTrimmedCombinedOutput ...
func (c command) RunAndReturnTrimmedCombinedOutput() (string, error) {
	outBytes, err := c.cmd.CombinedOutput()
	outStr := string(outBytes)
	return strings.TrimSpace(outStr), err
}

// Start ...
func (c command) Start() error {
	return c.cmd.Start()
}

// Wait ...
func (c command) Wait() error {
	return c.cmd.Wait()
}

func printableCommandArgs(isQuoteFirst bool, fullCommandArgs []string) string {
	var cmdArgsDecorated []string
	for idx, anArg := range fullCommandArgs {
		quotedArg := strconv.Quote(anArg)
		if idx == 0 && !isQuoteFirst {
			quotedArg = anArg
		}
		cmdArgsDecorated = append(cmdArgsDecorated, quotedArg)
	}

	return strings.Join(cmdArgsDecorated, " ")
}
