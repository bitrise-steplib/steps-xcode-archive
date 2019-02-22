package utils

import (
	"reflect"
	"testing"

	"github.com/bitrise-io/steps-xcode-archive/pretty"
	"github.com/bitrise-tools/go-xcode/plistutil"
)

func TestProjectEntitlementsByBundleID(t *testing.T) {
	type args struct {
		pth               string
		schemeName        string
		configurationName string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]plistutil.PlistData
		wantErr bool
	}{
		{
			name: "framework target",
			args: args{
				pth:               "~/Develop/iostest/framework-extension-test/framework-extension-test.xcodeproj",
				schemeName:        "framework-extension-test",
				configurationName: "Debug",
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ProjectEntitlementsByBundleID(tt.args.pth, tt.args.schemeName, tt.args.configurationName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProjectEntitlementsByBundleID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ProjectEntitlementsByBundleID() = %v, want %v", pretty.Object(got), tt.want)
			}
		})
	}
}
