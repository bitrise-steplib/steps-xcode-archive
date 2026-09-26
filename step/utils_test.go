package step

import (
	"testing"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/stretchr/testify/require"
)

func Test_findIDEDistrubutionLogsPath(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    string
		wantErr bool
	}{
		{
			name:   "match double quotes",
			output: `IDEDistribution: -[IDEDistributionLogging _createLoggingBundleAtPath:]: Created bundle at path "sample.xcdistributionlogs".`,
			want:   "sample.xcdistributionlogs",
		},
		{
			name:   "match single quotes",
			output: `IDEDistribution: -[IDEDistributionLogging _createLoggingBundleAtPath:]: Created bundle at path 'sample.xcdistributionlogs'.`,
			want:   "sample.xcdistributionlogs",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.NewLogger()

			got, err := findIDEDistrubutionLogsPath(tt.output, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("findIDEDistrubutionLogsPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("findIDEDistrubutionLogsPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_filterSPMAdditionalOptions(t *testing.T) {
	tests := []struct {
		name                        string
		xcodebuildAdditionalOptions []string
		want                        []string
	}{
		{
			name:                        "no SPM options",
			xcodebuildAdditionalOptions: []string{"-scheme", "MyScheme", "-configuration", "Release"},
			want:                        []string{},
		},
		{
			name:                        "with SPM flags",
			xcodebuildAdditionalOptions: []string{"-scheme", "MyScheme", "-skipPackagePluginValidation", "-skipMacroValidation", "-clonedSourcePackagesDirPath", "/path/to/packages"},
			want:                        []string{"-skipPackagePluginValidation", "-skipMacroValidation", "-clonedSourcePackagesDirPath", "/path/to/packages"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterSPMAdditionalOptions(tt.xcodebuildAdditionalOptions)
			require.Equal(t, tt.want, got)
		})
	}
}
