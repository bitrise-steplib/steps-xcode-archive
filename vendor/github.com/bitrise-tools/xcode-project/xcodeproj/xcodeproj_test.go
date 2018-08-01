package xcodeproj

import (
	"path/filepath"
	"testing"

	"github.com/bitrise-tools/xcode-project/serialized"
	"github.com/bitrise-tools/xcode-project/testhelper"
	"github.com/stretchr/testify/require"
)

func TestTargets(t *testing.T) {
	dir := testhelper.GitCloneIntoTmpDir(t, "https://github.com/bitrise-samples/xcode-project-test.git")
	project, err := Open(filepath.Join(dir, "Group/SubProject/SubProject.xcodeproj"))
	require.NoError(t, err)

	{
		target, ok := project.Proj.Target("7D0342D720F4B5AD0050B6A6")
		require.True(t, ok)

		dependentTargets := target.DependentTargets()
		require.Equal(t, 2, len(dependentTargets))
		require.Equal(t, "WatchKitApp", dependentTargets[0].Name)
		require.Equal(t, "WatchKitApp Extension", dependentTargets[1].Name)
	}

	{
		settings, err := project.TargetBuildSettings("SubProject", "Debug", "")
		require.NoError(t, err)
		require.True(t, len(settings) > 0)

		bundleID, err := settings.String("PRODUCT_BUNDLE_IDENTIFIER")
		require.NoError(t, err)
		require.Equal(t, "com.bitrise.SubProject", bundleID)

		infoPlist, err := settings.String("INFOPLIST_PATH")
		require.NoError(t, err)
		require.Equal(t, "SubProject.app/Info.plist", infoPlist)
	}

	{
		bundleID, err := project.TargetBundleID("SubProject", "Debug")
		require.NoError(t, err)
		require.Equal(t, "com.bitrise.SubProject", bundleID)
	}

	{
		properties, err := project.TargetInformationPropertyList("SubProject", "Debug")
		require.NoError(t, err)
		require.Equal(t, serialized.Object{"CFBundlePackageType": "APPL",
			"UISupportedInterfaceOrientations":      []interface{}{"UIInterfaceOrientationPortrait", "UIInterfaceOrientationLandscapeLeft", "UIInterfaceOrientationLandscapeRight"},
			"CFBundleInfoDictionaryVersion":         "6.0",
			"CFBundleName":                          "$(PRODUCT_NAME)",
			"UISupportedInterfaceOrientations~ipad": []interface{}{"UIInterfaceOrientationPortrait", "UIInterfaceOrientationPortraitUpsideDown", "UIInterfaceOrientationLandscapeLeft", "UIInterfaceOrientationLandscapeRight"},
			"CFBundleDevelopmentRegion":             "$(DEVELOPMENT_LANGUAGE)",
			"CFBundleExecutable":                    "$(EXECUTABLE_NAME)",
			"CFBundleShortVersionString":            "1.0",
			"CFBundleVersion":                       "1",
			"LSRequiresIPhoneOS":                    true,
			"UIMainStoryboardFile":                  "Main",
			"UIRequiredDeviceCapabilities":          []interface{}{"armv7"},
			"CFBundleIdentifier":                    "$(PRODUCT_BUNDLE_IDENTIFIER)",
			"UILaunchStoryboardName":                "LaunchScreen"}, properties)
	}

	{
		entitlements, err := project.TargetCodeSignEntitlements("WatchKitApp", "Debug")
		require.NoError(t, err)
		require.Equal(t, serialized.Object{"com.apple.security.application-groups": []interface{}{}}, entitlements)

	}
}

func TestScheme(t *testing.T) {
	dir := testhelper.GitCloneIntoTmpDir(t, "https://github.com/bitrise-samples/xcode-project-test.git")
	project, err := Open(filepath.Join(dir, "XcodeProj.xcodeproj"))
	require.NoError(t, err)

	{
		scheme, ok := project.Scheme("ProjectTodayExtensionScheme")
		require.True(t, ok)
		require.Equal(t, "ProjectTodayExtensionScheme", scheme.Name)
	}

	{
		scheme, ok := project.Scheme("NotExistScheme")
		require.False(t, ok)
		require.Equal(t, "", scheme.Name)
	}
}

func TestSchemes(t *testing.T) {
	dir := testhelper.GitCloneIntoTmpDir(t, "https://github.com/bitrise-samples/xcode-project-test.git")
	project, err := Open(filepath.Join(dir, "XcodeProj.xcodeproj"))
	require.NoError(t, err)

	schemes, err := project.Schemes()
	require.NoError(t, err)
	require.Equal(t, 2, len(schemes))

	require.Equal(t, "ProjectScheme", schemes[0].Name)
	require.Equal(t, "ProjectTodayExtensionScheme", schemes[1].Name)
}

func TestOpenXcodeproj(t *testing.T) {
	dir := testhelper.GitCloneIntoTmpDir(t, "https://github.com/bitrise-samples/xcode-project-test.git")
	project, err := Open(filepath.Join(dir, "XcodeProj.xcodeproj"))
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "XcodeProj.xcodeproj"), project.Path)
	require.Equal(t, "XcodeProj", project.Name)
}

func TestIsXcodeProj(t *testing.T) {
	require.True(t, IsXcodeProj("./BitriseSample.xcodeproj"))
	require.False(t, IsXcodeProj("./BitriseSample.xcworkspace"))
}
