package exportoptions

import (
	"fmt"

	"github.com/bitrise-io/go-xcode/utility"
)

// CompileBitcodeKey ...
const CompileBitcodeKey = "compileBitcode"

// CompileBitcodeDefault ...
const CompileBitcodeDefault = true

// EmbedOnDemandResourcesAssetPacksInBundleKey ...
const EmbedOnDemandResourcesAssetPacksInBundleKey = "embedOnDemandResourcesAssetPacksInBundle"

// EmbedOnDemandResourcesAssetPacksInBundleDefault ...
const EmbedOnDemandResourcesAssetPacksInBundleDefault = true

// ICloudContainerEnvironmentKey ...
const ICloudContainerEnvironmentKey = "iCloudContainerEnvironment"

const (
	// ICloudContainerEnvironmentDevelopment ...
	ICloudContainerEnvironmentDevelopment ICloudContainerEnvironment = "Development"
	// ICloudContainerEnvironmentProduction ...
	ICloudContainerEnvironmentProduction ICloudContainerEnvironment = "Production"
)

// ICloudContainerEnvironment ...
type ICloudContainerEnvironment string

// DistributionBundleIdentifier ...
const DistributionBundleIdentifier = "distributionBundleIdentifier"

// ManifestKey ...
const ManifestKey = "manifest"

// ManifestAppURLKey ...
const ManifestAppURLKey = "appURL"

// ManifestDisplayImageURLKey ...
const ManifestDisplayImageURLKey = "displayImageURL"

// ManifestFullSizeImageURLKey ...
const ManifestFullSizeImageURLKey = "fullSizeImageURL"

// ManifestAssetPackManifestURLKey ...
const ManifestAssetPackManifestURLKey = "assetPackManifestURL"

const xcode15Dot3BuildVersion = "15E204a"

// Manifest ...
type Manifest struct {
	AppURL               string
	DisplayImageURL      string
	FullSizeImageURL     string
	AssetPackManifestURL string
}

// IsEmpty ...
func (manifest Manifest) IsEmpty() bool {
	return (manifest.AppURL == "" && manifest.DisplayImageURL == "" && manifest.FullSizeImageURL == "" && manifest.AssetPackManifestURL == "")
}

// ToHash ...
func (manifest Manifest) ToHash() map[string]string {
	hash := map[string]string{}
	if manifest.AppURL != "" {
		hash[ManifestAppURLKey] = manifest.AppURL
	}
	if manifest.DisplayImageURL != "" {
		hash[ManifestDisplayImageURLKey] = manifest.DisplayImageURL
	}
	if manifest.FullSizeImageURL != "" {
		hash[ManifestFullSizeImageURLKey] = manifest.FullSizeImageURL
	}
	if manifest.AssetPackManifestURL != "" {
		hash[ManifestAssetPackManifestURLKey] = manifest.AssetPackManifestURL
	}
	return hash
}

const MethodKey = "method"

const (
	MethodAppStore Method = "app-store"
	MethodAdHoc Method = "ad-hoc"
	MethodPackage Method = "package"
	MethodEnterprise Method = "enterprise"
	MethodDevelopment Method = "development"
	MethodDeveloperID Method = "developer-id"
	MethodDebugging Method = "debugging"
	MethodAppStoreConnect Method = "app-store-connect"
	MethodReleaseTesting Method = "release-testing"
	MethodDefault Method = MethodDevelopment
)

// Method ...
type Method string

// ParseMethod ...
func ParseMethod(method string) (Method, error) {
	// TODO: Print warning if old export methods are used with Xcode 15.3 or newer
	newExportMethods, err := utility.XcodeBuildVersionIsAtLeast(xcode15Dot3BuildVersion)
	if err != nil {
		return Method(""), fmt.Errorf("check Xcode version: %s", err)
	}

	switch method {
	case "app-store":
		if newExportMethods {
			return MethodAppStoreConnect, nil
		} else {
			return MethodAppStore, nil
		}
	case "ad-hoc":
		if newExportMethods {
			return MethodReleaseTesting, nil
		} else {
			return MethodAdHoc, nil
		}
	case "package":
		return MethodPackage, nil
	case "enterprise":
		return MethodEnterprise, nil
	case "development":
		if newExportMethods {
			return MethodDebugging, nil
		} else {
			return MethodDevelopment, nil
		}
	case "developer-id":
		return MethodDeveloperID, nil
	default:
		return Method(""), fmt.Errorf("unkown method (%s)", method)
	}
}

// OnDemandResourcesAssetPacksBaseURLKey ....
const OnDemandResourcesAssetPacksBaseURLKey = "onDemandResourcesAssetPacksBaseURL"

// TeamIDKey ...
const TeamIDKey = "teamID"

// ThinningKey ...
const ThinningKey = "thinning"

const (
	// ThinningNone ...
	ThinningNone = "none"
	// ThinningThinForAllVariants ...
	ThinningThinForAllVariants = "thin-for-all-variants"
	// ThinningDefault ...
	ThinningDefault = ThinningNone
)

// UploadBitcodeKey ....
const UploadBitcodeKey = "uploadBitcode"

// UploadBitcodeDefault ...
const UploadBitcodeDefault = true

// UploadSymbolsKey ...
const UploadSymbolsKey = "uploadSymbols"

// UploadSymbolsDefault ...
const UploadSymbolsDefault = true

const (
	manageAppVersionKey     = "manageAppVersionAndBuildNumber"
	manageAppVersionDefault = true
)

// ProvisioningProfilesKey ...
const ProvisioningProfilesKey = "provisioningProfiles"

// SigningCertificateKey ...
const SigningCertificateKey = "signingCertificate"

// InstallerSigningCertificateKey ...
const InstallerSigningCertificateKey = "installerSigningCertificate"

// SigningStyleKey ...
const SigningStyleKey = "signingStyle"
