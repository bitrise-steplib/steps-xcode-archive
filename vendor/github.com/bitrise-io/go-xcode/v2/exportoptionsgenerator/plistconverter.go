package exportoptionsgenerator

import (
	plistutilv1 "github.com/bitrise-io/go-xcode/plistutil"
	"github.com/bitrise-io/go-xcode/v2/plistutil"
)

// TODO: remove this function when export package is migrated to v2 and uses plistutil/v2
func convertToV1PlistData(bundleIDEntitlementsMap map[string]plistutil.PlistData) map[string]plistutilv1.PlistData {
	converted := map[string]plistutilv1.PlistData{}
	for bundleID, entitlements := range bundleIDEntitlementsMap {
		converted[bundleID] = plistutilv1.PlistData(entitlements)
	}
	return converted
}
