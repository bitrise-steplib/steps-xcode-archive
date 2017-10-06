package provisioningprofile

import (
	"fmt"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-tools/go-xcode/exportoptions"
	"github.com/bitrise-tools/go-xcode/plistutil"
)

const (
	notValidParameterErrorMessage = "security: SecPolicySetValue: One or more parameters passed to a function were not valid."
)

// Profile ...
type Profile plistutil.PlistData

// NewProfileFromFile ...
func NewProfileFromFile(provisioningProfilePth string) (Profile, error) {
	cmd := command.New("security", "cms", "-D", "-i", provisioningProfilePth)

	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("command failed, error: %s", err)
	}

	// fix: security: SecPolicySetValue: One or more parameters passed to a function were not valid.
	outSplit := strings.Split(out, "\n")
	if len(outSplit) > 0 {
		if strings.Contains(outSplit[0], notValidParameterErrorMessage) {
			fixedOutSplit := outSplit[1:len(outSplit)]
			out = strings.Join(fixedOutSplit, "\n")
		}
	}
	// ---

	plistData, err := plistutil.NewPlistDataFromContent(out)
	if err != nil {
		return Profile{}, err
	}
	return Profile(plistData), nil
}

// GetUUID ...
func (profile Profile) GetUUID() string {
	data := plistutil.PlistData(profile)
	uuid, _ := data.GetString("UUID")
	return uuid
}

// GetName ...
func (profile Profile) GetName() string {
	data := plistutil.PlistData(profile)
	uuid, _ := data.GetString("Name")
	return uuid
}

// GetApplicationIdentifier ...
func (profile Profile) GetApplicationIdentifier() string {
	data := plistutil.PlistData(profile)
	entitlements, ok := data.GetMapStringInterface("Entitlements")
	if !ok {
		return ""
	}

	applicationID, ok := entitlements.GetString("application-identifier")
	if !ok {
		return ""
	}
	return applicationID
}

// GetBundleIdentifier ...
func (profile Profile) GetBundleIdentifier() string {
	applicationID := profile.GetApplicationIdentifier()
	teamID := profile.GetTeamID()
	return strings.TrimPrefix(applicationID, teamID+".")
}

// GetExportMethod ...
func (profile Profile) GetExportMethod() exportoptions.Method {
	data := plistutil.PlistData(profile)
	_, ok := data.GetStringArray("ProvisionedDevices")
	if !ok {
		if allDevices, ok := data.GetBool("ProvisionsAllDevices"); ok && allDevices {
			return exportoptions.MethodEnterprise
		}
		return exportoptions.MethodAppStore
	}

	entitlements, ok := data.GetMapStringInterface("Entitlements")
	if ok {
		if allow, ok := entitlements.GetBool("get-task-allow"); ok && allow {
			return exportoptions.MethodDevelopment
		}
		return exportoptions.MethodAdHoc
	}

	return exportoptions.MethodDefault
}

// GetTeamID ...
func (profile Profile) GetTeamID() string {
	data := plistutil.PlistData(profile)
	entitlements, ok := data.GetMapStringInterface("Entitlements")
	if ok {
		teamID, _ := entitlements.GetString("com.apple.developer.team-identifier")
		return teamID
	}
	return ""
}

// GetExpirationDate ...
func (profile Profile) GetExpirationDate() time.Time {
	data := plistutil.PlistData(profile)
	expiry, _ := data.GetTime("ExpirationDate")
	return expiry
}

// GetProvisionedDevices ...
func (profile Profile) GetProvisionedDevices() []string {
	data := plistutil.PlistData(profile)
	devices, _ := data.GetStringArray("ProvisionedDevices")
	return devices
}

// GetDeveloperCertificates ...
func (profile Profile) GetDeveloperCertificates() [][]byte {
	data := plistutil.PlistData(profile)
	developerCertificates, _ := data.GetByteArrayArray("DeveloperCertificates")
	return developerCertificates
}
