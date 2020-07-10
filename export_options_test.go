package main

import (
	"testing"

	"github.com/bitrise-io/go-xcode/certificateutil"
	"github.com/bitrise-io/go-xcode/profileutil"
	"github.com/bitrise-io/xcode-project/serialized"
	"github.com/bitrise-io/xcode-project/xcodeproj"
	"github.com/bitrise-io/xcode-project/xcscheme"
	"github.com/stretchr/testify/require"
)

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
	bundleID             string
	codesignEntitlements serialized.Object
}

// TargetBundleID ...
func (b MockTargetInfoProvider) TargetBundleID(target, configuration string) (string, error) {
	return b.bundleID, nil
}

// TargetCodeSignEntitlements ...
func (b MockTargetInfoProvider) TargetCodeSignEntitlements(target, configuration string) (serialized.Object, error) {
	return b.codesignEntitlements, nil
}

func TestExportOptionsGenerator_GenerateExportOptions(t *testing.T) {
	// log.SetEnableDebugLog(true) // uncomment for debugging

	// Arrange
	xcodeProj := &xcodeproj.XcodeProj{
		Proj: xcodeproj.Proj{
			Targets: []xcodeproj.Target{
				{ID: "target_id"},
			},
		},
	}
	scheme := &xcscheme.Scheme{
		BuildAction: xcscheme.BuildAction{
			BuildActionEntries: []xcscheme.BuildActionEntry{
				{
					BuildForArchiving: "YES",
					BuildableReference: xcscheme.BuildableReference{
						BuildableName:       "sample.app",
						BlueprintIdentifier: "target_id",
					},
				},
			},
		},
	}

	g := NewExportOptionsGenerator(xcodeProj, scheme, "")

	const teamID = "TEAM123"
	certificate := certificateutil.CertificateInfoModel{Serial: "serial", CommonName: "Development Certificate", TeamID: teamID}
	g.certificateProvider = MockCodesignIdentityProvider{
		[]certificateutil.CertificateInfoModel{certificate},
	}

	const bundleID = "io.bundle.id"
	const exportMethod = "development"
	profile := profileutil.ProvisioningProfileInfoModel{
		BundleID:              bundleID,
		TeamID:                teamID,
		ExportType:            exportMethod,
		Name:                  "Development Profile",
		DeveloperCertificates: []certificateutil.CertificateInfoModel{certificate},
	}
	g.profileProvider = MockProvisioningProfileProvider{
		[]profileutil.ProvisioningProfileInfoModel{profile},
	}

	cloudKitEntitlement := map[string]interface{}{"com.apple.developer.icloud-services": []string{"CloudKit"}}
	g.targetInfoProvider = MockTargetInfoProvider{bundleID: bundleID, codesignEntitlements: cloudKitEntitlement}

	// Act
	opts, err := g.GenerateExportOptions(exportMethod, "Production", teamID, true, true, false, 11)

	// Assert
	require.NoError(t, err)

	s, err := opts.String()
	require.NoError(t, err)

	expected := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
	<dict>
		<key>iCloudContainerEnvironment</key>
		<string>Production</string>
		<key>method</key>
		<string>development</string>
		<key>provisioningProfiles</key>
		<dict>
			<key>io.bundle.id</key>
			<string>Development Profile</string>
		</dict>
		<key>signingCertificate</key>
		<string>Development Certificate</string>
		<key>teamID</key>
		<string>TEAM123</string>
	</dict>
</plist>`
	require.Equal(t, expected, s)
}
