package stepconf

import (
	"errors"
	"strings"
)

// ErrNotStructPtr indicates a type is not a pointer to a struct.
var ErrNotStructPtr = errors.New("must be a pointer to a struct")

// ParseError occurs when a struct field cannot be set.
type ParseError struct {
	Field string
	Value string
	Err   error
}

// Error implements builtin errors.Error.
func (e *ParseError) Error() string {
	segments := []string{e.Field}
	if e.Value != "" {
		segments = append(segments, e.Value)
	}
	segments = append(segments, e.Err.Error())
	return strings.Join(segments, ": ")
}
