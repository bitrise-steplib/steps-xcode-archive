package localcodesignasset

import (
	"fmt"
	"io"

	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-io/go-xcode/v2/autocodesign"
	"github.com/bitrise-io/go-xcode/v2/profileutil"
)

// ProvisioningProfileConverter ...
type ProvisioningProfileConverter interface {
	ProfileInfoToProfile(info profileutil.ProvisioningProfileInfoModel) (autocodesign.Profile, error)
}

type provisioningProfileConverter struct {
	logger        log.Logger
	fileManager   fileutil.FileManager
	profileReader profileutil.ProfileReader
}

// NewProvisioningProfileConverter ...
func NewProvisioningProfileConverter() ProvisioningProfileConverter {
	logger := log.NewLogger()
	fileManager := fileutil.NewFileManager()
	pathModifier := pathutil.NewPathModifier()
	pathProvider := pathutil.NewPathProvider()
	profileReader := profileutil.NewProfileReader(logger, fileManager, pathModifier, pathProvider)

	return provisioningProfileConverter{
		logger:        logger,
		fileManager:   fileManager,
		profileReader: profileReader,
	}
}

// ProfileInfoToProfile ...
func (c provisioningProfileConverter) ProfileInfoToProfile(info profileutil.ProvisioningProfileInfoModel) (autocodesign.Profile, error) {
	pth, err := c.findProvisioningProfile(info.UUID)
	if err != nil {
		return nil, err
	}
	profile, err := c.fileManager.Open(pth)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = profile.Close(); err != nil {
			c.logger.Warnf("Failed to close profile: %s", err)
		}
	}()
	content, err := io.ReadAll(profile)
	if err != nil {
		return nil, err
	}

	return NewProfile(info, content), nil
}

func (c provisioningProfileConverter) findProvisioningProfile(uuid string) (string, error) {
	paths, err := c.profileReader.ListProfiles(profileutil.ProfileTypeIos, uuid)
	if err != nil {
		return "", err
	}
	macOSPaths, err := c.profileReader.ListProfiles(profileutil.ProfileTypeMacOs, uuid)
	if err != nil {
		return "", err
	}

	paths = append(paths, macOSPaths...)
	if len(paths) == 0 {
		return "", fmt.Errorf("no provisioning profile found for %s", uuid)
	}

	_, err = c.profileReader.ProvisioningProfileInfoFromFile(paths[0])
	if err != nil {
		return "", err
	}
	return paths[0], nil
}
