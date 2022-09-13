package step

import (
	"bufio"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/bitrise-steplib/steps-xcode-archive/nserror"
)

// XCPrettyInstallError is used to signal an error around xcpretty installation
type XCPrettyInstallError struct {
	err error
}

func (e XCPrettyInstallError) Error() string {
	return e.err.Error()
}

type Printable interface {
	PrintableCmd() string
}

func wrapXcodebuildCommandError(cmd Printable, out string, err error) error {
	if err == nil {
		return nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		reasons := findXcodebuildErrors(out)
		if len(reasons) > 0 {
			return fmt.Errorf("command (%s) failed with exit status %d: %w", cmd.PrintableCmd(), exitErr.ExitCode(), errors.New(strings.Join(reasons, "\n")))
		}
		return fmt.Errorf("command (%s) failed with exit status %d", cmd.PrintableCmd(), exitErr.ExitCode())
	}

	return fmt.Errorf("executing command (%s) failed: %w", cmd.PrintableCmd(), err)
}

func findXcodebuildErrors(out string) []string {
	var errorLines []string
	var nserrors []nserror.Error

	scanner := bufio.NewScanner(strings.NewReader(out))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "error: ") {
			errorLines = append(errorLines, line)
		} else if strings.HasPrefix(line, "Error ") {
			e := nserror.New(line)
			if e != nil {
				nserrors = append(nserrors, *e)
			}
		}

	}
	if err := scanner.Err(); err != nil {
		return nil
	}

	if len(nserrors) == len(errorLines) {
		errorLines = []string{}
		for _, nserror := range nserrors {
			errorLines = append(errorLines, nserror.Error())
		}
	}

	return errorLines
}
