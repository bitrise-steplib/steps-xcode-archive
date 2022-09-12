package main

import (
	"errors"
	"fmt"
	"testing"
)

func createWrappedError(reasons ...string) error {
	var err error
	for i := len(reasons) - 1; i >= 0; i-- {
		reason := reasons[i]
		if i == len(reasons)-1 {
			err = errors.New(reason)
		} else {
			err = fmt.Errorf("%s: %w", reason, err)
		}
	}
	return err
}

func Test_logError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "Simple error",
			err:      errors.New("step run failed"),
			expected: "step run failed",
		},
		{
			name: "Wrapped error",
			err:  createWrappedError("step run failed", "archiving the project failed", "command (xcodebuild archive) failed with exit status 65"),
			expected: `step run failed:
  archiving the project failed:
    command (xcodebuild archive) failed with exit status 65`,
		},
		{
			name:     "Not wrapped error",
			err:      fmt.Errorf("error 1: %s", fmt.Errorf("error 2")),
			expected: "error 1: error 2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := formattedError(tt.err)
			if s != tt.expected {
				t.Fatalf("expected (%s) != actual(%s)", tt.expected, s)
			}
		})
	}
}
