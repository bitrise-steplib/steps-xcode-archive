package localcodesignasset

import "github.com/bitrise-io/go-xcode/profileutil"

// ProvisioningProfileProvider can list profile infos.
type ProvisioningProfileProvider interface {
	ListProvisioningProfiles() ([]profileutil.ProvisioningProfileInfoModel, error)
}

type provisioningProfileProvider struct{}

// NewProvisioningProfileProvider ...
func NewProvisioningProfileProvider() ProvisioningProfileProvider {
	return provisioningProfileProvider{}
}

// ListProvisioningProfiles ...
func (p provisioningProfileProvider) ListProvisioningProfiles() ([]profileutil.ProvisioningProfileInfoModel, error) {
	return profileutil.InstalledProvisioningProfileInfos(profileutil.ProfileTypeIos)
}
