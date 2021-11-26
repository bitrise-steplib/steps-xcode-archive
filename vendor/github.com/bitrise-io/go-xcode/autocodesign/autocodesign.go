// Package autocodesign is a framework for automatic code signing.
//
// Contains common types, interfaces and logic needed for codesigning.
// Parsing an Xcode project or archive and applying settings is not part of the package, for modularity.
package autocodesign

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-xcode/autocodesign/devportalclient/appstoreconnect"
	"github.com/bitrise-io/go-xcode/certificateutil"
	"github.com/bitrise-io/go-xcode/devportalservice"
	"github.com/bitrise-io/go-xcode/xcodeproject/serialized"
)

// Profile represents a provisioning profiles
type Profile interface {
	ID() string
	Attributes() appstoreconnect.ProfileAttributes
	CertificateIDs() ([]string, error)
	DeviceIDs() ([]string, error)
	BundleID() (appstoreconnect.BundleID, error)
	Entitlements() (Entitlements, error)
}

// AppCodesignAssets is the result of ensuring codesigning assets
type AppCodesignAssets struct {
	ArchivableTargetProfilesByBundleID map[string]Profile
	UITestTargetProfilesByBundleID     map[string]Profile
	Certificate                        certificateutil.CertificateInfoModel
}

// Platform ...
type Platform string

// Const
const (
	IOS   Platform = "iOS"
	TVOS  Platform = "tvOS"
	MacOS Platform = "macOS"
)

// DistributionType ...
type DistributionType string

// DistributionTypes ...
var (
	Development DistributionType = "development"
	AppStore    DistributionType = "app-store"
	AdHoc       DistributionType = "ad-hoc"
	Enterprise  DistributionType = "enterprise"
)

// Entitlement ...
type Entitlement serialized.Object

// Entitlements is all the entitlements that are contained in a target or profile
type Entitlements serialized.Object

// Certificate is certificate present on Apple App Store Connect API, could match a local certificate
type Certificate struct {
	CertificateInfo certificateutil.CertificateInfoModel
	ID              string
}

// DevPortalClient abstract away the Apple Developer Portal API
type DevPortalClient interface {
	QueryCertificateBySerial(serial big.Int) (Certificate, error)
	QueryAllIOSCertificates() (map[appstoreconnect.CertificateType][]Certificate, error)

	ListDevices(UDID string, platform appstoreconnect.DevicePlatform) ([]appstoreconnect.Device, error)
	RegisterDevice(testDevice devportalservice.TestDevice) (*appstoreconnect.Device, error)

	FindProfile(name string, profileType appstoreconnect.ProfileType) (Profile, error)
	DeleteProfile(id string) error
	CreateProfile(name string, profileType appstoreconnect.ProfileType, bundleID appstoreconnect.BundleID, certificateIDs []string, deviceIDs []string) (Profile, error)

	FindBundleID(bundleIDIdentifier string) (*appstoreconnect.BundleID, error)
	CheckBundleIDEntitlements(bundleID appstoreconnect.BundleID, appEntitlements Entitlements) error
	SyncBundleID(bundleID appstoreconnect.BundleID, appEntitlements Entitlements) error
	CreateBundleID(bundleIDIdentifier, appIDName string) (*appstoreconnect.BundleID, error)
}

// AssetWriter ...
type AssetWriter interface {
	Write(codesignAssetsByDistributionType map[DistributionType]AppCodesignAssets) error
	InstallCertificate(certificate certificateutil.CertificateInfoModel) error
}

// AppLayout contains codesigning related settings that are needed to ensure codesigning files
type AppLayout struct {
	TeamID                                 string
	Platform                               Platform
	EntitlementsByArchivableTargetBundleID map[string]Entitlements
	UITestTargetBundleIDs                  []string
}

// CertificateProvider returns codesigning certificates (with private key)
type CertificateProvider interface {
	GetCertificates() ([]certificateutil.CertificateInfoModel, error)
}

// CodesignAssetsOpts are codesigning related paramters that are not specified by the project (or archive)
type CodesignAssetsOpts struct {
	DistributionType       DistributionType
	BitriseTestDevices     []devportalservice.TestDevice
	MinProfileValidityDays int
	VerboseLog             bool
}

// CodesignAssetManager ...
type CodesignAssetManager interface {
	EnsureCodesignAssets(appLayout AppLayout, opts CodesignAssetsOpts) (map[DistributionType]AppCodesignAssets, error)
}

type codesignAssetManager struct {
	devPortalClient     DevPortalClient
	certificateProvider CertificateProvider
	assetWriter         AssetWriter
}

// NewCodesignAssetManager ...
func NewCodesignAssetManager(devPortalClient DevPortalClient, certificateProvider CertificateProvider, assetWriter AssetWriter) CodesignAssetManager {
	return codesignAssetManager{
		devPortalClient:     devPortalClient,
		certificateProvider: certificateProvider,
		assetWriter:         assetWriter,
	}
}

// EnsureCodesignAssets is the main entry point of the codesigning logic
func (m codesignAssetManager) EnsureCodesignAssets(appLayout AppLayout, opts CodesignAssetsOpts) (map[DistributionType]AppCodesignAssets, error) {
	fmt.Println()
	log.Infof("Downloading certificates")

	certs, err := m.certificateProvider.GetCertificates()
	if err != nil {
		return nil, fmt.Errorf("failed to download certificates: %w", err)
	}
	if len(certs) > 0 {
		log.Printf("%d certificates downloaded:", len(certs))
		for _, cert := range certs {
			log.Printf("- %s", cert.String())
		}
	} else {
		log.Warnf("No certificates found")
	}

	signUITestTargets := len(appLayout.UITestTargetBundleIDs) > 0
	certsByType, distrTypes, err := selectCertificatesAndDistributionTypes(
		m.devPortalClient,
		certs,
		opts.DistributionType,
		appLayout.TeamID,
		signUITestTargets,
		opts.VerboseLog,
	)
	if err != nil {
		return nil, err
	}

	var devPortalDeviceIDs []string
	if DistributionTypeRequiresDeviceList(distrTypes) {
		var err error
		devPortalDeviceIDs, err = EnsureTestDevices(m.devPortalClient, opts.BitriseTestDevices, appLayout.Platform)
		if err != nil {
			return nil, fmt.Errorf("failed to ensure test devices: %w", err)
		}
	}

	// Ensure Profiles
	codesignAssetsByDistributionType, err := ensureProfiles(m.devPortalClient, distrTypes, certsByType, appLayout, devPortalDeviceIDs, opts.MinProfileValidityDays)
	if err != nil {
		switch {
		case errors.As(err, &ErrAppClipAppID{}):
			log.Warnf("Can't create Application Identifier for App Clip targets.")
			log.Warnf("Please generate the Application Identifier manually on Apple Developer Portal, after that the Step will continue working.")
		case errors.As(err, &ErrAppClipAppIDWithAppleSigning{}):
			log.Warnf("Can't manage Application Identifier for App Clip target with 'Sign In With Apple' capability.")
			log.Warnf("Please configure Capabilities on Apple Developer Portal for App Clip target manually, after that the Step will continue working.")
		}

		return nil, fmt.Errorf("failed to ensure profiles: %w", err)
	}

	// Install certificates and profiles
	fmt.Println()
	log.Infof("Install certificates and profiles")
	if err := m.assetWriter.Write(codesignAssetsByDistributionType); err != nil {
		return nil, fmt.Errorf("failed to install codesigning files: %s", err)
	}

	return codesignAssetsByDistributionType, nil
}
