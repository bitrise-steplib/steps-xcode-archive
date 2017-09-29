package profileutil

import (
	"fmt"
	"strings"
	"time"

	"github.com/bitrise-io/steps-certificate-and-profile-installer/certificateutil"
	"github.com/bitrise-tools/go-xcode/exportoptions"
	"github.com/bitrise-tools/go-xcode/plistutil"
	"github.com/bitrise-tools/go-xcode/provisioningprofile"
)

// ProfileModel ...
type ProfileModel struct {
	Name                  string
	TeamIdentifier        string
	UUID                  string
	ApplicationIdentifier string
	BundleIdentifier      string
	ProvisionedDevices    []string
	ExpirationDate        time.Time
	ExportType            exportoptions.Method
	DeveloperCertificates []certificateutil.CertificateInfosModel
}

// ProfileFromFile ...
func ProfileFromFile(provPath string) (ProfileModel, error) {
	profile, err := provisioningprofile.NewProfileFromFile(provPath)
	if err != nil {
		return ProfileModel{}, err
	}

	profileModel := ProfileModel{
		Name:                  profile.GetName(),
		UUID:                  profile.GetUUID(),
		TeamIdentifier:        profile.GetTeamID(),
		ExportType:            profile.GetExportMethod(),
		ExpirationDate:        profile.GetExpirationDate(),
		BundleIdentifier:      profile.GetBundleIdentifier(),
		ApplicationIdentifier: profile.GetApplicationIdentifier(),
	}

	profilePlistData := plistutil.PlistData(profile)

	if profile.GetExportMethod() == exportoptions.MethodDevelopment {
		if devicesList, ok := profilePlistData.GetStringArray("ProvisionedDevices"); ok {
			profileModel.ProvisionedDevices = devicesList
		}
	}

	if certData, ok := GetByteArray(profilePlistData, "DeveloperCertificates"); ok {
		for _, cert := range certData {
			certModel, err := certificateutil.CertificateInfosFromDerContent(cert)
			if err != nil {
				return ProfileModel{}, fmt.Errorf("Failed to get certificate info from profile(%s), error: %s", profileModel.UUID, err)
			}
			profileModel.DeveloperCertificates = append(profileModel.DeveloperCertificates, certModel)
		}
	}

	return profileModel, nil
}

// GetByteArray ...
func GetByteArray(data plistutil.PlistData, forKey string) ([][]byte, bool) {
	value, ok := data[forKey]
	if !ok {
		return nil, false
	}

	if casted, ok := value.([][]byte); ok {
		return casted, true
	}

	casted, ok := value.([]interface{})
	if !ok {
		return nil, false
	}

	array := [][]byte{}
	for _, v := range casted {
		casted, ok := v.([]byte)
		if !ok {
			return nil, false
		}

		array = append(array, casted)
	}
	return array, true
}

func (profileModel ProfileModel) String() string {
	certInfoString := ""

	certInfoString += fmt.Sprintf("- BundleIdentifier: %s\n", profileModel.BundleIdentifier)
	certInfoString += fmt.Sprintf("- ExpirationDate: %s\n", profileModel.ExpirationDate)
	certInfoString += fmt.Sprintf("- ExportType: %s\n", profileModel.ExportType)
	certInfoString += fmt.Sprintf("- TeamIdentifier: %s\n", profileModel.TeamIdentifier)
	certInfoString += fmt.Sprintf("- UUID: %s\n", profileModel.UUID)

	if len(profileModel.DeveloperCertificates) > 0 {
		certInfoString += fmt.Sprintf("- DeveloperCertificates: \n")
		for _, devCert := range profileModel.DeveloperCertificates {
			certInfoString += fmt.Sprintf("  %s\n", devCert.CommonName)
			certInfoString += fmt.Sprintf("  - TeamID: %s\n", devCert.TeamID)
			certInfoString += fmt.Sprintf("  - EndDate: %s\n", devCert.EndDate)
			certInfoString += fmt.Sprintf("  - IsDevelopment: %t", devCert.IsDevelopement)
		}
	}

	if len(profileModel.ProvisionedDevices) > 0 {
		certInfoString += "\n"
		redactedDeviceList := []string{}
		for _, deviceUUID := range profileModel.ProvisionedDevices {
			redactedDeviceList = append(redactedDeviceList, fmt.Sprintf("%s...%s", deviceUUID[:3], deviceUUID[len(deviceUUID)-3:]))
		}
		certInfoString += fmt.Sprintf("- ProvisionedDevices: %s", strings.Join(redactedDeviceList, ", "))
	}

	return certInfoString
}
