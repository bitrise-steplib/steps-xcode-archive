package step

import (
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		str  string
		want *NSError
	}{
		{
			name: "Real NSError",
			str:  `Error Domain=IDEProvisioningErrorDomain Code=9 ""ios-simple-objc.app" requires a provisioning profile." UserInfo={IDEDistributionIssueSeverity=3, NSLocalizedDescription="ios-simple-objc.app" requires a provisioning profile., NSLocalizedRecoverySuggestion=Add a profile to the "provisioningProfiles" dictionary in your Export Options property list.}`,
			want: &NSError{
				Description: `"ios-simple-objc.app" requires a provisioning profile.`,
				Suggestion:  `Add a profile to the "provisioningProfiles" dictionary in your Export Options property list.`,
			},
		},
		{
			name: "UserInfo properties order changed",
			str:  `Error Domain=IDEProvisioningErrorDomain Code=9 UserInfo={NSLocalizedRecoverySuggestion=Add a profile to the "provisioningProfiles" dictionary in your Export Options property list., IDEDistributionIssueSeverity=3, NSLocalizedDescription="ios-simple-objc.app" requires a provisioning profile.}`,
			want: &NSError{
				Description: `"ios-simple-objc.app" requires a provisioning profile.`,
				Suggestion:  `Add a profile to the "provisioningProfiles" dictionary in your Export Options property list.`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewNSError(tt.str); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewNSError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_findXcodebuildErrors(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want []string
	}{
		{
			name: "Regular error",
			out:  `error: exportArchive: "code-sign-test.app" requires a provisioning profile.`,
			want: []string{`error: exportArchive: "code-sign-test.app" requires a provisioning profile.`},
		},
		{
			name: "xcodebuild error",
			out: `xcodebuild: error: Failed to build project code-sign-test with scheme code-sign-test.
        Reason: This scheme builds an embedded Apple Watch app. watchOS 9.0 must be installed in order to archive the scheme
        Recovery suggestion: watchOS 9.0 is not installed. To use with Xcode, first download and install the platform`,
			want: []string{`xcodebuild: error: Failed to build project code-sign-test with scheme code-sign-test.
Reason: This scheme builds an embedded Apple Watch app. watchOS 9.0 must be installed in order to archive the scheme
Recovery suggestion: watchOS 9.0 is not installed. To use with Xcode, first download and install the platform`},
		},
		{
			name: "NSError",
			out:  `Error Domain=IDEProvisioningErrorDomain Code=9 ""code-sign-test.app" requires a provisioning profile." UserInfo={IDEDistributionIssueSeverity=3, NSLocalizedDescription="code-sign-test.app" requires a provisioning profile., NSLocalizedRecoverySuggestion=Add a profile to the "provisioningProfiles" dictionary in your Export Options property list.}`,
			want: []string{`"code-sign-test.app" requires a provisioning profile. Add a profile to the "provisioningProfiles" dictionary in your Export Options property list.`},
		},
		{
			name: "Regular error and NSError pair",
			out: `error: exportArchive: "share-extension.appex" requires a provisioning profile.

Error Domain=IDEProvisioningErrorDomain Code=9 ""share-extension.appex" requires a provisioning profile." UserInfo={IDEDistributionIssueSeverity=3, NSLocalizedDescription="share-extension.appex" requires a provisioning profile., NSLocalizedRecoverySuggestion=Add a profile to the "provisioningProfiles" dictionary in your Export Options property list.}`,
			want: []string{`"share-extension.appex" requires a provisioning profile. Add a profile to the "provisioningProfiles" dictionary in your Export Options property list.`},
		},
		{
			name: "Extra regular error",
			out: `error: exportArchive: "watchkit-app.app" requires a provisioning profile.

Error Domain=IDEProvisioningErrorDomain Code=9 ""watchkit-app.app" requires a provisioning profile." UserInfo={IDEDistributionIssueSeverity=3, NSLocalizedDescription="watchkit-app.app" requires a provisioning profile., NSLocalizedRecoverySuggestion=Add a profile to the "provisioningProfiles" dictionary in your Export Options property list.}

error: exportArchive: "share-extension.appex" requires a provisioning profile.
`,
			want: []string{
				`"watchkit-app.app" requires a provisioning profile. Add a profile to the "provisioningProfiles" dictionary in your Export Options property list.`,
				`error: exportArchive: "share-extension.appex" requires a provisioning profile.`,
			},
		},
		{
			name: "Regular error with NSError pair and xcodebuild error",
			out: `error: exportArchive: "watchkit-app.app" requires a provisioning profile.

Error Domain=IDEProvisioningErrorDomain Code=9 ""watchkit-app.app" requires a provisioning profile." UserInfo={IDEDistributionIssueSeverity=3, NSLocalizedDescription="watchkit-app.app" requires a provisioning profile., NSLocalizedRecoverySuggestion=Add a profile to the "provisioningProfiles" dictionary in your Export Options property list.}

xcodebuild: error: Failed to build project code-sign-test with scheme code-sign-test.
        Reason: This scheme builds an embedded Apple Watch app. watchOS 9.0 must be installed in order to archive the scheme
        Recovery suggestion: watchOS 9.0 is not installed. To use with Xcode, first download and install the platform.
`,
			want: []string{
				`"watchkit-app.app" requires a provisioning profile. Add a profile to the "provisioningProfiles" dictionary in your Export Options property list.`,
				`xcodebuild: error: Failed to build project code-sign-test with scheme code-sign-test.
Reason: This scheme builds an embedded Apple Watch app. watchOS 9.0 must be installed in order to archive the scheme
Recovery suggestion: watchOS 9.0 is not installed. To use with Xcode, first download and install the platform.`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findXcodebuildErrors(tt.out); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findXcodebuildErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}
