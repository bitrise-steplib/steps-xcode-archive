package export

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-xcode/profileutil"

	"github.com/bitrise-io/go-xcode/certificateutil"
)

func TestCreateIosCodeSignGroups(t *testing.T) {
	cert := certificateutil.CertificateInfoModel{
		Serial:     "my-serial",
		CommonName: "iPhone Distribution: Main",
		TeamID:     "tid",
	}

	profileMain1 := profileutil.ProvisioningProfileInfoModel{
		Name:                  "Main1",
		BundleID:              "io.bitrise.Main",
		TeamID:                "tid",
		DeveloperCertificates: []certificateutil.CertificateInfoModel{cert},
		UUID:                  "1",
	}
	profileMain2 := profileutil.ProvisioningProfileInfoModel{
		Name:                  "Main2",
		BundleID:              "io.bitrise.Main",
		TeamID:                "tid",
		DeveloperCertificates: []certificateutil.CertificateInfoModel{cert},
		UUID:                  "2",
	}
	profileWildcard1 := profileutil.ProvisioningProfileInfoModel{
		Name:                  "Wildcard1",
		BundleID:              "io.*",
		TeamID:                "tid",
		DeveloperCertificates: []certificateutil.CertificateInfoModel{cert},
		UUID:                  "3",
	}
	profileExtension1 := profileutil.ProvisioningProfileInfoModel{
		Name:                  "Extension1",
		BundleID:              "io.bitrise.Main.watchkitapp.watchkitextension",
		TeamID:                "tid",
		DeveloperCertificates: []certificateutil.CertificateInfoModel{cert},
		UUID:                  "5",
	}
	profileExtension2 := profileutil.ProvisioningProfileInfoModel{
		Name:                  "Extension2",
		BundleID:              "io.bitrise.Main.watchkitapp.watchkitextension",
		TeamID:                "tid",
		DeveloperCertificates: []certificateutil.CertificateInfoModel{cert},
		UUID:                  "6",
	}

	tests := []struct {
		name             string
		selectableGroups []SelectableCodeSignGroup
		want             []IosCodeSignGroup
	}{
		{
			name: "mixed group members",
			selectableGroups: []SelectableCodeSignGroup{
				SelectableCodeSignGroup{
					Certificate: cert,
					BundleIDProfilesMap: map[string][]profileutil.ProvisioningProfileInfoModel{
						"io.bitrise.Main":                               {profileMain1, profileMain2},
						"io.bitrise.Main.watchkitapp":                   {profileWildcard1},
						"io.bitrise.Main.watchkitapp.watchkitextension": {profileExtension1, profileExtension2},
					},
				},
			},
			want: []IosCodeSignGroup{
				*NewIOSGroup(cert, map[string]profileutil.ProvisioningProfileInfoModel{
					"io.bitrise.Main":                               profileMain1,
					"io.bitrise.Main.watchkitapp":                   profileWildcard1,
					"io.bitrise.Main.watchkitapp.watchkitextension": profileExtension1,
				}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CreateIosCodeSignGroups(tt.selectableGroups)

			log.Printf("\nFiltered groups:")
			for i, group := range got {
				log.Printf("Group #%d:", i)
				for bundleID, profile := range group.BundleIDProfileMap() {
					log.Printf(" - %s: %s (%s)", bundleID, profile.Name, profile.UUID)
				}
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateIosCodeSignGroups() = %v, want %v", got, tt.want)

				fmt.Println("\nExpecting:")
				for i, group := range tt.want {
					log.Printf("Group #%d:", i)
					for bundleID, profile := range group.BundleIDProfileMap() {
						log.Printf(" - %s: %s (%s)", bundleID, profile.Name, profile.UUID)
					}
				}
			}
		})
	}
}
