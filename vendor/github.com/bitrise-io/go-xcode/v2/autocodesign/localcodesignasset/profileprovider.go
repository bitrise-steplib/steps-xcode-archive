package localcodesignasset

import (
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-io/go-xcode/v2/profileutil"
)

// ProvisioningProfileProvider can list profile infos.
type ProvisioningProfileProvider interface {
	ListProvisioningProfiles() ([]profileutil.ProvisioningProfileInfoModel, error)
}

type provisioningProfileProvider struct {
	profileReader profileutil.ProfileReader
}

// NewProvisioningProfileProvider ...
func NewProvisioningProfileProvider() ProvisioningProfileProvider {
	profileReader := profileutil.NewProfileReader(log.NewLogger(), fileutil.NewFileManager(), pathutil.NewPathModifier(), pathutil.NewPathProvider())
	return provisioningProfileProvider{
		profileReader: profileReader,
	}
}

// ListProvisioningProfiles ...
func (p provisioningProfileProvider) ListProvisioningProfiles() ([]profileutil.ProvisioningProfileInfoModel, error) {
	return p.profileReader.InstalledProvisioningProfileInfos(profileutil.ProfileTypeIos)
}
