package profileutil

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/fullsailor/pkcs7"
)

// ProfileType ...
type ProfileType string

const (
	// ProfileTypeIos ...
	ProfileTypeIos ProfileType = "ios"
	// ProfileTypeMacOs ...
	ProfileTypeMacOs ProfileType = "osx"
	// ProfileTypeTvOs ...
	ProfileTypeTvOs ProfileType = "tvos"
)

const (
	// IOSExtension is the iOS provisioning profile extension
	IOSExtension = ".mobileprovision"
	// MacExtension is the macOS provisioning profile extension
	MacExtension = ".provisionprofile"
)

// ProvisioningProfilesDirPath returns the provisioning profile directory path based on the Xcode major version.
func ProvisioningProfilesDirPath(xcodeMajorVersion int64) (string, error) {
	if xcodeMajorVersion >= 16 || xcodeMajorVersion == 0 { // return the modern path used by Xcode 16 and later
		return ProvisioningProfilesDirModernPath()
	}

	return ProvisioningProfilesDirLegacyPath() // return the legacy path used by Xcode 15 and earlier
}

// ProvisioningProfilesDirModernPath is the absolute path used to store and look up provisioning profiles (used Xcode 16 and later)
func ProvisioningProfilesDirModernPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(homeDir, "Library", "Developer", "Xcode", "UserData", "Provisioning Profiles"), nil
}

// ProvisioningProfilesDirLegacyPath is the absolute path used to store and look up provisioning profiles (used Xcode 15 and earlier)
func ProvisioningProfilesDirLegacyPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(homeDir, "Library", "MobileDevice", "Provisioning Profiles"), nil
}

// ProvisioningProfileFromContent ...
func ProvisioningProfileFromContent(content []byte) (*pkcs7.PKCS7, error) {
	return pkcs7.Parse(content)
}

// ProvisioningProfileFromFile ...
func ProvisioningProfileFromFile(pth string) (*pkcs7.PKCS7, error) {
	content, err := fileutil.ReadBytesFromFile(pth)
	if err != nil {
		return nil, err
	}
	return ProvisioningProfileFromContent(content)
}

// InstalledProvisioningProfiles ...
func InstalledProvisioningProfiles(profileType ProfileType) ([]*pkcs7.PKCS7, error) {
	pths, err := listAllProfiles(profileType)
	if err != nil {
		return nil, err
	}

	profiles := []*pkcs7.PKCS7{}
	for _, pth := range pths {
		profile, err := ProvisioningProfileFromFile(pth)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}
	return profiles, nil
}

// FindProvisioningProfile ...
func FindProvisioningProfile(uuid string) (*pkcs7.PKCS7, string, error) {
	paths, err := listProfiles(ProfileTypeIos, uuid)
	if err != nil {
		return nil, "", err
	}
	macOSPaths, err := listProfiles(ProfileTypeMacOs, uuid)
	if err != nil {
		return nil, "", err
	}

	paths = append(paths, macOSPaths...)
	if len(paths) == 0 {
		// ToDo return error of not found, keeping the nil return values for backward compatibility for now
		return nil, "", nil
	}

	profile, err := ProvisioningProfileFromFile(paths[0])
	if err != nil {
		return nil, "", err
	}
	return profile, paths[0], nil
}

func listAllProfiles(profileType ProfileType) ([]string, error) {
	return listProfiles(profileType, "*")
}

func listProfiles(profileType ProfileType, uuid string) ([]string, error) {
	ext := IOSExtension
	if profileType == ProfileTypeMacOs {
		ext = MacExtension
	}

	modernDirPath, err := ProvisioningProfilesDirModernPath()
	if err != nil {
		return nil, err
	}
	legacyDirPath, err := ProvisioningProfilesDirLegacyPath()
	if err != nil {
		return nil, err
	}

	var allProfilePaths []string
	for _, dirPath := range []string{modernDirPath, legacyDirPath} {
		pattern := filepath.Join(pathutil.EscapeGlobPath(dirPath), uuid+ext)
		profilePaths, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}

		allProfilePaths = append(allProfilePaths, profilePaths...)
	}

	return allProfilePaths, nil
}
