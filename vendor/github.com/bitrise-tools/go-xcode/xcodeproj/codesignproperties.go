package xcodeproj

import (
	"bufio"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/command/rubyscript"
)

// CodeSignProperties ...
type CodeSignProperties struct {
	BundleIdentifier             string `json:"bundle_id"`
	ProvisioningStyle            string `json:"provisioning_style"`
	CodeSignIdentity             string `json:"code_sign_identity"`
	ProvisioningProfileSpecifier string `json:"provisioning_profile_specifier"`
	ProvisioningProfile          string `json:"provisioning_profile"`
}

const getCodeSignMappingScriptContent = `require 'xcodeproj'
require 'json'

def contained_projects(project_or_workspace_pth)
  project_paths = []
  if File.extname(project_or_workspace_pth) == '.xcodeproj'
    project_paths = [project_or_workspace_pth]
  else
    workspace_contents_pth = File.join(project_or_workspace_pth, 'contents.xcworkspacedata')
    workspace_contents = File.read(workspace_contents_pth)
    project_paths = workspace_contents.scan(/\"group:(.*)\"/).collect do |current_match|
      File.join(File.expand_path('..', project_or_workspace_pth), current_match.first)
    end.find_all do |current_match|
      # skip cocoapods projects
      !current_match.end_with?("Pods/Pods.xcodeproj")
    end
  end
  project_paths
end

def read_code_sign_map(project_or_workspace_pth)
  code_sign_map = {}

  project_paths = contained_projects(project_or_workspace_pth)
  project_paths.each do |project_path|
    project = Xcodeproj::Project.open(project_path)
    project.targets.each do |target|
      next if target.test_target_type?

      target.build_configuration_list.build_configurations.each do |build_configuration|
        attributes = project.root_object.attributes['TargetAttributes']
        target_id = target.uuid
        target_attributes = attributes[target_id]

        bundle_id = build_configuration.resolve_build_setting('PRODUCT_BUNDLE_IDENTIFIER') || ''
        provisioning_style = target_attributes['ProvisioningStyle'] || ''
        code_sign_identity = build_configuration.resolve_build_setting('CODE_SIGN_IDENTITY') || ''
        provisioning_profile_specifier = build_configuration.resolve_build_setting('PROVISIONING_PROFILE_SPECIFIER') || ''
        provisioning_profile = build_configuration.resolve_build_setting('PROVISIONING_PROFILE') || ''

        code_sign_map[target] = {
          bundle_id: bundle_id,
          provisioning_style: provisioning_style,
          code_sign_identity: code_sign_identity,
          provisioning_profile_specifier: provisioning_profile_specifier,
          provisioning_profile: provisioning_profile
        }
      end
    end
  end

  code_sign_map
end

begin
  project_path = ENV['project_path']
  mapping = read_code_sign_map(project_path)
  result = {
    data: mapping
  }
  result_json = result.to_json.to_s
  puts result_json
rescue => e
  error_message = e.to_s + "\n" + e.backtrace.to_s
  result = {
    error: error_message
  }
  result_json = result.to_json.to_s
  puts result_json
  exit(1)
end
`

const gemfileContent = `source "https://rubygems.org"
gem "xcodeproj"
gem "json"
`

func targetCodeSignMapping(projectPth string) (map[string]CodeSignProperties, error) {
	runner := rubyscript.New(getCodeSignMappingScriptContent)
	bundleInstallCmd, err := runner.BundleInstallCommand(gemfileContent, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create bundle install command, error: %s", err)
	}

	if out, err := bundleInstallCmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
		return nil, fmt.Errorf("bundle install failed, output: %s, error: %s", out, err)
	}

	runCmd, err := runner.RunScriptCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to create script runner command, error: %s", err)
	}
	runCmd.SetEnvs(append(runCmd.GetCmd().Env, "project_path="+projectPth)...)

	out, err := runCmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run ruby script, output: %s, error: %s", out, err)
	}

	// OutputModel ...
	type OutputModel struct {
		Data  map[string]CodeSignProperties `json:"data"`
		Error string                        `json:"error"`
	}
	var output OutputModel
	if err := json.Unmarshal([]byte(out), &output); err != nil {
		return nil, fmt.Errorf("failed to unmarshal output: %s", out)
	}

	if output.Error != "" {
		return nil, fmt.Errorf("failed to get provisioning profile - bundle id mapping, error: %s", output.Error)
	}

	return output.Data, nil
}

func parseBuildSettingsOut(out string) (map[string]string, error) {
	reader := strings.NewReader(out)
	scanner := bufio.NewScanner(reader)

	buildSettings := map[string]string{}
	isBuildSettings := false
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "Build settings for") {
			isBuildSettings = true
			continue
		}
		if !isBuildSettings {
			continue
		}

		split := strings.Split(line, " = ")
		if len(split) > 1 {
			key := strings.TrimSpace(split[0])
			value := strings.TrimSpace(strings.Join(split[1:], " = "))

			buildSettings[key] = value
		}
	}
	if err := scanner.Err(); err != nil {
		return map[string]string{}, err
	}

	return buildSettings, nil
}

func targetBuildSettings(projectPth, target string) (map[string]string, error) {
	args := []string{"-showBuildSettings"}
	if target != "" {
		args = append(args, "-target", target)
	}

	cmd := command.New("xcodebuild", args...)
	cmd.SetDir(filepath.Dir(projectPth))

	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return map[string]string{}, err
	}

	return parseBuildSettingsOut(out)
}

// TargetCodeSignMapping ...
func TargetCodeSignMapping(projectPth string) (map[string]CodeSignProperties, error) {
	mapping, err := targetCodeSignMapping(projectPth)
	if err != nil {
		return nil, err
	}

	codeSignMapping := map[string]CodeSignProperties{}
	for target, codeSignProperties := range mapping {
		buildSettings, err := targetBuildSettings(projectPth, target)
		if err != nil {
			return nil, fmt.Errorf("failed to read project build settings, error: %s", err)
		}

		bundleID := buildSettings["PRODUCT_BUNDLE_IDENTIFIER"]
		codeSignIdentity := buildSettings["CODE_SIGN_IDENTITY"]

		provisioningStyle := codeSignProperties.ProvisioningStyle
		provisioningProfileSpecifier := codeSignProperties.ProvisioningProfileSpecifier
		provisioningProfile := codeSignProperties.ProvisioningProfile

		if provisioningStyle == "" && provisioningProfile == "" && provisioningProfileSpecifier == "" {
			provisioningStyle = "Automatic"
		}

		properties := CodeSignProperties{
			BundleIdentifier:             bundleID,
			ProvisioningStyle:            provisioningStyle,
			CodeSignIdentity:             codeSignIdentity,
			ProvisioningProfileSpecifier: provisioningProfileSpecifier,
			ProvisioningProfile:          provisioningProfile,
		}

		codeSignMapping[target] = properties
	}

	return codeSignMapping, nil
}
