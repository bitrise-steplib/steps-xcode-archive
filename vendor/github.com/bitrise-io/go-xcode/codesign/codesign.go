package codesign

import (
	"errors"
	"fmt"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-xcode/appleauth"
	"github.com/bitrise-io/go-xcode/autocodesign"
	"github.com/bitrise-io/go-xcode/autocodesign/devportalclient"
	"github.com/bitrise-io/go-xcode/autocodesign/devportalclient/appstoreconnect"
	"github.com/bitrise-io/go-xcode/autocodesign/projectmanager"
	"github.com/bitrise-io/go-xcode/devportalservice"
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

	ExportMethod      autocodesign.DistributionType
	XcodeMajorVersion int

	RegisterTestDevices bool
	SignUITests         bool
	MinProfileValidity  int
	IsVerboseLog        bool
}

// Manager ...
type Manager struct {
	opts Opts

	appleAuthCredentials   appleauth.Credentials
	bitriseConnection      *devportalservice.AppleDeveloperConnection
	devPortalClientFactory devportalclient.Factory
	certDownloader         autocodesign.CertificateProvider
	assetWriter            autocodesign.AssetWriter

	projectFactory projectmanager.Factory
	project        Project

	logger log.Logger
}

// NewManager ...
func NewManager(
	opts Opts,
	logger log.Logger,
	appleAuth appleauth.Credentials,
	connection *devportalservice.AppleDeveloperConnection,
	clientFactory devportalclient.Factory,
	certDownloader autocodesign.CertificateProvider,
	assetWriter autocodesign.AssetWriter,
	projectFactory projectmanager.Factory,
) Manager {
	return Manager{
		opts:                   opts,
		appleAuthCredentials:   appleAuth,
		bitriseConnection:      connection,
		devPortalClientFactory: clientFactory,
		certDownloader:         certDownloader,
		assetWriter:            assetWriter,
		projectFactory:         projectFactory,
		logger:                 logger,
	}
}

// Project ...
type Project interface {
	IsSigningManagedAutomatically() (bool, error)
	Platform() (autocodesign.Platform, error)
	GetAppLayout(uiTestTargets bool) (autocodesign.AppLayout, error)
	ForceCodesignAssets(distribution autocodesign.DistributionType, codesignAssetsByDistributionType map[autocodesign.DistributionType]autocodesign.AppCodesignAssets) error
}

func (m *Manager) getProject() (Project, error) {
	if m.project == nil {
		var err error
		m.project, err = m.projectFactory.Create()
		if err != nil {
			return nil, fmt.Errorf("failed to open project: %s", err)
		}
	}

	return m.project, nil
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
			m.logger.Infof("Preparing for Xcode-managed code signing")
			m.logger.Printf(reason)
			m.logger.Println()
			m.logger.Infof("Downloading certificates from Bitrise")
			if err := m.downloadAndInstallCertificates(); err != nil {
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
			m.logger.Infof("Bitrise-managed code signing")
			m.logger.Printf(reason)
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
	const manualProfilesReason = "Using Bitrise-managed code signing via API key, as Automatically managed signing is disabled in Xcode for the project."

	if credentials.AppleID != nil {
		return codeSigningBitriseAppleID, "Using Bitrise-managed code signing via Apple ID, as Apple ID is not supported by Xcode-managed code signing.", nil
	}

	if credentials.APIKey == nil {
		panic("No Apple authentication credentials found.")
	}

	if !m.opts.ShouldConsiderXcodeSigning {
		return codeSigningBitriseAPIKey, "", nil
	}

	if m.opts.XcodeMajorVersion < 13 {
		return codeSigningBitriseAPIKey, "Using Bitrise-managed code signing via API key, as Xcode-managed code signing requires at least Xcode 13.", nil
	}

	project, err := m.getProject()
	if err != nil {
		return codeSigningXcode, "Using Xcode-managed code signing, as project parsing failed.", err
	}

	isManaged, err := project.IsSigningManagedAutomatically()
	if err != nil {
		return codeSigningBitriseAPIKey, manualProfilesReason, err
	}

	if isManaged {
		return codeSigningXcode, "Using Xcode-managed code signing, as Automatically managed signing is enabled in Xcode for the project", nil
	}

	return codeSigningBitriseAPIKey, manualProfilesReason, nil
}

func (m *Manager) downloadAndInstallCertificates() error {
	certificates, err := m.certDownloader.GetCertificates()
	if err != nil {
		return fmt.Errorf("failed to download certificates: %s", err)
	}

	certificateType, ok := autocodesign.CertificateTypeByDistribution[m.opts.ExportMethod]
	if !ok {
		panic(fmt.Sprintf("no valid certificate provided for distribution type: %s", m.opts.ExportMethod))
	}

	teamID := ""
	typeToLocalCerts, err := autocodesign.GetValidLocalCertificates(certificates, teamID)
	if err != nil {
		return err
	}

	if len(typeToLocalCerts[certificateType]) == 0 {
		if certificateType == appstoreconnect.IOSDevelopment {
			return fmt.Errorf("no valid development type certificate uploaded")
		}
		log.Warnf("no valid %s type certificate uploaded", certificateType)
	}

	m.logger.Infof("Installing downloaded certificates:")
	for _, cert := range certificates {
		m.logger.Printf("- %s", cert)
		// Empty passphrase provided, as already parsed certificate + private key
		if err := m.assetWriter.InstallCertificate(cert); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) registerTestDevices(credentials appleauth.Credentials, devices []devportalservice.TestDevice) error {
	project, err := m.getProject()
	if err != nil {
		return err
	}

	platform, err := project.Platform()
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
	project, err := m.getProject()
	if err != nil {
		return err
	}

	appLayout, err := project.GetAppLayout(m.opts.SignUITests)
	if err != nil {
		return err
	}

	devPortalClient, err := m.devPortalClientFactory.Create(credentials, appLayout.TeamID)
	if err != nil {
		return err
	}

	manager := autocodesign.NewCodesignAssetManager(devPortalClient, m.certDownloader, m.assetWriter)

	// Fetch and apply codesigning assets
	var testDevices []devportalservice.TestDevice
	if m.opts.RegisterTestDevices && m.bitriseConnection != nil {
		testDevices = m.bitriseConnection.TestDevices
	}
	codesignAssetsByDistributionType, err := manager.EnsureCodesignAssets(appLayout, autocodesign.CodesignAssetsOpts{
		DistributionType:       m.opts.ExportMethod,
		BitriseTestDevices:     testDevices,
		MinProfileValidityDays: m.opts.MinProfileValidity,
		VerboseLog:             m.opts.IsVerboseLog,
	})
	if err != nil {
		return err
	}

	if err := project.ForceCodesignAssets(m.opts.ExportMethod, codesignAssetsByDistributionType); err != nil {
		return fmt.Errorf("failed to force codesign settings: %s", err)
	}

	return nil
}
