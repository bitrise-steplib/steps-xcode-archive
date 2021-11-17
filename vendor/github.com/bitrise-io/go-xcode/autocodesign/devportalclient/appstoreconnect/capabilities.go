package appstoreconnect

import (
	"net/http"
	"strings"
)

// BundleIDCapabilitiesEndpoint ...
const BundleIDCapabilitiesEndpoint = "bundleIdCapabilities"

// CapabilityType ...
type CapabilityType string

// CapabilityTypes ...
const (
	Ignored                        CapabilityType = "-ignored-"
	ProfileAttachedEntitlement     CapabilityType = "-profile-attached-"
	ICloud                         CapabilityType = "ICLOUD"
	InAppPurchase                  CapabilityType = "IN_APP_PURCHASE"
	GameCenter                     CapabilityType = "GAME_CENTER"
	PushNotifications              CapabilityType = "PUSH_NOTIFICATIONS"
	Wallet                         CapabilityType = "WALLET"
	InterAppAudio                  CapabilityType = "INTER_APP_AUDIO"
	Maps                           CapabilityType = "MAPS"
	AssociatedDomains              CapabilityType = "ASSOCIATED_DOMAINS"
	PersonalVPN                    CapabilityType = "PERSONAL_VPN"
	AppGroups                      CapabilityType = "APP_GROUPS"
	Healthkit                      CapabilityType = "HEALTHKIT"
	Homekit                        CapabilityType = "HOMEKIT"
	WirelessAccessoryConfiguration CapabilityType = "WIRELESS_ACCESSORY_CONFIGURATION"
	ApplePay                       CapabilityType = "APPLE_PAY"
	DataProtection                 CapabilityType = "DATA_PROTECTION"
	Sirikit                        CapabilityType = "SIRIKIT"
	NetworkExtensions              CapabilityType = "NETWORK_EXTENSIONS"
	Multipath                      CapabilityType = "MULTIPATH"
	HotSpot                        CapabilityType = "HOT_SPOT"
	NFCTagReading                  CapabilityType = "NFC_TAG_READING"
	Classkit                       CapabilityType = "CLASSKIT"
	AutofillCredentialProvider     CapabilityType = "AUTOFILL_CREDENTIAL_PROVIDER"
	AccessWIFIInformation          CapabilityType = "ACCESS_WIFI_INFORMATION"
	NetworkCustomProtocol          CapabilityType = "NETWORK_CUSTOM_PROTOCOL"
	CoremediaHLSLowLatency         CapabilityType = "COREMEDIA_HLS_LOW_LATENCY"
	SystemExtensionInstall         CapabilityType = "SYSTEM_EXTENSION_INSTALL"
	UserManagement                 CapabilityType = "USER_MANAGEMENT"
	SignInWithApple                CapabilityType = "APPLE_ID_AUTH"
	ParentApplicationIdentifiers   CapabilityType = "ODIC_PARENT_BUNDLEID"
	OnDemandInstallCapable         CapabilityType = "ON_DEMAND_INSTALL_CAPABLE"
)

// Entitlement keys ...
const (
	ParentApplicationIdentifierEntitlementKey = "com.apple.developer.parent-application-identifiers"
	SignInWithAppleEntitlementKey             = "com.apple.developer.applesignin"
)

// ServiceTypeByKey ...
var ServiceTypeByKey = map[string]CapabilityType{
	"com.apple.security.application-groups":                                    AppGroups,
	"com.apple.developer.in-app-payments":                                      ApplePay,
	"com.apple.developer.associated-domains":                                   AssociatedDomains,
	"com.apple.developer.healthkit":                                            Healthkit,
	"com.apple.developer.homekit":                                              Homekit,
	"com.apple.developer.networking.HotspotConfiguration":                      HotSpot,
	"com.apple.InAppPurchase":                                                  InAppPurchase,
	"inter-app-audio":                                                          InterAppAudio,
	"com.apple.developer.networking.multipath":                                 Multipath,
	"com.apple.developer.networking.networkextension":                          NetworkExtensions,
	"com.apple.developer.nfc.readersession.formats":                            NFCTagReading,
	"com.apple.developer.networking.vpn.api":                                   PersonalVPN,
	"aps-environment":                                                          PushNotifications,
	"com.apple.developer.siri":                                                 Sirikit,
	SignInWithAppleEntitlementKey:                                              SignInWithApple,
	"com.apple.developer.on-demand-install-capable":                            OnDemandInstallCapable,
	"com.apple.developer.pass-type-identifiers":                                Wallet,
	"com.apple.external-accessory.wireless-configuration":                      WirelessAccessoryConfiguration,
	"com.apple.developer.default-data-protection":                              DataProtection,
	"com.apple.developer.icloud-services":                                      ICloud,
	"com.apple.developer.authentication-services.autofill-credential-provider": AutofillCredentialProvider,
	"com.apple.developer.networking.wifi-info":                                 AccessWIFIInformation,
	"com.apple.developer.ClassKit-environment":                                 Classkit,
	"com.apple.developer.coremedia.hls.low-latency":                            CoremediaHLSLowLatency,
	// does not appear on developer portal
	"com.apple.developer.icloud-container-identifiers":   Ignored,
	"com.apple.developer.ubiquity-container-identifiers": Ignored,
	ParentApplicationIdentifierEntitlementKey:            Ignored,
	// These are entitlements not supported via the API and this step,
	// profile needs to be manually generated on Apple Developer Portal.
	"com.apple.developer.contacts.notes":         ProfileAttachedEntitlement,
	"com.apple.developer.carplay-audio":          ProfileAttachedEntitlement,
	"com.apple.developer.carplay-communication":  ProfileAttachedEntitlement,
	"com.apple.developer.carplay-charging":       ProfileAttachedEntitlement,
	"com.apple.developer.carplay-maps":           ProfileAttachedEntitlement,
	"com.apple.developer.carplay-parking":        ProfileAttachedEntitlement,
	"com.apple.developer.carplay-quick-ordering": ProfileAttachedEntitlement,
	"com.apple.developer.exposure-notification":  ProfileAttachedEntitlement,
}

// CapabilitySettingAllowedInstances ...
type CapabilitySettingAllowedInstances string

// AllowedInstances ...
const (
	Entry    CapabilitySettingAllowedInstances = "ENTRY"
	Single   CapabilitySettingAllowedInstances = "SINGLE"
	Multiple CapabilitySettingAllowedInstances = "MULTIPLE"
)

// CapabilitySettingKey ...
type CapabilitySettingKey string

// CapabilitySettingKeys
const (
	IcloudVersion                 CapabilitySettingKey = "ICLOUD_VERSION"
	DataProtectionPermissionLevel CapabilitySettingKey = "DATA_PROTECTION_PERMISSION_LEVEL"
	AppleIDAuthAppConsent         CapabilitySettingKey = "APPLE_ID_AUTH_APP_CONSENT"
	AppGroupIdentifiers           CapabilitySettingKey = "APP_GROUP_IDENTIFIERS"
)

// CapabilityOptionKey ...
type CapabilityOptionKey string

// CapabilityOptionKeys ...
const (
	Xcode5                      CapabilityOptionKey = "XCODE_5"
	Xcode6                      CapabilityOptionKey = "XCODE_6"
	CompleteProtection          CapabilityOptionKey = "COMPLETE_PROTECTION"
	ProtectedUnlessOpen         CapabilityOptionKey = "PROTECTED_UNLESS_OPEN"
	ProtectedUntilFirstUserAuth CapabilityOptionKey = "PROTECTED_UNTIL_FIRST_USER_AUTH"
)

// CapabilityOption ...
type CapabilityOption struct {
	Description      string              `json:"description,omitempty"`
	Enabled          bool                `json:"enabled,omitempty"`
	EnabledByDefault bool                `json:"enabledByDefault,omitempty"`
	Key              CapabilityOptionKey `json:"key,omitempty"`
	Name             string              `json:"name,omitempty"`
	SupportsWildcard bool                `json:"supportsWildcard,omitempty"`
}

// CapabilitySetting ...
type CapabilitySetting struct {
	AllowedInstances CapabilitySettingAllowedInstances `json:"allowedInstances,omitempty"`
	Description      string                            `json:"description,omitempty"`
	EnabledByDefault bool                              `json:"enabledByDefault,omitempty"`
	Key              CapabilitySettingKey              `json:"key,omitempty"`
	Name             string                            `json:"name,omitempty"`
	Options          []CapabilityOption                `json:"options,omitempty"`
	Visible          bool                              `json:"visible,omitempty"`
	MinInstances     int                               `json:"minInstances,omitempty"`
}

//
// BundleIDCapabilityCreateRequest

// BundleIDCapabilityCreateRequestDataAttributes ...
type BundleIDCapabilityCreateRequestDataAttributes struct {
	CapabilityType CapabilityType      `json:"capabilityType"`
	Settings       []CapabilitySetting `json:"settings"`
}

// BundleIDCapabilityCreateRequestDataRelationships ...
type BundleIDCapabilityCreateRequestDataRelationships struct {
	BundleID BundleIDCapabilityCreateRequestDataRelationshipsBundleID `json:"bundleId"`
}

// BundleIDCapabilityCreateRequestDataRelationshipsBundleID ...
type BundleIDCapabilityCreateRequestDataRelationshipsBundleID struct {
	Data BundleIDCapabilityCreateRequestDataRelationshipsBundleIDData `json:"data"`
}

// BundleIDCapabilityCreateRequestDataRelationshipsBundleIDData ...
type BundleIDCapabilityCreateRequestDataRelationshipsBundleIDData struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// BundleIDCapabilityCreateRequestData ...
type BundleIDCapabilityCreateRequestData struct {
	Attributes    BundleIDCapabilityCreateRequestDataAttributes    `json:"attributes"`
	Relationships BundleIDCapabilityCreateRequestDataRelationships `json:"relationships"`
	Type          string                                           `json:"type"`
}

// BundleIDCapabilityCreateRequest ...
type BundleIDCapabilityCreateRequest struct {
	Data BundleIDCapabilityCreateRequestData `json:"data"`
}

//
// BundleIDCapabilityUpdateRequest

// BundleIDCapabilityUpdateRequestDataAttributes ...
type BundleIDCapabilityUpdateRequestDataAttributes struct {
	CapabilityType CapabilityType      `json:"capabilityType"`
	Settings       []CapabilitySetting `json:"settings"`
}

// BundleIDCapabilityUpdateRequestData ...
type BundleIDCapabilityUpdateRequestData struct {
	Attributes BundleIDCapabilityUpdateRequestDataAttributes `json:"attributes"`
	ID         string                                        `json:"id"`
	Type       string                                        `json:"type"`
}

// BundleIDCapabilityUpdateRequest ...
type BundleIDCapabilityUpdateRequest struct {
	Data BundleIDCapabilityUpdateRequestData `json:"data"`
}

// BundleIDCapabilityAttributes ...
type BundleIDCapabilityAttributes struct {
	CapabilityType CapabilityType      `json:"capabilityType"`
	Settings       []CapabilitySetting `json:"settings"`
}

// BundleIDCapability ...
type BundleIDCapability struct {
	Attributes BundleIDCapabilityAttributes
	ID         string `json:"id"`
	Type       string `json:"type"`
}

// BundleIDCapabilityResponse ...
type BundleIDCapabilityResponse struct {
	Data BundleIDCapability `json:"data"`
}

// BundleIDCapabilitiesResponse ...
type BundleIDCapabilitiesResponse struct {
	Data []BundleIDCapability `json:"data"`
}

// EnableCapability ...
func (s ProvisioningService) EnableCapability(body BundleIDCapabilityCreateRequest) (*BundleIDCapabilityResponse, error) {
	req, err := s.client.NewRequest(http.MethodPost, BundleIDCapabilitiesEndpoint, body)
	if err != nil {
		return nil, err
	}

	r := &BundleIDCapabilityResponse{}
	if _, err := s.client.Do(req, r); err != nil {
		return nil, err
	}

	return r, nil
}

// UpdateCapability ...
func (s ProvisioningService) UpdateCapability(id string, body BundleIDCapabilityUpdateRequest) (*BundleIDCapabilityResponse, error) {
	req, err := s.client.NewRequest(http.MethodPatch, BundleIDCapabilitiesEndpoint+"/"+id, body)
	if err != nil {
		return nil, err
	}

	r := &BundleIDCapabilityResponse{}
	if _, err := s.client.Do(req, r); err != nil {
		return nil, err
	}
	return r, nil
}

// Capabilities ...
func (s ProvisioningService) Capabilities(relationshipLink string) (*BundleIDCapabilitiesResponse, error) {
	endpoint := strings.TrimPrefix(relationshipLink, baseURL+apiVersion)
	req, err := s.client.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	r := &BundleIDCapabilitiesResponse{}
	if _, err := s.client.Do(req, r); err != nil {
		return nil, err
	}

	return r, nil
}
