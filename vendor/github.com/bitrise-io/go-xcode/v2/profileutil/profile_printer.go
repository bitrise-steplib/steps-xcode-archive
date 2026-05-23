package profileutil

import (
	"encoding/json"
	"fmt"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/certificateutil"
	"github.com/bitrise-io/go-xcode/v2/plistutil"
)

// ProfilePrinter ...
type ProfilePrinter struct {
	logger       log.Logger
	timeProvider TimeProvider
}

// NewProfilePrinter ...
func NewProfilePrinter(logger log.Logger, timeProvider TimeProvider) *ProfilePrinter {
	return &ProfilePrinter{
		logger:       logger,
		timeProvider: timeProvider,
	}
}

// PrintableProfile ...
func (printer ProfilePrinter) PrintableProfile(profile ProvisioningProfileInfoModel, installedCertificates ...certificateutil.CertificateInfoModel) string {
	printable := map[string]any{}
	printable["name"] = fmt.Sprintf("%s (%s)", profile.Name, profile.UUID)
	printable["export_type"] = string(profile.ExportType)
	printable["team"] = fmt.Sprintf("%s (%s)", profile.TeamName, profile.TeamID)
	printable["bundle_id"] = profile.BundleID
	printable["expiry"] = profile.ExpirationDate.String()
	printable["is_xcode_managed"] = profile.IsXcodeManaged()

	printable["capabilities"] = collectCapabilitiesPrintableInfo(profile.Entitlements)

	if profile.ProvisionedDevices != nil {
		printable["devices"] = profile.ProvisionedDevices
	}

	var certificates []map[string]any
	for _, certificateInfo := range profile.DeveloperCertificates {
		certificate := map[string]any{}
		certificate["name"] = certificateInfo.CommonName
		certificate["serial"] = certificateInfo.Serial
		certificate["team_id"] = certificateInfo.TeamID
		certificates = append(certificates, certificate)
	}
	printable["certificates"] = certificates

	var errs []string
	if installedCertificates != nil && !profile.HasInstalledCertificate(installedCertificates) {
		errs = append(errs, "none of the profile's certificates are installed")
	}

	if err := profile.CheckValidity(printer.timeProvider.Now); err != nil {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		printable["errors"] = errs
	}

	data, err := json.MarshalIndent(printable, "", "\t")
	if err != nil {
		printer.logger.Errorf("Failed to marshal: %v, error: %s", printable, err)
		return ""
	}

	return string(data)
}

func collectCapabilitiesPrintableInfo(entitlements plistutil.PlistData) map[string]any {
	capabilities := map[string]any{}

	for key, value := range entitlements {
		if KnownProfileCapabilitiesMap[ProfileTypeIos][key] ||
			KnownProfileCapabilitiesMap[ProfileTypeMacOs][key] {
			capabilities[key] = value
		}
	}

	return capabilities
}
