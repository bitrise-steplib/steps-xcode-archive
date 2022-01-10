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

// ReadCodesignParameters ...
func (archive IosArchive) ReadCodesignParameters() (*autocodesign.AppLayout, error) {
	platform, err := archive.Platform()
	if err != nil {
		return nil, err
	}

	bundleIDEntitlementsMap := archive.BundleIDEntitlementsMap()

	entitlementsMap := map[string]autocodesign.Entitlements{}
	for bundleID, entitlements := range bundleIDEntitlementsMap {
		entitlementsMap[bundleID] = autocodesign.Entitlements(entitlements)
	}

	return &autocodesign.AppLayout{
		Platform:                               platform,
		EntitlementsByArchivableTargetBundleID: entitlementsMap,
		UITestTargetBundleIDs:                  nil,
	}, nil
}
