package main

import (
	"errors"
	"strings"
)

func formattedError(err error) string {
	var formatted string

	i := -1
	for {
		i++

		reason := err.Error()

		if err = errors.Unwrap(err); err == nil {
			formatted = appendError(formatted, reason, i, true)
			return formatted
		}

		reason = strings.TrimSuffix(reason, err.Error())
		reason = strings.TrimRight(reason, " ")
		reason = strings.TrimSuffix(reason, ":")

		formatted = appendError(formatted, reason, i, false)
	}
}

func appendError(errorMessage, reason string, i int, last bool) string {
	if i == 0 {
		errorMessage = reason
	} else {
		errorMessage += "\n"
		errorMessage += strings.Repeat("  ", i)
		errorMessage += reason
	}

	if !last {
		errorMessage += ":"
	}

	return errorMessage
}
