package errorfinder

import (
	"regexp"
	"strings"
)

type nsError struct {
	Description string
	Suggestion  string
}

func newNSError(str string) *nsError {
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

	return &nsError{
		Description: description,
		Suggestion:  suggestion,
	}
}

func (e nsError) Error() string {
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

func isNSError(str string) bool {
	// example: Error Domain=IDEProvisioningErrorDomain Code=9 ""ios-simple-objc.app" requires a provisioning profile."
	//   UserInfo={IDEDistributionIssueSeverity=3, NSLocalizedDescription="ios-simple-objc.app" requires a provisioning profile.,
	//   NSLocalizedRecoverySuggestion=Add a profile to the "provisioningProfiles" dictionary in your Export Options property list.}
	return strings.Contains(str, "Error ") &&
		strings.Contains(str, "Domain=") &&
		strings.Contains(str, "Code=") &&
		strings.Contains(str, "UserInfo=")
}
