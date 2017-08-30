package provisioningprofile

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-tools/go-xcode/exportoptions"
	"github.com/bitrise-tools/go-xcode/plistutil"
)

const (
	notValidParameterErrorMessage = "security: SecPolicySetValue: One or more parameters passed to a function were not valid."
)

// NewPlistDataFromFile ...
func NewPlistDataFromFile(provisioningProfilePth string) (plistutil.PlistData, error) {
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

	return plistutil.NewPlistDataFromContent(out)
}

// GetExportMethod ...
func GetExportMethod(data plistutil.PlistData) exportoptions.Method {
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

// GetDeveloperTeam ...
func GetDeveloperTeam(data plistutil.PlistData) string {
	entitlements, ok := data.GetMapStringInterface("Entitlements")
	if !ok {
		return ""
	}

	teamID, ok := entitlements.GetString("com.apple.developer.team-identifier")
	if !ok {
		return ""
	}
	return teamID
}
