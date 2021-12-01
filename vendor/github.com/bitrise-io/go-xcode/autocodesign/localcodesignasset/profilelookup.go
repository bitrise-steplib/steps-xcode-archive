package localcodesignasset

import (
	"reflect"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-io/go-xcode/autocodesign"
	"github.com/bitrise-io/go-xcode/profileutil"
)

func findProfile(localProfiles []profileutil.ProvisioningProfileInfoModel, platform autocodesign.Platform, distributionType autocodesign.DistributionType, bundleID string, entitlements autocodesign.Entitlements, minProfileDaysValid int, certSerials []string, deviceIDs []string) *profileutil.ProvisioningProfileInfoModel {
	for _, profile := range localProfiles {
		if isProfileMatching(profile, platform, distributionType, bundleID, entitlements, minProfileDaysValid, certSerials, deviceIDs) {
			return &profile
		}
	}

	return nil
}

func isProfileMatching(profile profileutil.ProvisioningProfileInfoModel, platform autocodesign.Platform, distributionType autocodesign.DistributionType, bundleID string, entitlements autocodesign.Entitlements, minProfileDaysValid int, certSerials []string, deviceIDs []string) bool {
	if !isActive(profile, minProfileDaysValid) {
		return false
	}

	if !hasMatchingDistributionType(profile, distributionType) {
		return false
	}

	if !hasMatchingBundleID(profile, bundleID) {
		return false
	}

	if !hasMatchingPlatform(profile, platform) {
		return false
	}

	if !hasMatchingLocalCertificates(profile, certSerials) {
		return false
	}

	if !containsAllAppEntitlements(profile, entitlements) {
		return false
	}

	if !provisionsDevices(profile, deviceIDs) {
		return false
	}

	// Drop Xcode-managed profiles
	// as Bitrise-managed automatic code signing enforces manually managed code signing on the given project.
	if profile.IsXcodeManaged() {
		return false
	}

	return true
}

func hasMatchingBundleID(profile profileutil.ProvisioningProfileInfoModel, bundleID string) bool {
	return profile.BundleID == bundleID
}

func hasMatchingLocalCertificates(profile profileutil.ProvisioningProfileInfoModel, localCertificateSerials []string) bool {
	var profileCertificateSerials []string
	for _, certificate := range profile.DeveloperCertificates {
		profileCertificateSerials = append(profileCertificateSerials, certificate.Serial)
	}

	for _, serial := range localCertificateSerials {
		if !sliceutil.IsStringInSlice(serial, profileCertificateSerials) {
			return false
		}
	}

	return true
}

func containsAllAppEntitlements(profile profileutil.ProvisioningProfileInfoModel, appEntitlements autocodesign.Entitlements) bool {
	profileEntitlements := autocodesign.Entitlements(profile.Entitlements)
	hasMissingEntitlement := false

	for key, value := range appEntitlements {
		profileEntitlementValue := profileEntitlements[key]

		// The project entitlement values can have variables coming from build settings which will be resolved later
		// during the archive action. It is not the best but this is also the logic used at other places. An example of
		// what we could be comparing:
		// 		$(AppIdentifierPrefix)${BASE_BUNDLE_ID}.ios == 72SA8V3WYL.io.bitrise.samples.fruta.los
		if key == autocodesign.ICloudIdentifiersEntitlementKey {
			missingContainers, err := autocodesign.FindMissingContainers(appEntitlements, profileEntitlements)
			if err != nil || len(missingContainers) > 0 {
				return false
			}
		} else if !reflect.DeepEqual(profileEntitlementValue, value) {
			return false
		}
	}

	return !hasMissingEntitlement
}

func hasMatchingDistributionType(profile profileutil.ProvisioningProfileInfoModel, distributionType autocodesign.DistributionType) bool {
	return autocodesign.DistributionType(profile.ExportType) == distributionType
}

func isActive(profile profileutil.ProvisioningProfileInfoModel, minProfileDaysValid int) bool {
	expiration := time.Now()
	if minProfileDaysValid > 0 {
		expiration = expiration.AddDate(0, 0, minProfileDaysValid)
	}

	return expiration.Before(profile.ExpirationDate)
}

func hasMatchingPlatform(profile profileutil.ProvisioningProfileInfoModel, platform autocodesign.Platform) bool {
	return strings.ToLower(string(platform)) == string(profile.Type)
}

func provisionsDevices(profile profileutil.ProvisioningProfileInfoModel, deviceIDs []string) bool {
	if profile.ProvisionsAllDevices || len(deviceIDs) == 0 {
		return true
	}

	if len(profile.ProvisionedDevices) == 0 {
		return false
	}

	for _, deviceID := range deviceIDs {
		if contains(profile.ProvisionedDevices, deviceID) {
			continue
		}
		return false
	}

	return true
}
