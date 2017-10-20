package profileutil

import (
	"crypto/x509"
	"errors"
	"strings"
	"time"

	"github.com/bitrise-tools/go-xcode/certificateutil"
	"github.com/bitrise-tools/go-xcode/exportoptions"
	"github.com/bitrise-tools/go-xcode/plistutil"
	"github.com/fullsailor/pkcs7"
	"howett.net/plist"
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
	ExpirationDate        time.Time
	Entitlements          plistutil.PlistData
}

// IsXcodeManaged ...
func IsXcodeManaged(profileName string) bool {
	return strings.HasPrefix(profileName, "XC") || (strings.HasPrefix(profileName, "iOS Team") && strings.Contains(profileName, "Provisioning Profile"))
}

// IsXcodeManaged ...
func (info ProvisioningProfileInfoModel) IsXcodeManaged() bool {
	return IsXcodeManaged(info.Name)
}

// IsExpired ...
func (info ProvisioningProfileInfoModel) IsExpired() bool {
	if info.ExpirationDate.IsZero() {
		return false
	}

	return info.ExpirationDate.Before(time.Now())
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
	}
	return has
}

// NewProvisioningProfileInfo ...
func NewProvisioningProfileInfo(provisioningProfile pkcs7.PKCS7) (ProvisioningProfileInfoModel, error) {
	var data plistutil.PlistData
	if _, err := plist.Unmarshal(provisioningProfile.Content, &data); err != nil {
		return ProvisioningProfileInfoModel{}, err
	}

	teamName, _ := data.GetString("TeamName")
	profile := PlistData(data)
	info := ProvisioningProfileInfoModel{
		UUID:           profile.GetUUID(),
		Name:           profile.GetName(),
		TeamName:       teamName,
		TeamID:         profile.GetTeamID(),
		BundleID:       profile.GetBundleIdentifier(),
		ExportType:     profile.GetExportMethod(),
		ExpirationDate: profile.GetExpirationDate(),
	}

	if devicesList := profile.GetProvisionedDevices(); devicesList != nil {
		info.ProvisionedDevices = devicesList
	}

	developerCertificates, found := data.GetByteArrayArray("DeveloperCertificates")
	if found {
		certificates := []*x509.Certificate{}
		for _, certificateBytes := range developerCertificates {
			certificate, err := certificateutil.CertificateFromDERContent(certificateBytes)
			if err == nil && certificate != nil {
				certificates = append(certificates, certificate)
			}
		}
		info.DeveloperCertificates = certificateutil.CertificateInfos(certificates)
	}

	info.Entitlements = profile.GetEntitlements()

	return info, nil
}

// NewProvisioningProfileInfoFromFile ...
func NewProvisioningProfileInfoFromFile(pth string) (ProvisioningProfileInfoModel, error) {
	provisioningProfile, err := ProvisioningProfileFromFile(pth)
	if err != nil {
		return ProvisioningProfileInfoModel{}, err
	}
	if provisioningProfile != nil {
		return NewProvisioningProfileInfo(*provisioningProfile)
	}
	return ProvisioningProfileInfoModel{}, errors.New("failed to parse provisioning profile infos")
}

// InstalledIosProvisioningProfileInfos ...
func InstalledIosProvisioningProfileInfos() ([]ProvisioningProfileInfoModel, error) {
	provisioningProfiles, err := InstalledIosProvisioningProfiles()
	if err != nil {
		return nil, err
	}

	infos := []ProvisioningProfileInfoModel{}
	for _, provisioningProfile := range provisioningProfiles {
		if provisioningProfile != nil {
			info, err := NewProvisioningProfileInfo(*provisioningProfile)
			if err != nil {
				return nil, err
			}
			infos = append(infos, info)
		}
	}
	return infos, nil
}
