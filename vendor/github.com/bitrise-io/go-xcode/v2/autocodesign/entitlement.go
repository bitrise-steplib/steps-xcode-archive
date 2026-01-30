package autocodesign

import (
	"errors"
	"fmt"
	"slices"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/devportalclient/appstoreconnect"
	"github.com/bitrise-io/go-xcode/xcodeproject/serialized"
)

// ICloudIdentifiersEntitlementKey ...
const ICloudIdentifiersEntitlementKey = "com.apple.developer.icloud-container-identifiers"

// DataProtections ...
var DataProtections = map[string]appstoreconnect.CapabilityOptionKey{
	"NSFileProtectionComplete":                             appstoreconnect.CompleteProtection,
	"NSFileProtectionCompleteUnlessOpen":                   appstoreconnect.ProtectedUnlessOpen,
	"NSFileProtectionCompleteUntilFirstUserAuthentication": appstoreconnect.ProtectedUntilFirstUserAuth,
}

// Capability ...
func (e Entitlement) Capability() (*appstoreconnect.BundleIDCapability, error) {
	if len(e) == 0 {
		return nil, nil
	}

	// List of capabilities that need to be configured manually on the Developer portal
	capabilitiesWarn := map[appstoreconnect.CapabilityType]string{
		appstoreconnect.AppGroups:       "App Groups",
		appstoreconnect.ApplePay:        "Apple Pay Payment Processing",
		appstoreconnect.ICloud:          "iCloud",
		appstoreconnect.SignInWithApple: "Sign In with Apple",
	}

	// List of capabilities that the API does not support and prevent autoprovisioning
	capabilitiesError := map[appstoreconnect.CapabilityType]string{
		appstoreconnect.OnDemandInstallCapable:       "On Demand Install Capable (App Clips)",
		appstoreconnect.ParentApplicationIdentifiers: "Parent Bundle ID",
	}

	entKey := serialized.Object(e).Keys()[0]

	capType, ok := appstoreconnect.ServiceTypeByKey[entKey]
	if !ok {
		return nil, errors.New("unknown entitlement key: " + entKey)
	}

	if capType == appstoreconnect.Ignored {
		return nil, nil
	}

	capSetts := []appstoreconnect.CapabilitySetting{}
	if capType == appstoreconnect.ICloud {
		capSett := appstoreconnect.CapabilitySetting{
			Key: appstoreconnect.IcloudVersion,
			Options: []appstoreconnect.CapabilityOption{
				{
					Key: appstoreconnect.Xcode6,
				},
			},
		}
		capSetts = append(capSetts, capSett)
	} else if capType == appstoreconnect.DataProtection {
		entVal, err := serialized.Object(e).String(entKey)
		if err != nil {
			return nil, errors.New("no entitlements value for key: " + entKey)
		}

		key, ok := DataProtections[entVal]
		if !ok {
			return nil, errors.New("no data protection level found for entitlement value: " + entVal)
		}

		capSett := appstoreconnect.CapabilitySetting{
			Key: appstoreconnect.DataProtectionPermissionLevel,
			Options: []appstoreconnect.CapabilityOption{
				{
					Key: key,
				},
			},
		}
		capSetts = append(capSetts, capSett)
	} else if capType == appstoreconnect.SignInWithApple {
		capSett := appstoreconnect.CapabilitySetting{
			Key: appstoreconnect.AppleIDAuthAppConsent,
			Options: []appstoreconnect.CapabilityOption{
				{
					Key: "PRIMARY_APP_CONSENT",
				},
			},
		}
		capSetts = append(capSetts, capSett)
	}

	if capName, contains := capabilitiesWarn[capType]; contains {
		log.Warnf("This will enable the \"%s\" capability but details will have to be configured manually using the Apple Developer Portal", capName)
	}

	if capName, contains := capabilitiesError[capType]; contains {
		return nil, fmt.Errorf("step does not support creating an application identifier using the \"%s\" capability, please add your Application Identifier manually using the Apple Developer Portal", capName)
	}

	return &appstoreconnect.BundleIDCapability{
		Attributes: appstoreconnect.BundleIDCapabilityAttributes{
			CapabilityType: capType,
			Settings:       capSetts,
		},
	}, nil
}

// IsProfileAttached returns an error if an entitlement does not match a Capability but needs to be addded to the profile
// as an additional entitlement, after submitting a request to Apple.
func (e Entitlement) IsProfileAttached() bool {
	if len(e) == 0 {
		return false
	}
	entKey := serialized.Object(e).Keys()[0]

	capType, ok := appstoreconnect.ServiceTypeByKey[entKey]
	return ok && capType == appstoreconnect.ProfileAttachedEntitlement
}

// AppearsOnDeveloperPortal reports whether the given (project) Entitlement needs to be registered on Apple Developer Portal or not.
// List of services, to be registered: https://developer.apple.com/documentation/appstoreconnectapi/capabilitytype.
func (e Entitlement) AppearsOnDeveloperPortal() bool {
	if len(e) == 0 {
		return false
	}
	entKey := serialized.Object(e).Keys()[0]

	capType, ok := appstoreconnect.ServiceTypeByKey[entKey]
	return ok && capType != appstoreconnect.Ignored && capType != appstoreconnect.ProfileAttachedEntitlement
}

// Equal ...
func (e Entitlement) Equal(cap appstoreconnect.BundleIDCapability, allEntitlements Entitlements) (bool, error) {
	if len(e) == 0 {
		return false, nil
	}

	entKey := serialized.Object(e).Keys()[0]

	capType, ok := appstoreconnect.ServiceTypeByKey[entKey]
	if !ok {
		return false, errors.New("unknown entitlement key: " + entKey)
	}

	if cap.Attributes.CapabilityType != capType {
		return false, nil
	}

	if capType == appstoreconnect.ICloud {
		return iCloudEquals(allEntitlements, cap)
	} else if capType == appstoreconnect.DataProtection {
		entVal, err := serialized.Object(e).String(entKey)
		if err != nil {
			return false, err
		}
		return dataProtectionEquals(entVal, cap)
	}

	return true, nil
}

func (e Entitlements) iCloudServices() (iCloudDocuments, iCloudKit, keyValueStorage bool, err error) {
	v, err := serialized.Object(e).String("com.apple.developer.ubiquity-kvstore-identifier")
	if err != nil && !serialized.IsKeyNotFoundError(err) {
		return false, false, false, err
	}
	keyValueStorage = v != ""

	iCloudServices, err := serialized.Object(e).StringSlice("com.apple.developer.icloud-services")
	if err != nil && !serialized.IsKeyNotFoundError(err) {
		return false, false, false, err
	}

	if len(iCloudServices) > 0 {
		iCloudDocuments = slices.Contains(iCloudServices, "CloudDocuments")
		iCloudKit = slices.Contains(iCloudServices, "CloudKit")
	}
	return
}

// ICloudContainers returns the list of iCloud containers
func (e Entitlements) ICloudContainers() ([]string, error) {
	usesDocuments, usesCloudKit, _, err := e.iCloudServices()
	if err != nil && !serialized.IsKeyNotFoundError(err) {
		return nil, err
	}

	if !usesCloudKit && !usesDocuments {
		return nil, nil
	}

	containers, err := serialized.Object(e).StringSlice(ICloudIdentifiersEntitlementKey)
	if err != nil && !serialized.IsKeyNotFoundError(err) {
		return nil, err
	}
	return containers, nil
}

func iCloudEquals(ent Entitlements, cap appstoreconnect.BundleIDCapability) (bool, error) {
	documents, cloudKit, kvStorage, err := ent.iCloudServices()
	if err != nil {
		return false, err
	}

	if len(cap.Attributes.Settings) != 1 {
		return false, nil
	}

	capSett := cap.Attributes.Settings[0]
	if capSett.Key != appstoreconnect.IcloudVersion {
		return false, nil
	}
	if len(capSett.Options) != 1 {
		return false, nil
	}

	capSettOpt := capSett.Options[0]
	if (documents || cloudKit || kvStorage) && capSettOpt.Key != appstoreconnect.Xcode6 {
		return false, nil
	}
	return true, nil
}

func dataProtectionEquals(entVal string, cap appstoreconnect.BundleIDCapability) (bool, error) {
	key, ok := DataProtections[entVal]
	if !ok {
		return false, errors.New("no data protection level found for entitlement value: " + entVal)
	}

	if len(cap.Attributes.Settings) != 1 {
		return false, nil
	}

	capSett := cap.Attributes.Settings[0]
	if capSett.Key != appstoreconnect.DataProtectionPermissionLevel {
		return false, nil
	}
	if len(capSett.Options) != 1 {
		return false, nil
	}

	capSettOpt := capSett.Options[0]
	if capSettOpt.Key != key {
		return false, nil
	}
	return true, nil
}
