package profileutil

import (
	"strings"
	"time"

	"github.com/bitrise-io/steps-certificate-and-profile-installer/certificateutil"
	"github.com/bitrise-tools/go-xcode/exportoptions"
	"github.com/bitrise-tools/go-xcode/provisioningprofile"
	"github.com/pkg/errors"
)

// ProfileInfoModel ...
type ProfileInfoModel struct {
	Name                  string
	TeamIdentifier        string
	UUID                  string
	ApplicationIdentifier string
	BundleIdentifier      string
	ProvisionedDevices    []string
	ExpirationDate        time.Time
	ExportType            exportoptions.Method
	DeveloperCertificates []certificateutil.CertificateInfoModel
}

// IsXcodeManaged ...
func IsXcodeManaged(profileName string) bool {
	return strings.HasPrefix(profileName, "iOS Team Provisioning Profile") || strings.HasPrefix(profileName, "XC")
}

// IsXcodeManaged ...
func (prof ProfileInfoModel) IsXcodeManaged() bool {
	return IsXcodeManaged(prof.Name)
}

// IsExpired ...
func (prof ProfileInfoModel) IsExpired() bool {
	if prof.ExpirationDate.IsZero() {
		return false
	}

	return prof.ExpirationDate.Before(time.Now())
}

// HasInstalledCertificate ...
func (prof ProfileInfoModel) HasInstalledCertificate(installedCertificates []certificateutil.CertificateInfoModel) bool {
	has := false
	for _, certificate := range prof.DeveloperCertificates {
		for _, installedCertificate := range installedCertificates {
			if certificate.RawEndDate == installedCertificate.RawEndDate && certificate.RawSubject == installedCertificate.RawSubject {
				has = true
				break
			}
		}
	}
	return has
}

// ProfileFromFile ...
func ProfileFromFile(profilePth string) (ProfileInfoModel, error) {
	profile, err := provisioningprofile.NewProfileFromFile(profilePth)
	if err != nil {
		return ProfileInfoModel{}, err
	}

	profileModel := ProfileInfoModel{
		Name:                  profile.GetName(),
		UUID:                  profile.GetUUID(),
		TeamIdentifier:        profile.GetTeamID(),
		ExportType:            profile.GetExportMethod(),
		ExpirationDate:        profile.GetExpirationDate(),
		BundleIdentifier:      profile.GetBundleIdentifier(),
		ApplicationIdentifier: profile.GetApplicationIdentifier(),
	}

	if devicesList := profile.GetProvisionedDevices(); devicesList != nil {
		profileModel.ProvisionedDevices = devicesList
	}

	if certData := profile.GetDeveloperCertificates(); certData != nil {
		for _, cert := range certData {
			certModel, err := certificateutil.CertificateInfosFromDerContent(cert)
			if err != nil {
				return ProfileInfoModel{}, errors.Wrapf(err, "failed to parse profile's (%s)", profileModel.UUID)
			}
			profileModel.DeveloperCertificates = append(profileModel.DeveloperCertificates, certModel)
		}
	}

	return profileModel, nil
}
