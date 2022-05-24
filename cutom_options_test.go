package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_xcodebuildCustomOptions(t *testing.T) {
	tests := []struct {
		name          string
		platform      string
		customOptions []string
		want          []string
	}{
		{
			name:     "no custom options",
			platform: "iOS",
			want:     []string{"-destination", "generic/platform=iOS"},
		},
		{
			name:          "custom opts",
			platform:      "iOS",
			customOptions: []string{"-scmProvider", "system"},
			want:          []string{"-destination", "generic/platform=iOS", "-scmProvider", "system"},
		},
		{
			name:          "custom opts with destination",
			platform:      "iOS",
			customOptions: []string{"-scmProvider", "system", "-destination", "generic/platform=iOS"},
			want:          []string{"-scmProvider", "system", "-destination", "generic/platform=iOS"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := xcodebuildCustomOptions(tt.platform, tt.customOptions)

			require.Equal(t, tt.want, got)
		})
	}
}
