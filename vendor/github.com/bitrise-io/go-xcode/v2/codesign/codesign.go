package codesign

import (
	"fmt"
	"time"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/appleauth"
	"github.com/bitrise-io/go-xcode/certificateutil"
	"github.com/bitrise-io/go-xcode/devportalservice"
	"github.com/bitrise-io/go-xcode/v2/autocodesign"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/devportalclient"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/devportalclient/appstoreconnect"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/projectmanager"
)

// AuthType ...
type AuthType int

const (
	// APIKeyAuth ...
	APIKeyAuth AuthType = iota
	// AppleIDAuth ...
	AppleIDAuth
)

type codeSigningStrategy int

const (
	codeSigningXcode codeSigningStrategy = iota
	codeSigningBitriseAPIKey
	codeSigningBitriseAppleID
)

// Opts ...
type Opts struct {
	AuthType                   AuthType
	ShouldConsiderXcodeSigning bool
	TeamID                     string

	ExportMethod      autocodesign.DistributionType
	XcodeMajorVersion int

	RegisterTestDevices    bool
	SignUITests            bool
	MinDaysProfileValidity int
	IsVerboseLog           bool
}

// Manager ...
type Manager struct {
	opts Opts

	appleAuthCredentials      appleauth.Credentials
	bitriseTestDevices        []devportalservice.TestDevice
	devPortalClientFactory    devportalclient.Factory
	certDownloader            autocodesign.CertificateProvider
	fallbackProfileDownloader autocodesign.ProfileProvider
	assetInstaller            autocodesign.AssetWriter
	localCodeSignAssetManager autocodesign.LocalCodeSignAssetManager

	detailsProvider DetailsProvider
	assetWriter     AssetWriter

	logger log.Logger
}

// NewManagerWithArchive creates a codesign manager, which reads the code signing asset requirements from an XCArchive file.
func NewManagerWithArchive(
	opts Opts,
	appleAuth appleauth.Credentials,
	bitriseTestDevices []devportalservice.TestDevice,
	clientFactory devportalclient.Factory,
	certDownloader autocodesign.CertificateProvider,
	fallbackProfileDownloader autocodesign.ProfileProvider,
	assetInstaller autocodesign.AssetWriter,
	localCodeSignAssetManager autocodesign.LocalCodeSignAssetManager,
	archive Archive,
	logger log.Logger,
) Manager {
	return Manager{
		opts:                      opts,
		appleAuthCredentials:      appleAuth,
		bitriseTestDevices:        bitriseTestDevices,
		devPortalClientFactory:    clientFactory,
		certDownloader:            certDownloader,
		fallbackProfileDownloader: fallbackProfileDownloader,
		assetInstaller:            assetInstaller,
		localCodeSignAssetManager: localCodeSignAssetManager,
		detailsProvider:           archive,
		logger:                    logger,
	}
}

// NewManagerWithProject creates a codesign manager, which reads the code signing asset requirements from an Xcode Project.
func NewManagerWithProject(
	opts Opts,
	appleAuth appleauth.Credentials,
	bitriseTestDevices []devportalservice.TestDevice,
	clientFactory devportalclient.Factory,
	certDownloader autocodesign.CertificateProvider,
	fallbackProfileDownloader autocodesign.ProfileProvider,
	assetInstaller autocodesign.AssetWriter,
	localCodeSignAssetManager autocodesign.LocalCodeSignAssetManager,
	project projectmanager.Project,
	logger log.Logger,
) Manager {
	return Manager{
		opts:                      opts,
		appleAuthCredentials:      appleAuth,
		bitriseTestDevices:        bitriseTestDevices,
		devPortalClientFactory:    clientFactory,
		certDownloader:            certDownloader,
		fallbackProfileDownloader: fallbackProfileDownloader,
		assetInstaller:            assetInstaller,
		localCodeSignAssetManager: localCodeSignAssetManager,
		detailsProvider:           project,
		assetWriter:               project,
		logger:                    logger,
	}
}

// DetailsProvider ...
type DetailsProvider interface {
	IsSigningManagedAutomatically() (bool, error)
	Platform() (autocodesign.Platform, error)
	GetAppLayout(uiTestTargets bool) (autocodesign.AppLayout, error)
}

// AssetWriter ...
type AssetWriter interface {
	ForceCodesignAssets(distribution autocodesign.DistributionType, codesignAssetsByDistributionType map[autocodesign.DistributionType]autocodesign.AppCodesignAssets) error
}

// PrepareCodesigning selects a suitable code signing strategy based on the step and project configuration,
// then downloads code signing assets (profiles, certificates) and registers test devices if needed
func (m *Manager) PrepareCodesigning() (*devportalservice.APIKeyConnection, error) {
	strategy, reason, err := m.selectCodeSigningStrategy(m.appleAuthCredentials)
	if err != nil {
		m.logger.Warnf("%s", err)
	}

	switch strategy {
	case codeSigningXcode:
		{
			m.logger.Println()
			m.logger.Infof("Code signing asset management with xcodebuild")
			m.logger.Printf("Reason: %s", reason)
			m.logger.Println()
			m.logger.TInfof("Downloading certificates...")
			certificates, err := m.downloadCertificates()
			if err != nil {
				return nil, err
			}

			if err := m.validateCertificatesForXcodeManagedSigning(certificates); err != nil {
				return nil, err
			}

			m.logger.Println()
			m.logger.TInfof("Installing certificates...")
			if err := m.installCertificates(certificates); err != nil {
				return nil, err
			}

			needsTestDevices := autocodesign.DistributionTypeRequiresDeviceList([]autocodesign.DistributionType{m.opts.ExportMethod})
			if needsTestDevices && m.opts.RegisterTestDevices && len(m.bitriseTestDevices) != 0 {
				if err := m.registerTestDevices(m.appleAuthCredentials, m.bitriseTestDevices); err != nil {
					return nil, err
				}
			}

			return m.appleAuthCredentials.APIKey, nil
		}
	case codeSigningBitriseAPIKey, codeSigningBitriseAppleID:
		{
			m.logger.Println()
			m.logger.Infof("Code signing asset management by Bitrise")
			m.logger.Printf("Reason: %s", reason)
			if err := m.prepareCodeSigningWithBitrise(m.appleAuthCredentials, m.bitriseTestDevices); err != nil {
				return nil, err
			}

			return nil, nil
		}
	default:
		return nil, fmt.Errorf("unknown code sign strategy")
	}
}

// SelectConnectionCredentials selects the final credentials for Apple services based on:
// - connections set up on Bitrise.io (globally for app)
// - step inputs for overriding the global config
func SelectConnectionCredentials(
	authType AuthType,
	bitriseConnection *devportalservice.AppleDeveloperConnection,
	inputs ConnectionOverrideInputs, logger log.Logger) (appleauth.Credentials, error) {
	if authType == APIKeyAuth && inputs.APIKeyPath != "" && inputs.APIKeyIssuerID != "" && inputs.APIKeyID != "" {
		logger.Infof("Overriding Bitrise Apple Service connection with Step-provided credentials (api_key_path, api_key_id, api_key_issuer_id)")

		config, err := parseConnectionOverrideConfig(inputs.APIKeyPath, inputs.APIKeyID, inputs.APIKeyIssuerID, logger)
		if err != nil {
			return appleauth.Credentials{}, err
		}
		return appleauth.Credentials{
			APIKey:  config,
			AppleID: nil,
		}, nil
	}

	if authType == APIKeyAuth {
		if bitriseConnection == nil || bitriseConnection.APIKeyConnection == nil {
			logger.Warnf(devportalclient.NotConnectedWarning)
			return appleauth.Credentials{}, fmt.Errorf("Apple Service connection via App Store Connect API key is not estabilished")
		}

		logger.Donef("Using Apple Service connection with API key.")
		return appleauth.Credentials{
			APIKey:  bitriseConnection.APIKeyConnection,
			AppleID: nil,
		}, nil
	}

	if authType == AppleIDAuth {
		if bitriseConnection == nil || bitriseConnection.AppleIDConnection == nil {
			logger.Warnf(devportalclient.NotConnectedWarning)
			return appleauth.Credentials{}, fmt.Errorf("Apple Service connection through Apple ID is not estabilished")
		}

		session, err := bitriseConnection.AppleIDConnection.FastlaneLoginSession()
		if err != nil {
			return appleauth.Credentials{}, fmt.Errorf("failed to restore Apple ID login session: %w", err)
		}

		if session != "" &&
			bitriseConnection.AppleIDConnection.SessionExpiryDate != nil &&
			bitriseConnection.AppleIDConnection.SessionExpiryDate.Before(time.Now()) {
			logger.Warnf("Two-factor session has expired at: %s", bitriseConnection.AppleIDConnection.SessionExpiryDate.Format("2006-01-02 15:04"))
		}

		logger.Donef("Using Apple Service connection with Apple ID.")
		return appleauth.Credentials{
			AppleID: &appleauth.AppleID{
				Username:            bitriseConnection.AppleIDConnection.AppleID,
				Password:            bitriseConnection.AppleIDConnection.Password,
				Session:             session,
				AppSpecificPassword: bitriseConnection.AppleIDConnection.AppSpecificPassword,
			},
			APIKey: nil,
		}, nil
	}

	panic("Unexpected AuthType")
}

func (m *Manager) selectCodeSigningStrategy(credentials appleauth.Credentials) (codeSigningStrategy, string, error) {
	const manualProfilesReason = "Using Bitrise-managed code signing assets with API key because Automatically managed signing is disabled in Xcode for the project."

	if credentials.AppleID != nil {
		return codeSigningBitriseAppleID, "Using Bitrise-managed code signing assets with Apple ID because Apple ID authentication is not supported by xcodebuild.", nil
	}

	if credentials.APIKey == nil {
		panic("No App Store Connect API authentication credentials found.")
	}

	if !m.opts.ShouldConsiderXcodeSigning {
		return codeSigningBitriseAPIKey, "", nil
	}

	if m.opts.XcodeMajorVersion < 13 {
		return codeSigningBitriseAPIKey, "Using Bitrise-managed code signing assets with API key because 'xcodebuild -allowProvisioningUpdates' with API authentication requires Xcode 13 or higher.", nil
	}

	isManaged, err := m.detailsProvider.IsSigningManagedAutomatically()
	if err != nil {
		return codeSigningBitriseAPIKey, manualProfilesReason, err
	}

	if !isManaged {
		return codeSigningBitriseAPIKey, manualProfilesReason, nil
	}

	if m.opts.MinDaysProfileValidity > 0 {
		return codeSigningBitriseAPIKey, "Specifying the minimum validity period of the Provisioning Profile is not supported by xcodebuild.", nil
	}

	return codeSigningXcode, "Automatically managed signing is enabled in Xcode for the project.", nil
}

func (m *Manager) downloadCertificates() ([]certificateutil.CertificateInfoModel, error) {
	certificates, err := m.certDownloader.GetCertificates()
	if err != nil {
		return nil, fmt.Errorf("failed to download certificates: %s", err)
	}

	if len(certificates) == 0 {
		m.logger.Warnf("No certificates are uploaded.")

		return nil, nil
	}

	m.logger.Printf("%d certificates downloaded:", len(certificates))
	for _, cert := range certificates {
		m.logger.Printf("- %s", cert)
	}

	return certificates, nil
}

func (m *Manager) installCertificates(certificates []certificateutil.CertificateInfoModel) error {
	for _, cert := range certificates {
		// Empty passphrase provided, as already parsed certificate + private key
		if err := m.assetInstaller.InstallCertificate(cert); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) validateCertificatesForXcodeManagedSigning(certificates []certificateutil.CertificateInfoModel) error {
	typeToLocalCerts, err := autocodesign.GetValidLocalCertificates(certificates)
	if err != nil {
		return err
	}

	certificateType, ok := autocodesign.CertificateTypeByDistribution[m.opts.ExportMethod]
	if !ok {
		panic(fmt.Sprintf("no valid certificate provided for distribution type: %s", m.opts.ExportMethod))
	}

	if len(typeToLocalCerts[certificateType]) == 0 {
		if certificateType == appstoreconnect.IOSDevelopment {
			return fmt.Errorf("no valid development type certificate uploaded")
		}

		m.logger.Warnf("no valid %s type certificate uploaded", certificateType)
	}

	return nil
}

func (m *Manager) registerTestDevices(credentials appleauth.Credentials, devices []devportalservice.TestDevice) error {
	platform, err := m.detailsProvider.Platform()
	if err != nil {
		return fmt.Errorf("failed to read platform from project: %s", err)
	}

	// No Team ID required for API key client
	devPortalClient, err := m.devPortalClientFactory.Create(credentials, "")
	if err != nil {
		return err
	}

	if _, err = autocodesign.EnsureTestDevices(devPortalClient, devices, autocodesign.Platform(platform)); err != nil {
		return fmt.Errorf("failed to ensure test devices: %w", err)
	}

	return nil
}

func (m *Manager) prepareCodeSigningWithBitrise(credentials appleauth.Credentials, testDevices []devportalservice.TestDevice) error {
	fmt.Println()
	m.logger.TDebugf("Analyzing project")
	appLayout, err := m.detailsProvider.GetAppLayout(m.opts.SignUITests)
	if err != nil {
		return err
	}

	fmt.Println()
	m.logger.TDebugf("Downloading certificates")
	certs, err := m.downloadCertificates()
	if err != nil {
		return err
	}

	typeToLocalCerts, err := autocodesign.GetValidLocalCertificates(certs)
	if err != nil {
		return err
	}

	var testDevicesToRegister []devportalservice.TestDevice
	if m.opts.RegisterTestDevices {
		testDevicesToRegister = testDevices
	}

	codesignAssetsByDistributionType, err := m.prepareAutomaticAssets(credentials, appLayout, typeToLocalCerts, testDevicesToRegister)
	if err != nil {
		if !m.fallbackProfileDownloader.IsAvailable() {
			return err
		}

		m.logger.Println()
		m.logger.Warnf("Automatic code signing failed: %s", err)
		m.logger.Println()
		m.logger.Infof("Falling back to manually managed codesigning assets.")

		return m.prepareManualAssets(certs)
	}

	if m.assetWriter != nil {
		if err := m.assetWriter.ForceCodesignAssets(m.opts.ExportMethod, codesignAssetsByDistributionType); err != nil {
			return fmt.Errorf("failed to force codesign settings: %s", err)
		}
	}

	return nil
}

func (m *Manager) prepareAutomaticAssets(credentials appleauth.Credentials, appLayout autocodesign.AppLayout, typeToLocalCerts autocodesign.LocalCertificates, testDevicesToRegister []devportalservice.TestDevice) (map[autocodesign.DistributionType]autocodesign.AppCodesignAssets, error) {
	devPortalClient, err := m.devPortalClientFactory.Create(credentials, m.opts.TeamID)
	if err != nil {
		return nil, err
	}

	if err := devPortalClient.Login(); err != nil {
		return nil, fmt.Errorf("Developer Portal client login failed: %w", err)
	}

	manager := autocodesign.NewCodesignAssetManager(devPortalClient, m.assetInstaller, m.localCodeSignAssetManager)

	codesignAssets, err := manager.EnsureCodesignAssets(appLayout, autocodesign.CodesignAssetsOpts{
		DistributionType:        m.opts.ExportMethod,
		TypeToLocalCertificates: typeToLocalCerts,
		BitriseTestDevices:      testDevicesToRegister,
		MinProfileValidityDays:  m.opts.MinDaysProfileValidity,
		VerboseLog:              m.opts.IsVerboseLog,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to ensure code signing assets: %w", err)
	}

	return codesignAssets, nil
}

func (m *Manager) prepareManualAssets(certificates []certificateutil.CertificateInfoModel) error {
	if err := m.installCertificates(certificates); err != nil {
		return err
	}

	profiles, err := m.fallbackProfileDownloader.GetProfiles()
	if err != nil {
		return fmt.Errorf("failed to fetch profiles: %w", err)
	}

	m.logger.Printf("Installing manual profiles:")
	for _, profile := range profiles {
		m.logger.Printf("%s", profile.Info.String(certificates...))

		if err := m.assetInstaller.InstallProfile(profile.Profile); err != nil {
			return fmt.Errorf("failed to install profile: %w", err)
		}
	}

	return nil
}
