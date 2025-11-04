package step

import (
	"testing"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/xcodeproject/serialized"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockBuildSettingsProvider struct {
	mock.Mock
}

// TargetBuildSettings ...
func (m *MockBuildSettingsProvider) ReadSchemeBuildSettingString(key string) (string, error) {
	args := m.Called(key)
	return args.Get(0).(string), args.Error(1)
}

func TestBuildableTargetPlatform(t *testing.T) {
	tests := []struct {
		name        string
		settings    string
		settingsErr error
		want        Platform
		wantErr     bool
	}{
		{
			name:        "SDKROOT build settings not defined in the project, but showBuildSettings returns it",
			settings:    "iphoneos",
			settingsErr: nil,
			want:        iOS,
			wantErr:     false,
		},
		{
			name:        "fails if showBuildSettings does not return SDKROOT",
			settings:    "",
			settingsErr: serialized.NewKeyNotFoundError("SDKROOT", serialized.Object(map[string]interface{}{})),
			want:        Platform(""),
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &MockBuildSettingsProvider{}
			provider.
				On("ReadSchemeBuildSettingString", mock.AnythingOfType("string")).
				Return(tt.settings, nil)

			got, err := BuildableTargetPlatform(log.NewLogger(), provider)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildableTargetPlatform() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			provider.AssertExpectations(t)
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_getPlatform(t *testing.T) {
	tests := []struct {
		name    string
		sdk     string
		want    Platform
		wantErr bool
	}{
		{
			name:    "iOS",
			sdk:     "iphoneos",
			want:    iOS,
			wantErr: false,
		},
		{
			name:    "osX",
			sdk:     "macosx",
			want:    osX,
			wantErr: false,
		},
		{
			name:    "tvOS",
			sdk:     "appletvos",
			want:    tvOS,
			wantErr: false,
		},
		{
			name:    "watchOS",
			sdk:     "watchos",
			want:    watchOS,
			wantErr: false,
		},
		{
			name:    "iOS with SDK path",
			sdk:     "/Applications/Xcode.app/Contents/Developer/Platforms/iPhoneOS.platform/Developer/SDKs/iPhoneOS13.4.sdk",
			want:    iOS,
			wantErr: false,
		},
		{
			name:    "osX with SDK path",
			sdk:     "/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX10.15.sdk",
			want:    osX,
			wantErr: false,
		},
		{
			name:    "tvOS with SDK path",
			sdk:     "/Applications/Xcode.app/Contents/Developer/Platforms/AppleTVOS.platform/Developer/SDKs/AppleTVOS.sdk",
			want:    tvOS,
			wantErr: false,
		},
		{
			name:    "watchOS with SDK path",
			sdk:     "/Applications/Xcode.app/Contents/Developer/Platforms/WatchOS.platform/Developer/SDKs/WatchOS.sdk",
			want:    watchOS,
			wantErr: false,
		},
		{
			name:    "unkown SDK path",
			sdk:     "/Applications/Xcode.app/Contents/Developer/Platforms/WatchSimulator.platform/Developer/SDKs/WatchSimulator.sdk",
			want:    Platform(""),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getPlatform(tt.sdk)
			if (err != nil) != tt.wantErr {
				t.Errorf("getPlatform() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}
