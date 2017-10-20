package utils

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-tools/go-xcode/profileutil"
)

// GetDefaultProvisioningProfile ...
func GetDefaultProvisioningProfile() (profileutil.ProvisioningProfileInfoModel, error) {
	defaultProfileURL := os.Getenv("BITRISE_DEFAULT_PROVISION_URL")
	if defaultProfileURL == "" {
		return profileutil.ProvisioningProfileInfoModel{}, nil
	}

	tmpDir, err := pathutil.NormalizedOSTempDirPath("tmp_default_profile")
	if err != nil {
		return profileutil.ProvisioningProfileInfoModel{}, err
	}

	tmpDst := filepath.Join(tmpDir, "default.mobileprovision")
	tmpDstFile, err := os.Create(tmpDst)
	if err != nil {
		return profileutil.ProvisioningProfileInfoModel{}, err
	}
	defer func() {
		if err := tmpDstFile.Close(); err != nil {
			log.Errorf("Failed to close file (%s), error: %s", tmpDst, err)
		}
	}()

	response, err := http.Get(defaultProfileURL)
	if err != nil {
		return profileutil.ProvisioningProfileInfoModel{}, err
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			log.Errorf("Failed to close response body, error: %s", err)
		}
	}()

	if _, err := io.Copy(tmpDstFile, response.Body); err != nil {
		return profileutil.ProvisioningProfileInfoModel{}, err
	}

	defaultProfile, err := profileutil.NewProvisioningProfileInfoFromFile(tmpDst)
	if err != nil {
		return profileutil.ProvisioningProfileInfoModel{}, err
	}

	return defaultProfile, nil
}
