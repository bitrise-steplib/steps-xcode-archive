package utils

import (
	"testing"

	"github.com/bitrise-io/xcode-project/serialized"
	"github.com/bitrise-io/xcode-project/xcodeproj"
	"github.com/stretchr/testify/require"
)

func createProject(configuration string, buildSettings serialized.Object) *xcodeproj.XcodeProj {
	return &xcodeproj.XcodeProj{
		Proj: xcodeproj.Proj{
			BuildConfigurationList: xcodeproj.ConfigurationList{
				BuildConfigurations: []xcodeproj.BuildConfiguration{
					xcodeproj.BuildConfiguration{
						Name:          configuration,
						BuildSettings: buildSettings,
					},
				},
			},
		},
	}
}

func TestProjectPlatform1(t *testing.T) {
	proj := createProject("Debug", serialized.Object(map[string]interface{}{"SDKROOT": "iphoneos"}))
	platform, err := ProjectPlatform(proj, "Debug")
	require.NoError(t, err)
	require.Equal(t, iOS, platform)
}

func TestProjectPlatform(t *testing.T) {
	tests := []struct {
		name              string
		xcodeProj         *xcodeproj.XcodeProj
		configurationName string
		want              Platform
		wantErr           string
	}{
		{
			name:              "Detects iOS platform",
			xcodeProj:         createProject("Debug", serialized.Object(map[string]interface{}{"SDKROOT": "iphoneos"})),
			configurationName: "Debug",
			want:              iOS,
			wantErr:           "",
		},
		{
			name:              "Fails if configuration not found",
			xcodeProj:         createProject("Release", serialized.Object(map[string]interface{}{"SDKROOT": "iphoneos"})),
			configurationName: "Debug",
			want:              "",
			wantErr:           "Debug project configuration not found",
		},
		{
			name:              "Fails if SDKROOT not found",
			xcodeProj:         createProject("Debug", serialized.Object(map[string]interface{}{})),
			configurationName: "Debug",
			want:              "",
			wantErr:           `failed to get SDKROOT: key: string("SDKROOT") not found in: serialized.Object(serialized.Object{})`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ProjectPlatform(tt.xcodeProj, tt.configurationName)
			hasError := err != nil
			wantError := tt.wantErr != ""
			if hasError != wantError {
				t.Errorf("ProjectPlatform() error: %s, want: %s", err, tt.wantErr)
				return
			} else if hasError && err.Error() != tt.wantErr {
				t.Errorf("ProjectPlatform() error: %s, want: %s", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("ProjectPlatform() = %v, want %v", got, tt.want)
			}
		})
	}
}
