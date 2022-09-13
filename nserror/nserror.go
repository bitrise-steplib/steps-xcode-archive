package nserror

import (
	"regexp"
)

type Error struct {
	Description string
	Suggestion  string
}

func New(str string) *Error {
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

	return &Error{
		Description: description,
		Suggestion:  suggestion,
	}
}

func (e Error) Error() string {
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
