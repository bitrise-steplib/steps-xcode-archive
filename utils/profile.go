package utils

import (
	"path/filepath"

	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-tools/go-xcode/provisioningprofile"
	"github.com/pkg/errors"
)

const (
	provProfileSystemDirPath = "~/Library/MobileDevice/Provisioning Profiles"
)

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
