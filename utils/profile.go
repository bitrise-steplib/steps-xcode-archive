package utils

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/steps-certificate-and-profile-installer/profileutil"
	"github.com/bitrise-tools/go-xcode/provisioningprofile"
	"github.com/pkg/errors"
)

const (
	provProfileSystemDirPath = "~/Library/MobileDevice/Provisioning Profiles"
)

// InstalledIosProfiles ...
func InstalledIosProfiles() ([]profileutil.ProfileInfoModel, error) {
	profiles := []profileutil.ProfileInfoModel{}

	if err := WalkIOSProvProfilesPth(func(pth string) bool {
		profile, err := profileutil.ProfileFromFile(pth)
		if err != nil {
			log.Errorf("Failed to walk provisioning profiles, error: %s", err)
			os.Exit(1)
		}

		profiles = append(profiles, profile)
		return false
	}); err != nil {
		return nil, err
	}

	return profiles, nil
}

// WalkIOSProvProfilesPth ...
func WalkIOSProvProfilesPth(walkFunc func(pth string) bool) error {
	absProvProfileDirPath, err := pathutil.AbsPath(provProfileSystemDirPath)
	if err != nil {
		return errors.Wrap(err, "failed to get Absolute path of Provisioning Profiles dir")
	}

	pths, err := filepath.Glob(absProvProfileDirPath + "/*.mobileprovision")
	if err != nil {
		return errors.Wrap(err, "failed to perform *.mobileprovision search")
	}

	for _, pth := range pths {
		if breakWalk := walkFunc(pth); breakWalk {
			break
		}
	}

	return nil
}

// WalkIOSProvProfiles ...
func WalkIOSProvProfiles(walkFunc func(profile provisioningprofile.Profile) bool) error {
	var profileErr error
	if walkErr := WalkIOSProvProfilesPth(func(pth string) bool {
		profile, err := provisioningprofile.NewProfileFromFile(pth)
		if err != nil {
			profileErr = err
			return true
		}

		return walkFunc(profile)
	}); walkErr != nil {
		return walkErr
	}

	return profileErr
}

// GetDefaultProvisioningProfile ...
func GetDefaultProvisioningProfile() (profileutil.ProfileInfoModel, error) {
	defaultProfileURL := os.Getenv("BITRISE_DEFAULT_PROVISION_URL")
	if defaultProfileURL == "" {
		return profileutil.ProfileInfoModel{}, nil
	}

	tmpDir, err := pathutil.NormalizedOSTempDirPath("tmp_default_profile")
	if err != nil {
		return profileutil.ProfileInfoModel{}, err
	}

	tmpDst := filepath.Join(tmpDir, "default.mobileprovision")
	tmpDstFile, err := os.Create(tmpDst)
	if err != nil {
		return profileutil.ProfileInfoModel{}, err
	}
	defer func() {
		if err := tmpDstFile.Close(); err != nil {
			log.Errorf("Failed to close file (%s), error: %s", tmpDst, err)
		}
	}()

	response, err := http.Get(defaultProfileURL)
	if err != nil {
		return profileutil.ProfileInfoModel{}, err
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			log.Errorf("Failed to close response body, error: %s", err)
		}
	}()

	if _, err := io.Copy(tmpDstFile, response.Body); err != nil {
		return profileutil.ProfileInfoModel{}, err
	}

	defaultProfile, err := profileutil.ProfileFromFile(tmpDst)
	if err != nil {
		return profileutil.ProfileInfoModel{}, err
	}

	return defaultProfile, nil
}
