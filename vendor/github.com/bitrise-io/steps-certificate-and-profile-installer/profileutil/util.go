package profileutil

import (
	"path/filepath"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/fullsailor/pkcs7"
)

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

// InstalledIosProvisioningProfiles ...
func InstalledIosProvisioningProfiles() ([]*pkcs7.PKCS7, error) {
	provProfileSystemDirPath := "~/Library/MobileDevice/Provisioning Profiles"
	absProvProfileDirPath, err := pathutil.AbsPath(provProfileSystemDirPath)
	if err != nil {
		return nil, err
	}
	pths, err := filepath.Glob(absProvProfileDirPath + "/*.mobileprovision")
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
