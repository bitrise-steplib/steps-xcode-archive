package profileutil

import (
	"fmt"
	"time"

	"github.com/bitrise-io/go-plist"
	"github.com/bitrise-io/go-xcode/certificateutil"
	"github.com/bitrise-io/go-xcode/exportoptions"
	"github.com/bitrise-io/go-xcode/v2/plistutil"
	"github.com/fullsailor/pkcs7"
)

// ProfileType ...
type ProfileType string

// ProfileTypes ...
const (
	ProfileTypeIos   ProfileType = "ios"
	ProfileTypeMacOs ProfileType = "osx"
	ProfileTypeTvOs  ProfileType = "tvos"
)

// ProvisioningProfileInfoModel ...
type ProvisioningProfileInfoModel struct {
	UUID                  string
	Name                  string
	TeamName              string
	TeamID                string
	BundleID              string
	ExportType            exportoptions.Method
	ProvisionedDevices    []string
	DeveloperCertificates []certificateutil.CertificateInfoModel
	CreationDate          time.Time
	ExpirationDate        time.Time
	Entitlements          plistutil.PlistData
	ProvisionsAllDevices  bool
	Type                  ProfileType
}

// NewProvisioningProfileInfo ...
func NewProvisioningProfileInfo(profilePKCS7 pkcs7.PKCS7) (ProvisioningProfileInfoModel, error) {
	var data plistutil.PlistData
	if _, err := plist.Unmarshal(profilePKCS7.Content, &data); err != nil {
		return ProvisioningProfileInfoModel{}, err
	}
	profile := PlistData(data)

	profileType, err := profile.GetProfileType()
	if err != nil {
		return ProvisioningProfileInfoModel{}, err
	}

	info := ProvisioningProfileInfoModel{
		Type:                  profileType,
		UUID:                  profile.GetUUID(),
		Name:                  profile.GetName(),
		TeamName:              profile.GetTeamName(),
		TeamID:                profile.GetTeamID(),
		BundleID:              profile.GetBundleIdentifier(),
		CreationDate:          profile.GetCreationDate(),
		ExpirationDate:        profile.GetExpirationDate(),
		ProvisionsAllDevices:  profile.GetProvisionsAllDevices(),
		ExportType:            profile.GetExportMethod(),
		ProvisionedDevices:    profile.GetProvisionedDevices(),
		DeveloperCertificates: profile.GetDeveloperCertificateInfo(),
		Entitlements:          profile.GetEntitlements(),
	}

	return info, nil
}

// NewProvisioningProfileInfoFromPKCS7Content ...
func NewProvisioningProfileInfoFromPKCS7Content(content []byte) (ProvisioningProfileInfoModel, error) {
	profilePKCS7, err := pkcs7.Parse(content)
	if err != nil {
		return ProvisioningProfileInfoModel{}, err
	}

	return NewProvisioningProfileInfo(*profilePKCS7)
}

// IsXcodeManaged ...
func (info ProvisioningProfileInfoModel) IsXcodeManaged() bool {
	return IsXcodeManaged(info.Name)
}

// CheckValidity ...
func (info ProvisioningProfileInfoModel) CheckValidity(currentTime func() time.Time) error {
	timeNow := currentTime()
	if !timeNow.Before(info.ExpirationDate) {
		return fmt.Errorf("provisioning profile is not valid anymore, validity ended at: %s", info.ExpirationDate)
	}
	return nil
}

// HasInstalledCertificate ...
func (info ProvisioningProfileInfoModel) HasInstalledCertificate(installedCertificates []certificateutil.CertificateInfoModel) bool {
	has := false
	for _, certificate := range info.DeveloperCertificates {
		for _, installedCertificate := range installedCertificates {
			if certificate.Serial == installedCertificate.Serial {
				has = true
				break
			}
		}
		if has {
			break
		}
	}
	return has
}
