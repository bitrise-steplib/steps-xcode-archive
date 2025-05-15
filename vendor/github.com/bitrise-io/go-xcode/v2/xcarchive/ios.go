package xcarchive

import (
	"fmt"

	"github.com/bitrise-io/go-xcode/v2/autocodesign"
	"github.com/bitrise-io/go-xcode/xcarchive"
)

// IosArchive ...
type IosArchive struct {
	xcarchive.IosArchive
}

// NewIosArchive ...
func NewIosArchive(path string) (IosArchive, error) {
	archive, err := xcarchive.NewIosArchive(path)

	return IosArchive{
		IosArchive: archive,
	}, err
}

// IsSigningManagedAutomatically ...
func (archive IosArchive) IsSigningManagedAutomatically() (bool, error) {
	return archive.IsXcodeManaged(), nil
}

// Platform ...
func (archive IosArchive) Platform() (autocodesign.Platform, error) {
	platformName := archive.Application.InfoPlist["DTPlatformName"]
	switch platformName {
	case "iphoneos":
		return autocodesign.IOS, nil
	case "appletvos":
		return autocodesign.TVOS, nil
	default:
		return "", fmt.Errorf("unsupported platform found: %s", platformName)
	}
}

// GetAppLayout ...
func (archive IosArchive) GetAppLayout(_ bool) (autocodesign.AppLayout, error) {
	platform, err := archive.Platform()
	if err != nil {
		return autocodesign.AppLayout{}, err
	}

	bundleIDEntitlementsMap := archive.BundleIDEntitlementsMap()

	fmt.Printf("Reading %v code sign entitlements", len(bundleIDEntitlementsMap))

	entitlementsMap := map[string]autocodesign.Entitlements{}
	for bundleID, entitlements := range bundleIDEntitlementsMap {
		entitlementsMap[bundleID] = autocodesign.Entitlements(entitlements)
	}

	return autocodesign.AppLayout{
		Platform:                               platform,
		EntitlementsByArchivableTargetBundleID: entitlementsMap,
		UITestTargetBundleIDs:                  nil,
	}, nil
}
