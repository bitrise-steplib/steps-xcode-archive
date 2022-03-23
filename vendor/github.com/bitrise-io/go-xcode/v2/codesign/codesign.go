package codesign

import (
	"errors"
	"fmt"

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

type localCertificates map[appstoreconnect.CertificateType][]certificateutil.CertificateInfoModel

type codeSigningStrategy int

const (
	codeSigningXcode codeSigningStrategy = iota
	codeSigningBitriseAPIKey
	codeSigningBitriseAppleID
)

// Opts ...
type Opts struct {
	AuthType                          AuthType
	FallbackToLocalAssetsOnAPIFailure bool
	ShouldConsiderXcodeSigning        bool
	TeamID                            string

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
	bitriseConnection         *devportalservice.AppleDeveloperConnection
	devPortalClientFactory    devportalclient.Factory
	certDownloader            autocodesign.CertificateProvider
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
	connection *devportalservice.AppleDeveloperConnection,
	clientFactory devportalclient.Factory,
	certDownloader autocodesign.CertificateProvider,
	assetInstaller autocodesign.AssetWriter,
	localCodeSignAssetManager autocodesign.LocalCodeSignAssetManager,
	archive Archive,
	logger log.Logger,
) Manager {
	return Manager{
		opts:                      opts,
		appleAuthCredentials:      appleAuth,
		bitriseConnection:         connection,
		devPortalClientFactory:    clientFactory,
		certDownloader:            certDownloader,
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
	connection *devportalservice.AppleDeveloperConnection,
	clientFactory devportalclient.Factory,
	certDownloader autocodesign.CertificateProvider,
	assetInstaller autocodesign.AssetWriter,
	localCodeSignAssetManager autocodesign.LocalCodeSignAssetManager,
	project projectmanager.Project,
	logger log.Logger,
) Manager {
	return Manager{
		opts:                      opts,
		appleAuthCredentials:      appleAuth,
		bitriseConnection:         connection,
		devPortalClientFactory:    clientFactory,
		certDownloader:            certDownloader,
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
			m.logger.Infof("Downloading certificates from Bitrise")
			certificates, err := m.downloadCertificates()
			if err != nil {
				return nil, err
			}

			if err := m.checkXcodeManagedCertificates(certificates); err != nil {
				return nil, err
			}

			if err := m.installCertificates(certificates); err != nil {
				return nil, err
			}

			needsTestDevices := autocodesign.DistributionTypeRequiresDeviceList([]autocodesign.DistributionType{m.opts.ExportMethod})
			if needsTestDevices && m.opts.RegisterTestDevices && m.bitriseConnection != nil && len(m.bitriseConnection.TestDevices) != 0 {
				if err := m.registerTestDevices(m.appleAuthCredentials, m.bitriseConnection.TestDevices); err != nil {
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
			if err := m.prepareCodeSigningWithBitrise(m.appleAuthCredentials); err != nil {
				return nil, err
			}

			return nil, nil
		}
	default:
		return nil, fmt.Errorf("unknown code sign strategy")
	}
}

// SelectConnectionCredentials ...
func SelectConnectionCredentials(authType AuthType, conn *devportalservice.AppleDeveloperConnection, logger log.Logger) (appleauth.Credentials, error) {
	var authSource appleauth.Source

	switch authType {
	case APIKeyAuth:
		authSource = &appleauth.ConnectionAPIKeySource{}
	case AppleIDAuth:
		authSource = &appleauth.ConnectionAppleIDFastlaneSource{}
	default:
		panic("missing implementation")
	}

	authConfig, err := appleauth.Select(conn, []appleauth.Source{authSource}, appleauth.Inputs{})
	if err != nil {
		if conn != nil && conn.APIKeyConnection == nil && conn.AppleIDConnection == nil {
			fmt.Println()
			logger.Warnf("%s", devportalclient.NotConnectedWarning)
		}

		if errors.Is(err, &appleauth.MissingAuthConfigError{}) {
			if authType == AppleIDAuth {
				return appleauth.Credentials{}, fmt.Errorf("Apple ID authentication is selected in Step inputs, but Bitrise Apple Service connection is unset")
			}

			return appleauth.Credentials{}, fmt.Errorf("API key authentication is selected in Step inputs, but Bitrise Apple Service connection is unset")
		}

		return appleauth.Credentials{}, fmt.Errorf("could not select Apple authentication credentials: %w", err)
	}

	if authConfig.APIKey != nil {
		authConfig.AppleID = nil
		logger.Donef("Using Apple Service connection with API key.")
	} else if authConfig.AppleID != nil {
		authConfig.APIKey = nil
		logger.Donef("Using Apple Service connection with Apple ID.")
	} else {
		panic("No Apple authentication credentials found.")
	}

	return authConfig, nil
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
		m.logger.Warnf("No certificates are uploaded to Bitrise.")

		return nil, nil
	}

	return certificates, nil
}

func (m *Manager) installCertificates(certificates []certificateutil.CertificateInfoModel) error {
	m.logger.Infof("Installing certificates:")
	for _, cert := range certificates {
		m.logger.Printf("- %s", cert)
		// Empty passphrase provided, as already parsed certificate + private key
		if err := m.assetInstaller.InstallCertificate(cert); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) checkXcodeManagedCertificates(certificates []certificateutil.CertificateInfoModel) error {
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

func (m *Manager) prepareCodeSigningWithBitrise(credentials appleauth.Credentials) error {
	// Analyze project
	fmt.Println()
	m.logger.Infof("Analyzing project")
	appLayout, err := m.detailsProvider.GetAppLayout(m.opts.SignUITests)
	if err != nil {
		return err
	}

	devPortalClient, err := m.devPortalClientFactory.Create(credentials, m.opts.TeamID)
	if err != nil {
		return err
	}

	fmt.Println()
	m.logger.Infof("Downloading certificates")

	certs, err := m.certDownloader.GetCertificates()
	if err != nil {
		return fmt.Errorf("failed to download certificates: %w", err)
	}

	if len(certs) > 0 {
		m.logger.Printf("%d certificates downloaded:", len(certs))
		for _, cert := range certs {
			m.logger.Printf("- %s", cert.String())
		}
	} else {
		m.logger.Warnf("No certificates are uploaded to Bitrise.")
	}

	typeToLocalCerts, err := autocodesign.GetValidLocalCertificates(certs)
	if err != nil {
		return err
	}

	manager := autocodesign.NewCodesignAssetManager(devPortalClient, m.assetInstaller, m.localCodeSignAssetManager)

	// Fetch and apply codesigning assets
	var testDevices []devportalservice.TestDevice
	if m.opts.RegisterTestDevices && m.bitriseConnection != nil {
		testDevices = m.bitriseConnection.TestDevices
	}

	codesignAssetsByDistributionType, err := manager.EnsureCodesignAssets(appLayout, autocodesign.CodesignAssetsOpts{
		DistributionType:          m.opts.ExportMethod,
		TypeToBitriseCertificates: typeToLocalCerts,
		BitriseTestDevices:        testDevices,
		MinProfileValidityDays:    m.opts.MinDaysProfileValidity,
		VerboseLog:                m.opts.IsVerboseLog,
	})
	if err != nil {
		if !m.opts.FallbackToLocalAssetsOnAPIFailure {
			return err
		}

		m.logger.Warnf("Error: %s", err)
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

func (m *Manager) prepareManualAssets(certificates []certificateutil.CertificateInfoModel) error {
	if err := m.installCertificates(certificates); err != nil {
		return err
	}

	return nil
}
