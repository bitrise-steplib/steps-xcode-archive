package errorfinder

import (
	"bufio"
	"strings"
)

// FindXcodebuildErrors ...
func FindXcodebuildErrors(out string) []string {
	var errorLines []string       // single line errors with "error: " prefix
	var xcodebuildErrors []string // multiline errors starting with "xcodebuild: error: " prefix
	var nserrors []nsError        // single line NSErrors with schema: Error Domain=<domain> Code=<code> "<reason>" UserInfo=<user_info>

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
				xcodebuildError = ""
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
			if e := newNSError(line); e != nil {
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

func intersection(errorLines []string, nserrors []nsError) []string {
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
