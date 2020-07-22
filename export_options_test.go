package main

import (
	"fmt"
	"testing"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-xcode/certificateutil"
	"github.com/bitrise-io/go-xcode/profileutil"
	"github.com/bitrise-io/xcode-project/serialized"
	"github.com/bitrise-io/xcode-project/xcodeproj"
	"github.com/bitrise-io/xcode-project/xcscheme"
	"github.com/stretchr/testify/require"
)

func TestExportOptionsGenerator_GenerateApplicationExportOptions(t *testing.T) {
	log.SetEnableDebugLog(true)

	// Arrange
	appClipTarget := givenAppClipTarget()
	applicationTarget := givenApplicationTarget([]xcodeproj.Target{appClipTarget})
	xcodeProj := givenXcodeproj([]xcodeproj.Target{applicationTarget, appClipTarget})
	scheme := givenScheme(applicationTarget)

	g := NewExportOptionsGenerator(&xcodeProj, &scheme, "")

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
		Name:                  "Development Application Profile",
		DeveloperCertificates: []certificateutil.CertificateInfoModel{certificate},
	}
	g.profileProvider = MockProvisioningProfileProvider{
		[]profileutil.ProvisioningProfileInfoModel{
			profile,
			{
				BundleID:              "io.bundle.AppClipID",
				TeamID:                teamID,
				ExportType:            exportMethod,
				Name:                  "Development App Clip Profile",
				DeveloperCertificates: []certificateutil.CertificateInfoModel{certificate},
			},
		},
	}

	cloudKitEntitlement := map[string]interface{}{"com.apple.developer.icloud-services": []string{"CloudKit"}}
	g.targetInfoProvider = MockTargetInfoProvider{
		bundleID:             map[string]string{"Application": bundleID, "App Clip": "io.bundle.AppClipID"},
		codesignEntitlements: map[string]serialized.Object{"Application": cloudKitEntitlement},
	}

	// Act
	opts, err := g.GenerateApplicationExportOptions(exportMethod, "Production", teamID, true, true, false, 12)

	// Assert
	require.NoError(t, err)

	s, err := opts.String()
	require.NoError(t, err)

	fmt.Println(s)

	expected := `<?xml version="1.0" encoding="UTF-8"?>
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
	require.Equal(t, expected, s)
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
