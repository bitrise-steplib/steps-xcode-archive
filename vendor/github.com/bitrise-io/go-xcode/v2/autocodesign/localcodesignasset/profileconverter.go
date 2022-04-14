package localcodesignasset

import (
	"io/ioutil"

	"github.com/bitrise-io/go-xcode/profileutil"
	"github.com/bitrise-io/go-xcode/v2/autocodesign"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/devportalclient/appstoreconnect"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/devportalclient/time"
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
	content, err := ioutil.ReadFile(pth)
	if err != nil {
		return nil, err
	}

	return Profile{
		attributes: appstoreconnect.ProfileAttributes{
			Name:           info.Name,
			UUID:           info.UUID,
			ProfileContent: content,
			Platform:       getBundleIDPlatform(info.Type),
			ExpirationDate: time.Time(info.ExpirationDate),
		},
		id:             "", // only in case of Developer Portal Profiles
		bundleID:       info.BundleID,
		certificateIDs: nil, // only in case of Developer Portal Profiles
		deviceIDs:      nil, // only in case of Developer Portal Profiles
	}, nil
}

func getBundleIDPlatform(profileType profileutil.ProfileType) appstoreconnect.BundleIDPlatform {
	switch profileType {
	case profileutil.ProfileTypeIos, profileutil.ProfileTypeTvOs:
		return appstoreconnect.IOS
	case profileutil.ProfileTypeMacOs:
		return appstoreconnect.MacOS
	}

	return ""
}
