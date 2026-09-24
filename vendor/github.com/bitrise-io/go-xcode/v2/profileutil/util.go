package profileutil

import (
	"strings"

	"github.com/bitrise-io/go-xcode/v2/plistutil"
)

// KnownProfileCapabilitiesMap ...
var KnownProfileCapabilitiesMap = map[ProfileType]map[string]bool{
	ProfileTypeMacOs: {
		"com.apple.developer.networking.networkextension":                        true,
		"com.apple.developer.icloud-container-environment":                       true,
		"com.apple.developer.icloud-container-development-container-identifiers": true,
		"com.apple.developer.aps-environment":                                    true,
		"keychain-access-groups":                                                 true,
		"com.apple.developer.icloud-services":                                    true,
		"com.apple.developer.icloud-container-identifiers":                       true,
		"com.apple.developer.networking.vpn.api":                                 true,
		"com.apple.developer.ubiquity-kvstore-identifier":                        true,
		"com.apple.developer.ubiquity-container-identifiers":                     true,
		"com.apple.developer.game-center":                                        true,
		"com.apple.application-identifier":                                       true,
		"com.apple.developer.team-identifier":                                    true,
		"com.apple.developer.maps":                                               true,
	},
	ProfileTypeIos: {
		"com.apple.developer.in-app-payments":                 true,
		"com.apple.security.application-groups":               true,
		"com.apple.developer.default-data-protection":         true,
		"com.apple.developer.healthkit":                       true,
		"com.apple.developer.homekit":                         true,
		"com.apple.developer.networking.HotspotConfiguration": true,
		"inter-app-audio":                                     true,
		"keychain-access-groups":                              true,
		"com.apple.developer.networking.multipath":            true,
		"com.apple.developer.nfc.readersession.formats":       true,
		"com.apple.developer.networking.networkextension":     true,
		"aps-environment":                                     true,
		"com.apple.developer.associated-domains":              true,
		"com.apple.developer.siri":                            true,
		"com.apple.developer.networking.vpn.api":              true,
		"com.apple.external-accessory.wireless-configuration": true,
		"com.apple.developer.pass-type-identifiers":           true,
		"com.apple.developer.icloud-container-identifiers":    true,
	},
}

// IsXcodeManaged ...
func IsXcodeManaged(profileName string) bool {
	if strings.HasPrefix(profileName, "XC") {
		return true
	}
	if strings.Contains(profileName, "Provisioning Profile") {
		if strings.HasPrefix(profileName, "iOS Team") ||
			strings.HasPrefix(profileName, "Mac Catalyst Team") ||
			strings.HasPrefix(profileName, "tvOS Team") ||
			strings.HasPrefix(profileName, "Mac Team") {
			return true
		}
	}
	return false
}

// MatchTargetAndProfileEntitlements ...
func MatchTargetAndProfileEntitlements(targetEntitlements plistutil.PlistData, profileEntitlements plistutil.PlistData, profileType ProfileType) []string {
	var missingEntitlements []string

	for key := range targetEntitlements {
		knownCapabilities, ok := KnownProfileCapabilitiesMap[profileType]
		if !ok {
			continue
		}
		_, known := knownCapabilities[key]
		if !known {
			continue
		}
		_, found := profileEntitlements[key]
		if !found {
			missingEntitlements = append(missingEntitlements, key)
		}
	}

	return missingEntitlements
}
