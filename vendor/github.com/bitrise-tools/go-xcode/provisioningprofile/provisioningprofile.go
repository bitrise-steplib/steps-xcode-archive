package provisioningprofile

import (
	"fmt"
	"strings"

	plist "github.com/DHowett/go-plist"
	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-tools/go-xcode/exportoptions"
)

const (
	notValidParameterErrorMessage = "security: SecPolicySetValue: One or more parameters passed to a function were not valid."
)

// EntitlementsModel ...
type EntitlementsModel struct {
	GetTaskAllow    *bool   `plist:"get-task-allow"`
	DeveloperTeamID *string `plist:"com.apple.developer.team-identifier"`
}

// Model ...
type Model struct {
	Name                 *string            `plist:"Name"`
	ProvisionedDevices   *[]string          `plist:"ProvisionedDevices"`
	ProvisionsAllDevices *bool              `plist:"ProvisionsAllDevices"`
	Entitlements         *EntitlementsModel `plist:"Entitlements"`
}

func newFromProfileContent(content string) (Model, error) {
	var mobileProvision Model
	if _, err := plist.Unmarshal([]byte(content), &mobileProvision); err != nil {
		return Model{}, fmt.Errorf("failed to mobileprovision, error: %s", err)
	}

	return mobileProvision, nil
}

// NewFromFile ...
func NewFromFile(pth string) (Model, error) {
	cmd := cmdex.NewCommand("security", "cms", "-D", "-i", pth)

	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return Model{}, fmt.Errorf("command failed, error: %s", err)
	}

	outSplit := strings.Split(out, "\n")
	if len(outSplit) > 0 {
		if strings.Contains(outSplit[0], notValidParameterErrorMessage) {
			fixedOutSplit := outSplit[1:len(outSplit)]
			out = strings.Join(fixedOutSplit, "\n")
		}
	}

	return newFromProfileContent(out)
}

// GetExportMethod ...
func (profile Model) GetExportMethod() exportoptions.Method {
	method := exportoptions.MethodDefault
	if profile.ProvisionedDevices == nil {
		if profile.ProvisionsAllDevices != nil && *profile.ProvisionsAllDevices {
			method = exportoptions.MethodEnterprise
		} else {
			method = exportoptions.MethodAppStore
		}
	} else if profile.Entitlements != nil {
		entitlements := *profile.Entitlements
		if entitlements.GetTaskAllow != nil && *entitlements.GetTaskAllow {
			method = exportoptions.MethodDevelopment
		} else {
			method = exportoptions.MethodAdHoc
		}
	}
	return method
}

// GetDeveloperTeam ...
func (profile Model) GetDeveloperTeam() string {
	developerTeamID := ""
	if profile.Entitlements != nil {
		entitlements := *profile.Entitlements
		if entitlements.DeveloperTeamID != nil {
			developerTeamID = *entitlements.DeveloperTeamID
		}
	}
	return developerTeamID
}
