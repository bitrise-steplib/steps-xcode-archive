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
