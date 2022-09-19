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
	if !isNSError(str) {
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

func isNSError(str string) bool {
	// example: Error Domain=IDEProvisioningErrorDomain Code=9 ""ios-simple-objc.app" requires a provisioning profile."
	//   UserInfo={IDEDistributionIssueSeverity=3, NSLocalizedDescription="ios-simple-objc.app" requires a provisioning profile.,
	//   NSLocalizedRecoverySuggestion=Add a profile to the "provisioningProfiles" dictionary in your Export Options property list.}
	return strings.Contains(str, "Error ") &&
		strings.Contains(str, "Domain=") &&
		strings.Contains(str, "Code=") &&
		strings.Contains(str, "UserInfo=")
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
			return fmt.Errorf("command failed with exit status %d (%s): %w", exitErr.ExitCode(), cmd.PrintableCmd(), errors.New(strings.Join(reasons, "\n")))
		}
		return fmt.Errorf("command failed with exit status %d (%s)", exitErr.ExitCode(), cmd.PrintableCmd())
	}

	return fmt.Errorf("executing command failed (%s): %w", cmd.PrintableCmd(), err)
}

func findXcodebuildErrors(out string) []string {
	var errorLines []string       // single line errors with "error: " prefix
	var xcodebuildErrors []string // multiline errors starting with "xcodebuild: error: " prefix
	var nserrors []NSError        // single line NSErrors with schema: Error Domain=<domain> Code=<code> "<reason>" UserInfo=<user_info>

	isXcodebuildError := false
	var xcodebuildError string

	scanner := bufio.NewScanner(strings.NewReader(out))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()

		if isXcodebuildError {
			line = strings.TrimLeft(line, " ")
			if strings.HasPrefix(line, "Reason: ") || strings.HasPrefix(line, "Recovery suggestion: ") {
				xcodebuildError += "\n" + line
				continue
			} else {
				xcodebuildErrors = append(xcodebuildErrors, xcodebuildError)
				isXcodebuildError = false
			}
		}

		switch {
		case strings.HasPrefix(line, "xcodebuild: error: "):
			xcodebuildError = line
			isXcodebuildError = true
		case strings.HasPrefix(line, "error: ") || strings.Contains(line, " error: "):
			errorLines = append(errorLines, line)
		case strings.HasPrefix(line, "Error "):
			if e := NewNSError(line); e != nil {
				nserrors = append(nserrors, *e)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil
	}

	if xcodebuildError != "" {
		xcodebuildErrors = append(xcodebuildErrors, xcodebuildError)
	}

	// Regular error lines (line with 'error: ' prefix) seems to have
	// NSError line pairs (same description) in some cases.
	errorLines = intersection(errorLines, nserrors)

	return append(errorLines, xcodebuildErrors...)
}

func intersection(errorLines []string, nserrors []NSError) []string {
	union := make([]string, len(errorLines))
	copy(union, errorLines)

	for _, nserror := range nserrors {
		found := false
		for i, errorLine := range errorLines {
			// Checking suffix, as regular error lines have additional prefixes, like "error: exportArchive: "
			if strings.HasSuffix(errorLine, nserror.Description) {
				union[i] = nserror.Error()
				found = true
				break
			}
		}
		if !found {
			union = append(union, nserror.Error())
		}
	}
	return union
}
