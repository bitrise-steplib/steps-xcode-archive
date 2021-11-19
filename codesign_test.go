package main

import (
	"testing"

	"github.com/bitrise-io/go-xcode/devportalservice"
	"github.com/bitrise-steplib/steps-xcode-archive/mocks"
	"github.com/stretchr/testify/mock"
)

func Test_selectCodeSigningStrategy(t *testing.T) {
	tests := []struct {
		name          string
		opts          CodeSignOpts
		projectHelper ProjectHelper
		want          codeSigningStrategy
		wantErr       bool
	}{
		{
			name: "No auth",
			opts: CodeSignOpts{
				AuthType: NoAuth,
			},
			projectHelper: newProjectHelper(false),
			want:          noCodeSign,
		},
		{
			name: "Apple ID, no connection",
			opts: CodeSignOpts{
				AuthType: AppleIDAuth,
			},
			projectHelper: newProjectHelper(false),
			want:          noCodeSign,
			wantErr:       true,
		},
		{
			name: "Apple ID, with connection",
			opts: CodeSignOpts{
				AuthType: AppleIDAuth,
				AppleServiceConnection: devportalservice.AppleDeveloperConnection{
					AppleIDConnection: &devportalservice.AppleIDConnection{},
				},
			},
			projectHelper: newProjectHelper(false),
			want:          codeSigningBitriseAppleID,
		},
		{
			name: "API Key, Xcode 12",
			opts: CodeSignOpts{
				AuthType: APIKeyAuth,
				AppleServiceConnection: devportalservice.AppleDeveloperConnection{
					APIKeyConnection: &devportalservice.APIKeyConnection{},
				},
				XcodeMajorVersion: 12,
			},
			projectHelper: newProjectHelper(false),
			want:          codeSigningBitriseAPIKey,
		},
		{
			name: "API Key, Xcode 13, Manual signing",
			opts: CodeSignOpts{
				AuthType: APIKeyAuth,
				AppleServiceConnection: devportalservice.AppleDeveloperConnection{
					APIKeyConnection: &devportalservice.APIKeyConnection{},
				},
				XcodeMajorVersion: 13,
			},
			projectHelper: newProjectHelper(false),
			want:          codeSigningBitriseAPIKey,
		},
		{
			name: "API Key, Xcode 13, Xcode managed signing",
			opts: CodeSignOpts{
				AuthType: APIKeyAuth,
				AppleServiceConnection: devportalservice.AppleDeveloperConnection{
					APIKeyConnection: &devportalservice.APIKeyConnection{},
				},
				XcodeMajorVersion: 13,
			},
			projectHelper: newProjectHelper(true),
			want:          codeSigningXcode,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := selectCodeSigningStrategy(tt.opts, tt.projectHelper)
			if (err != nil) != tt.wantErr {
				t.Errorf("selectCodeSigningStrategy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("selectCodeSigningStrategy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func newProjectHelper(isAutoSign bool) ProjectHelper {
	mockProjectHelper := new(mocks.ProjectHelper)

	mockProjectHelper.On("IsSigningManagedAutomatically", mock.Anything).Return(isAutoSign, nil)
	return mockProjectHelper
}
