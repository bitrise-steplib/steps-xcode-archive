// Package codesignasset implements a autocodesign.AssetWriter which writes certificates, profiles to the keychain and filesystem.
package codesignasset

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-xcode/certificateutil"
	"github.com/bitrise-io/go-xcode/v2/autocodesign"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/devportalclient/appstoreconnect"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/keychain"
)

// Writer ...
type Writer struct {
	keychain keychain.Keychain
}

// NewWriter ...
func NewWriter(keychain keychain.Keychain) Writer {
	return Writer{
		keychain: keychain,
	}
}

// Write ...
func (w Writer) Write(codesignAssetsByDistributionType map[autocodesign.DistributionType]autocodesign.AppCodesignAssets) error {
	i := 0
	for _, codesignAssets := range codesignAssetsByDistributionType {
		log.Printf("certificate: %s", codesignAssets.Certificate.CommonName)

		if err := w.keychain.InstallCertificate(codesignAssets.Certificate, ""); err != nil {
			return fmt.Errorf("failed to install certificate: %s", err)
		}

		log.Printf("profiles:")
		for _, profile := range codesignAssets.ArchivableTargetProfilesByBundleID {
			log.Printf("- %s", profile.Attributes().Name)

			if err := w.InstallProfile(profile); err != nil {
				return fmt.Errorf("failed to write profile to file: %s", err)
			}
		}

		for _, profile := range codesignAssets.UITestTargetProfilesByBundleID {
			log.Printf("- %s", profile.Attributes().Name)

			if err := w.InstallProfile(profile); err != nil {
				return fmt.Errorf("failed to write profile to file: %s", err)
			}
		}

		if i < len(codesignAssetsByDistributionType)-1 {
			fmt.Println()
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
	homeDir := os.Getenv("HOME")
	profilesDir := path.Join(homeDir, "Library/MobileDevice/Provisioning Profiles")
	if exists, err := pathutil.IsDirExists(profilesDir); err != nil {
		return fmt.Errorf("failed to check directory (%s) for provisioning profiles: %s", profilesDir, err)
	} else if !exists {
		if err := os.MkdirAll(profilesDir, 0600); err != nil {
			return fmt.Errorf("failed to generate directory (%s) for provisioning profiles: %s", profilesDir, err)
		}
	}

	var ext string
	switch profile.Attributes().Platform {
	case appstoreconnect.IOS:
		ext = ".mobileprovision"
	case appstoreconnect.MacOS:
		ext = ".provisionprofile"
	default:
		return fmt.Errorf("failed to write profile to file, unsupported platform: (%s). Supported platforms: %s, %s", profile.Attributes().Platform, appstoreconnect.IOS, appstoreconnect.MacOS)
	}

	name := path.Join(profilesDir, profile.Attributes().UUID+ext)
	if err := ioutil.WriteFile(name, profile.Attributes().ProfileContent, 0600); err != nil {
		return fmt.Errorf("failed to write profile to file: %s", err)
	}

	return nil
}
