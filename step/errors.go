package step

import (
	"bufio"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// XCPrettyInstallError is used to signal an error around xcpretty installation
type XCPrettyInstallError struct {
	err error
}

func (e XCPrettyInstallError) Error() string {
	return e.err.Error()
}

type NSError struct {
	Description string
	Suggestion  string
}

func NewNSError(str string) *NSError {
	nserrorPattern := `Error Domain=.* Code=.*UserInfo=.*`
	exp := regexp.MustCompile(nserrorPattern)
	if !exp.MatchString(str) {
		return nil
	}

	descriptionPattern := `NSLocalizedDescription=(.+?),|NSLocalizedDescription=(.+?)}`
	description := findFirstSubMatch(str, descriptionPattern)
	if description == "" {
		return nil
	}

	suggestionPattern := `NSLocalizedRecoverySuggestion=(.+?),|NSLocalizedRecoverySuggestion=(.+?)}`
	suggestion := findFirstSubMatch(str, suggestionPattern)

	return &NSError{
		Description: description,
		Suggestion:  suggestion,
	}
}

func (e NSError) Error() string {
	msg := e.Description
	if e.Suggestion != "" {
		msg += " " + e.Suggestion
	}
	return msg
}

func findFirstSubMatch(str, pattern string) string {
	exp := regexp.MustCompile(pattern)
	matches := exp.FindStringSubmatch(str)
	if len(matches) > 1 {
		for _, match := range matches[1:] {
			if match != "" {
				return match
			}
		}
	}
	return ""
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
	var nserrors []NSError

	scanner := bufio.NewScanner(strings.NewReader(out))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "error: ") {
			errorLines = append(errorLines, line)
		} else if strings.HasPrefix(line, "Error ") {
			if e := NewNSError(line); e != nil {
				nserrors = append(nserrors, *e)
			}
		}

	}
	if err := scanner.Err(); err != nil {
		return nil
	}

	// Prefer NSErrors if found for all errors,
	// this is because an NSError has a suggestion in addition to the error reason,
	// but we use regular expression for parsing NSErrors.
	if len(nserrors) == len(errorLines) {
		errorLines = []string{}
		for _, nserror := range nserrors {
			errorLines = append(errorLines, nserror.Error())
		}
	}

	return errorLines
}
