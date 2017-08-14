package provisioningprofile

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/command/rubyscript"
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

const getBundleIDProfileMappingScriptContent = `require 'xcodeproj'
require 'json'

def contained_projects(project_or_workspace_pth)
  project_paths = []
  if File.extname(project_or_workspace_pth) == '.xcodeproj'
    project_paths = [project_or_workspace_pth]
  else
    workspace_contents_pth = File.join(project_or_workspace_pth, 'contents.xcworkspacedata')
    workspace_contents = File.read(workspace_contents_pth)
    project_paths = workspace_contents.scan(/\"group:(.*)\"/).collect do |current_match|
      # skip cocoapods projects
      return nil if current_match.end_with?('Pods/Pods.xcodeproj')

      File.join(File.expand_path('..', project_or_workspace_pth), current_match.first)
    end
  end
  project_paths
end

def get_bundle_id_provisioning_profile_mapping(project_or_workspace_pth)
  bundle_id_provisioning_profile_map = {}

  project_paths = contained_projects(project_or_workspace_pth)
  project_paths.each do |project_path|
    target_bundle_ids = []

    project = Xcodeproj::Project.open(project_path)
    project.targets.each do |target|
      next if target.test_target_type?

      target.build_configuration_list.build_configurations.each do |build_configuration|
        bundle_identifier = build_configuration.resolve_build_setting("PRODUCT_BUNDLE_IDENTIFIER")
        provisioning_profile_specifier = build_configuration.resolve_build_setting("PROVISIONING_PROFILE_SPECIFIER")

        next if provisioning_profile_specifier.to_s.length == 0
        
        bundle_id_provisioning_profile_map[bundle_identifier] = provisioning_profile_specifier
      end
    end
  end

  bundle_id_provisioning_profile_map
end

begin
  mapping = get_bundle_id_provisioning_profile_mapping('/Users/godrei/Develop/iOS/sample-apps-ios-simple-objc/ios-simple-objc/ios-simple-objc.xcodeproj')
  puts "#{{ :data =>  mapping }.to_json}"
rescue => e
	puts "#{{ :error => e.to_s }.to_json}"
end`

const getBundleIDProfileMappingGemfileContent = `source "https://rubygems.org"
gem "xcodeproj"
gem "json"
`

// BundleIDProvisionigProfileMapping ...
func BundleIDProvisionigProfileMapping(projectPth string) (map[string]string, error) {
	runner := rubyscript.New(getBundleIDProfileMappingScriptContent)
	bundleInstallCmd, err := runner.BundleInstallCommand(getBundleIDProfileMappingGemfileContent, "")
	if err != nil {
		return map[string]string{}, err
	}

	if out, err := bundleInstallCmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
		return map[string]string{}, fmt.Errorf("bundle install failed, output: %s, error: %s", out, err)
	}

	runCmd, err := runner.RunScriptCommand()
	if err != nil {
		return map[string]string{}, err
	}

	out, err := runCmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return map[string]string{}, err
	}

	type OutputModel struct {
		Data  map[string]string
		Error string
	}
	var output OutputModel
	if err := json.Unmarshal([]byte(out), &output); err != nil {
		return map[string]string{}, err
	}

	if output.Error != "" {
		return map[string]string{}, fmt.Errorf("failed to get provisioning profile - bundle id mapping, error: %s", output.Error)
	}

	return output.Data, nil
}
