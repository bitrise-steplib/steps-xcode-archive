package step

import (
	"testing"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/xcodeproject/serialized"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcodeproj"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcscheme"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockTargetBuildSettingsProvider struct {
	mock.Mock
}

// TargetBuildSettings ...
func (m *MockTargetBuildSettingsProvider) TargetBuildSettings(xcodeProj *xcodeproj.XcodeProj, target, configuration string, customOptions ...string) (serialized.Object, error) {
	args := m.Called(xcodeProj, target, configuration)
	return args.Get(0).(serialized.Object), args.Error(1)
}

func TestBuildableTargetPlatform(t *testing.T) {
	tests := []struct {
		name              string
		xcodeProj         *xcodeproj.XcodeProj
		scheme            *xcscheme.Scheme
		configurationName string
		settings          serialized.Object
		want              Platform
		wantErr           bool
	}{
		{
			name: "SDKROOT build settings not defined in the project, but showBuildSettings returns it",
			xcodeProj: &xcodeproj.XcodeProj{
				Proj: xcodeproj.Proj{
					Targets: []xcodeproj.Target{
						{
							ID: "target_id",
						},
					},
				},
			},
			scheme: &xcscheme.Scheme{
				BuildAction: xcscheme.BuildAction{
					BuildActionEntries: []xcscheme.BuildActionEntry{
						{
							BuildForArchiving: "YES",
							BuildableReference: xcscheme.BuildableReference{
								BuildableName:       "bitrise.app",
								BlueprintIdentifier: "target_id",
							},
						},
					},
				},
			},
			configurationName: "",
			settings:          serialized.Object(map[string]interface{}{"SDKROOT": "iphoneos"}),
			want:              iOS,
			wantErr:           false,
		},
		{
			name: "fails if showBuildSettings does not return SDKROOT",
			xcodeProj: &xcodeproj.XcodeProj{
				Proj: xcodeproj.Proj{
					Targets: []xcodeproj.Target{
						{
							ID: "target_id",
						},
					},
				},
			},
			scheme: &xcscheme.Scheme{
				BuildAction: xcscheme.BuildAction{
					BuildActionEntries: []xcscheme.BuildActionEntry{
						{
							BuildForArchiving: "YES",
							BuildableReference: xcscheme.BuildableReference{
								BuildableName:       "bitrise.app",
								BlueprintIdentifier: "target_id",
							},
						},
					},
				},
			},
			configurationName: "",
			settings:          serialized.Object(map[string]interface{}{}),
			want:              Platform(""),
			wantErr:           true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &MockTargetBuildSettingsProvider{}
			provider.
				On("TargetBuildSettings", mock.AnythingOfType("*xcodeproj.XcodeProj"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).
				Return(tt.settings, nil)

			got, err := BuildableTargetPlatform(tt.xcodeProj, tt.scheme, tt.configurationName, []string{}, provider, log.NewLogger())
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
		name          string
		buildSettings serialized.Object
		want          Platform
		wantErr       bool
	}{
		{
			name:          "iOS",
			buildSettings: serialized.Object(map[string]interface{}{"SDKROOT": "iphoneos"}),
			want:          iOS,
			wantErr:       false,
		},
		{
			name:          "osX",
			buildSettings: serialized.Object(map[string]interface{}{"SDKROOT": "macosx"}),
			want:          osX,
			wantErr:       false,
		},
		{
			name:          "tvOS",
			buildSettings: serialized.Object(map[string]interface{}{"SDKROOT": "appletvos"}),
			want:          tvOS,
			wantErr:       false,
		},
		{
			name:          "watchOS",
			buildSettings: serialized.Object(map[string]interface{}{"SDKROOT": "watchos"}),
			want:          watchOS,
			wantErr:       false,
		},
		{
			name:          "iOS with SDK path",
			buildSettings: serialized.Object(map[string]interface{}{"SDKROOT": "/Applications/Xcode.app/Contents/Developer/Platforms/iPhoneOS.platform/Developer/SDKs/iPhoneOS13.4.sdk"}),
			want:          iOS,
			wantErr:       false,
		},
		{
			name:          "osX with SDK path",
			buildSettings: serialized.Object(map[string]interface{}{"SDKROOT": "/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX10.15.sdk"}),
			want:          osX,
			wantErr:       false,
		},
		{
			name:          "tvOS with SDK path",
			buildSettings: serialized.Object(map[string]interface{}{"SDKROOT": "/Applications/Xcode.app/Contents/Developer/Platforms/AppleTVOS.platform/Developer/SDKs/AppleTVOS.sdk"}),
			want:          tvOS,
			wantErr:       false,
		},
		{
			name:          "watchOS with SDK path",
			buildSettings: serialized.Object(map[string]interface{}{"SDKROOT": "/Applications/Xcode.app/Contents/Developer/Platforms/WatchOS.platform/Developer/SDKs/WatchOS.sdk"}),
			want:          watchOS,
			wantErr:       false,
		},
		{
			name:          "unkown SDK path",
			buildSettings: serialized.Object(map[string]interface{}{"SDKROOT": "/Applications/Xcode.app/Contents/Developer/Platforms/WatchSimulator.platform/Developer/SDKs/WatchSimulator.sdk"}),
			want:          Platform(""),
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getPlatform(tt.buildSettings)
			if (err != nil) != tt.wantErr {
				t.Errorf("getPlatform() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}
