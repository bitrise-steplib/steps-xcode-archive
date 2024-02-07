package localcodesignasset

import (
	"os"

	"github.com/bitrise-io/go-xcode/profileutil"
	"github.com/bitrise-io/go-xcode/v2/autocodesign"
)

// ProvisioningProfileConverter ...
type ProvisioningProfileConverter interface {
	ProfileInfoToProfile(info profileutil.ProvisioningProfileInfoModel) (autocodesign.Profile, error)
}

type provisioningProfileConverter struct {
}

// NewProvisioningProfileConverter ...
func NewProvisioningProfileConverter() ProvisioningProfileConverter {
	return provisioningProfileConverter{}
}

// ProfileInfoToProfile ...
func (c provisioningProfileConverter) ProfileInfoToProfile(info profileutil.ProvisioningProfileInfoModel) (autocodesign.Profile, error) {
	_, pth, err := profileutil.FindProvisioningProfile(info.UUID)
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(pth)
	if err != nil {
		return nil, err
	}

	return NewProfile(info, content), nil
}
