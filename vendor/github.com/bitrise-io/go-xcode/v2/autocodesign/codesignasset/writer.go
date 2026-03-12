// Package codesignasset implements a autocodesign.AssetWriter which writes certificates, profiles to the keychain and filesystem.
package codesignasset

import (
	"fmt"
	"path"

	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/certificateutil"
	"github.com/bitrise-io/go-xcode/profileutil"
	"github.com/bitrise-io/go-xcode/v2/autocodesign"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/devportalclient/appstoreconnect"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/keychain"
)

// Writer ...
type Writer struct {
	logger            log.Logger
	keychain          keychain.Keychain
	fileManager       fileutil.FileManager
	xcodeMajorVersion int64
}

// NewWriter ...
func NewWriter(logger log.Logger, keychain keychain.Keychain, fileManager fileutil.FileManager, xcodeMajorVersion int64) Writer {
	return Writer{
		logger:            logger,
		keychain:          keychain,
		fileManager:       fileManager,
		xcodeMajorVersion: xcodeMajorVersion,
	}
}

// Write ...
func (w Writer) Write(codesignAssetsByDistributionType map[autocodesign.DistributionType]autocodesign.AppCodesignAssets) error {
	i := 0
	for _, codesignAssets := range codesignAssetsByDistributionType {
		w.logger.Printf("certificate: %s", codesignAssets.Certificate.CommonName)

		if err := w.keychain.InstallCertificate(codesignAssets.Certificate, ""); err != nil {
			return fmt.Errorf("failed to install certificate: %w", err)
		}

		w.logger.Printf("profiles:")
		for _, profile := range codesignAssets.ArchivableTargetProfilesByBundleID {
			w.logger.Printf("- %s", profile.Attributes().Name)

			if err := w.InstallProfile(profile); err != nil {
				return fmt.Errorf("failed to write profile to file: %w", err)
			}
		}

		for _, profile := range codesignAssets.UITestTargetProfilesByBundleID {
			w.logger.Printf("- %s", profile.Attributes().Name)

			if err := w.InstallProfile(profile); err != nil {
				return fmt.Errorf("failed to write profile to file: %w", err)
			}
		}

		if i < len(codesignAssetsByDistributionType)-1 {
			w.logger.Println()
		}
		i++
	}

	return nil
}

// InstallCertificate installs the certificate to the Keychain
func (w Writer) InstallCertificate(certificate certificateutil.CertificateInfoModel) error {
	// Empty passphrase provided, as already parsed certificate + private key
	return w.keychain.InstallCertificate(certificate, "")
}

// InstallProfile writes the provided profile under the `$HOME/Library/MobileDevice/Provisioning Profiles` directory.
// Xcode uses profiles located in that directory.
// The file extension depends on the profile's platform `IOS` => `.mobileprovision`, `MAC_OS` => `.provisionprofile`
func (w Writer) InstallProfile(profile autocodesign.Profile) error {
	var ext string
	switch profile.Attributes().Platform {
	case appstoreconnect.IOS:
		ext = profileutil.IOSExtension
	case appstoreconnect.MacOS:
		ext = profileutil.MacExtension
	default:
		return fmt.Errorf("failed to write profile to file, unsupported platform: (%s). Supported platforms: %s, %s", profile.Attributes().Platform, appstoreconnect.IOS, appstoreconnect.MacOS)
	}

	profilesDir, err := profileutil.ProvisioningProfilesDirPath(w.xcodeMajorVersion)
	if err != nil {
		return fmt.Errorf("failed to get provisioning profiles directory path: %w", err)
	}

	name := path.Join(profilesDir, profile.Attributes().UUID+ext)
	// Write will create directory if does not exist
	if err := w.fileManager.Write(name, string(profile.Attributes().ProfileContent), 0600); err != nil {
		return fmt.Errorf("failed to write profile to file: %w", err)
	}

	return nil
}
