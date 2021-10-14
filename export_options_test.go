package main

import (
	"fmt"
	"testing"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-xcode/certificateutil"
	"github.com/bitrise-io/go-xcode/exportoptions"
	"github.com/bitrise-io/go-xcode/profileutil"
	"github.com/bitrise-io/go-xcode/xcodeproject/serialized"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcodeproj"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcscheme"
	"github.com/stretchr/testify/require"
)

const (
	expectedDevelopementExportOptions = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
	<dict>
		<key>distributionBundleIdentifier</key>
		<string>io.bundle.id</string>
		<key>iCloudContainerEnvironment</key>
		<string>Production</string>
		<key>method</key>
		<string>development</string>
		<key>provisioningProfiles</key>
		<dict>
			<key>io.bundle.id</key>
			<string>Development Application Profile</string>
		</dict>
		<key>signingCertificate</key>
		<string>Development Certificate</string>
		<key>teamID</key>
		<string>TEAM123</string>
	</dict>
</plist>`
	expectedAdHocExportOptions = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
	<dict>
		<key>distributionBundleIdentifier</key>
		<string>io.bundle.id</string>
		<key>iCloudContainerEnvironment</key>
		<string>Production</string>
		<key>method</key>
		<string>ad-hoc</string>
		<key>provisioningProfiles</key>
		<dict>
			<key>io.bundle.id</key>
			<string>Development Application Profile</string>
		</dict>
		<key>signingCertificate</key>
		<string>Development Certificate</string>
		<key>teamID</key>
		<string>TEAM123</string>
	</dict>
</plist>`
	expectedXcode12AppStoreExportOptions = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
	<dict>
		<key>iCloudContainerEnvironment</key>
		<string>Production</string>
		<key>method</key>
		<string>app-store</string>
		<key>provisioningProfiles</key>
		<dict>
			<key>io.bundle.id</key>
			<string>Development Application Profile</string>
			<key>io.bundle.id.AppClipID</key>
			<string>Development App Clip Profile</string>
		</dict>
		<key>signingCertificate</key>
		<string>Development Certificate</string>
		<key>teamID</key>
		<string>TEAM123</string>
	</dict>
</plist>`
	expectedXcode13AppStoreExportOptions = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
	<dict>
		<key>iCloudContainerEnvironment</key>
		<string>Production</string>
		<key>manageAppVersionAndBuildNumber</key>
		<false/>
		<key>method</key>
		<string>app-store</string>
		<key>provisioningProfiles</key>
		<dict>
			<key>io.bundle.id</key>
			<string>Development Application Profile</string>
			<key>io.bundle.id.AppClipID</key>
			<string>Development App Clip Profile</string>
		</dict>
		<key>signingCertificate</key>
		<string>Development Certificate</string>
		<key>teamID</key>
		<string>TEAM123</string>
	</dict>
</plist>`
)

func TestExportOptionsGenerator_GenerateApplicationExportOptions(t *testing.T) {
	log.SetEnableDebugLog(true)
	const (
		bundleID     = "io.bundle.id"
		bundleIDClip = "io.bundle.id.AppClipID"
		teamID       = "TEAM123"
	)

	certificate := certificateutil.CertificateInfoModel{Serial: "serial", CommonName: "Development Certificate", TeamID: teamID}

	tests := []struct {
		name         string
		exportMethod exportoptions.Method
		xcodeVersion int64
		want         string
		wantErr      bool
	}{
		{
			name:         "Development",
			exportMethod: exportoptions.MethodDevelopment,
			xcodeVersion: 13,
			want:         expectedDevelopementExportOptions,
		},
		{
			name:         "Ad-hoc",
			exportMethod: exportoptions.MethodAdHoc,
			xcodeVersion: 13,
			want:         expectedAdHocExportOptions,
		},
		{
			name:         "App-store Xcode 12",
			exportMethod: exportoptions.MethodAppStore,
			xcodeVersion: 12,
			want:         expectedXcode12AppStoreExportOptions,
		},
		{
			name:         "App-store Xcode 13",
			exportMethod: exportoptions.MethodAppStore,
			xcodeVersion: 13,
			want:         expectedXcode13AppStoreExportOptions,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			appClipTarget := givenAppClipTarget()
			applicationTarget := givenApplicationTarget([]xcodeproj.Target{appClipTarget})
			xcodeProj := givenXcodeproj([]xcodeproj.Target{applicationTarget, appClipTarget})
			scheme := givenScheme(applicationTarget)
			g := NewExportOptionsGenerator(&xcodeProj, &scheme, "")
			g.certificateProvider = MockCodesignIdentityProvider{
				[]certificateutil.CertificateInfoModel{certificate},
			}

			profile := profileutil.ProvisioningProfileInfoModel{
				BundleID:              bundleID,
				TeamID:                teamID,
				ExportType:            tt.exportMethod,
				Name:                  "Development Application Profile",
				DeveloperCertificates: []certificateutil.CertificateInfoModel{certificate},
			}
			profileForClip := profileutil.ProvisioningProfileInfoModel{
				BundleID:              bundleIDClip,
				TeamID:                teamID,
				ExportType:            tt.exportMethod,
				Name:                  "Development App Clip Profile",
				DeveloperCertificates: []certificateutil.CertificateInfoModel{certificate},
			}
			g.profileProvider = MockProvisioningProfileProvider{
				[]profileutil.ProvisioningProfileInfoModel{
					profile,
					profileForClip,
				},
			}

			cloudKitEntitlement := map[string]interface{}{"com.apple.developer.icloud-services": []string{"CloudKit"}}
			g.targetInfoProvider = MockTargetInfoProvider{
				bundleID:             map[string]string{"Application": bundleID, "App Clip": bundleIDClip},
				codesignEntitlements: map[string]serialized.Object{"Application": cloudKitEntitlement},
			}

			// Act
			gotOpts, err := g.GenerateApplicationExportOptions(tt.exportMethod, "Production", teamID, true, true, false, tt.xcodeVersion)

			// Assert
			require.NoError(t, err)

			got, err := gotOpts.String()
			require.NoError(t, err)
			fmt.Println(got)
			require.Equal(t, tt.want, got)
		})
	}
}

type MockCodesignIdentityProvider struct {
	codesignIdentities []certificateutil.CertificateInfoModel
}

func (p MockCodesignIdentityProvider) ListCodesignIdentities() ([]certificateutil.CertificateInfoModel, error) {
	return p.codesignIdentities, nil
}

type MockProvisioningProfileProvider struct {
	profileInfos []profileutil.ProvisioningProfileInfoModel
}

func (p MockProvisioningProfileProvider) ListProvisioningProfiles() ([]profileutil.ProvisioningProfileInfoModel, error) {
	return p.profileInfos, nil
}

type MockTargetInfoProvider struct {
	bundleID             map[string]string
	codesignEntitlements map[string]serialized.Object
}

func (b MockTargetInfoProvider) TargetBundleID(target, configuration string) (string, error) {
	return b.bundleID[target], nil
}

func (b MockTargetInfoProvider) TargetCodeSignEntitlements(target, configuration string) (serialized.Object, error) {
	return b.codesignEntitlements[target], nil
}

func givenAppClipTarget() xcodeproj.Target {
	return xcodeproj.Target{
		ID:               "app_clip_id",
		Name:             "App Clip",
		ProductReference: xcodeproj.ProductReference{Path: "Fruta iOS Clip.app"},
		ProductType:      appClipProductType,
	}
}

func givenApplicationTarget(dependentTargets []xcodeproj.Target) xcodeproj.Target {
	var dependencies []xcodeproj.TargetDependency
	for _, target := range dependentTargets {
		dependencies = append(dependencies, xcodeproj.TargetDependency{Target: target})
	}

	return xcodeproj.Target{
		ID:               "application_id",
		Name:             "Application",
		Dependencies:     dependencies,
		ProductReference: xcodeproj.ProductReference{Path: "Fruta.app"},
	}
}

func givenXcodeproj(targets []xcodeproj.Target) xcodeproj.XcodeProj {
	return xcodeproj.XcodeProj{
		Proj: xcodeproj.Proj{
			Targets: targets,
		},
	}
}

func givenScheme(archivableTarget xcodeproj.Target) xcscheme.Scheme {
	return xcscheme.Scheme{
		BuildAction: xcscheme.BuildAction{
			BuildActionEntries: []xcscheme.BuildActionEntry{
				{
					BuildForArchiving: "YES",
					BuildableReference: xcscheme.BuildableReference{
						BuildableName:       archivableTarget.ProductReference.Path,
						BlueprintIdentifier: archivableTarget.ID,
					},
				},
			},
		},
	}
}
