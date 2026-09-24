package profileutil

import (
	"errors"
	"io"
	"path/filepath"

	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/fullsailor/pkcs7"
)

const (
	// IOSExtension is the iOS provisioning profile extension
	IOSExtension = ".mobileprovision"
	// MacExtension is the macOS provisioning profile extension
	MacExtension = ".provisionprofile"
)

// ProfileReader ...
type ProfileReader struct {
	logger       log.Logger
	fileManager  fileutil.FileManager
	pathModifier pathutil.PathModifier
	pathProvider pathutil.PathProvider
}

// NewProfileReader ...
func NewProfileReader(logger log.Logger, fileManager fileutil.FileManager, pathModifier pathutil.PathModifier, pathProvider pathutil.PathProvider) ProfileReader {
	return ProfileReader{
		logger:       logger,
		fileManager:  fileManager,
		pathModifier: pathModifier,
		pathProvider: pathProvider,
	}
}

// ProvisioningProfileInfoFromFile ...
func (reader ProfileReader) ProvisioningProfileInfoFromFile(pth string) (ProvisioningProfileInfoModel, error) {
	provisioningProfile, err := reader.provisioningProfileFromFile(pth)
	if err != nil {
		return ProvisioningProfileInfoModel{}, err
	}
	if provisioningProfile != nil {
		return NewProvisioningProfileInfo(*provisioningProfile)
	}
	return ProvisioningProfileInfoModel{}, errors.New("failed to parse provisioning profile infos")
}

// InstalledProvisioningProfileInfos ...
func (reader ProfileReader) InstalledProvisioningProfileInfos(profileType ProfileType) ([]ProvisioningProfileInfoModel, error) {
	provisioningProfiles, err := reader.installedProvisioningProfiles(profileType)
	if err != nil {
		return nil, err
	}

	var infos []ProvisioningProfileInfoModel
	for _, provisioningProfile := range provisioningProfiles {
		if provisioningProfile != nil {
			info, err := NewProvisioningProfileInfo(*provisioningProfile)
			if err != nil {
				return nil, err
			}
			infos = append(infos, info)
		}
	}
	return infos, nil
}

// ListProfiles ...
func (reader ProfileReader) ListProfiles(profileType ProfileType, uuid string) ([]string, error) {
	ext := IOSExtension
	if profileType == ProfileTypeMacOs {
		ext = MacExtension
	}

	modernDirPath, err := reader.provisioningProfilesDirModernPath()
	if err != nil {
		return nil, err
	}

	legacyDirPath, err := reader.provisioningProfilesDirLegacyPath()
	if err != nil {
		return nil, err
	}

	var allProfilePaths []string
	for _, dirPath := range []string{modernDirPath, legacyDirPath} {
		pattern := filepath.Join(reader.pathModifier.EscapeGlobPath(dirPath), uuid+ext)
		profilePaths, err := reader.pathProvider.Glob(pattern)
		if err != nil {
			return nil, err
		}

		allProfilePaths = append(allProfilePaths, profilePaths...)
	}

	return allProfilePaths, nil
}

// ProvisioningProfilesDirPath returns the provisioning profile directory path based on the Xcode major version.
func (reader ProfileReader) ProvisioningProfilesDirPath(xcodeMajorVersion int64) (string, error) {
	if xcodeMajorVersion >= 16 || xcodeMajorVersion == 0 { // return the modern path used by Xcode 16 and later
		return reader.provisioningProfilesDirModernPath()
	}

	return reader.provisioningProfilesDirLegacyPath() // return the legacy path used by Xcode 15 and earlier
}

func (reader ProfileReader) provisioningProfileFromFile(pth string) (*pkcs7.PKCS7, error) {
	f, err := reader.fileManager.Open(pth)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			reader.logger.Warnf("Failed to close file %s, error: %s", pth, err)
		}
	}()

	content, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return pkcs7.Parse(content)
}

func (reader ProfileReader) installedProvisioningProfiles(profileType ProfileType) ([]*pkcs7.PKCS7, error) {
	pths, err := reader.ListProfiles(profileType, "*")
	if err != nil {
		return nil, err
	}

	var profiles []*pkcs7.PKCS7
	for _, pth := range pths {
		profile, err := reader.provisioningProfileFromFile(pth)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}
	return profiles, nil
}

// ProvisioningProfilesDirModernPath is the absolute path used to store and look up provisioning profiles (used Xcode 16 and later)
func (reader ProfileReader) provisioningProfilesDirModernPath() (string, error) {
	return reader.pathModifier.AbsPath("~/Library/Developer/Xcode/UserData/Provisioning Profiles")
}

// ProvisioningProfilesDirLegacyPath is the absolute path used to store and look up provisioning profiles (used Xcode 15 and earlier)
func (reader ProfileReader) provisioningProfilesDirLegacyPath() (string, error) {
	return reader.pathModifier.AbsPath("~/Library/MobileDevice/Provisioning Profiles")
}
