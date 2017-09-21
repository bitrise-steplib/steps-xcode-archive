package xcodeproj

import (
	"bufio"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/command/rubyscript"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-tools/go-xcode/plistutil"
	"github.com/pkg/errors"
)

// CodeSignInfo ...
type CodeSignInfo struct {
	ProjectPth                   string `json:"project"`
	InfoPlistPth                 string `json:"info_plist_file"`
	Configuration                string `json:"configuration"`
	BundleIdentifier             string `json:"bundle_id"`
	ProvisioningStyle            string `json:"provisioning_style"`
	CodeSignIdentity             string `json:"code_sign_identity"`
	ProvisioningProfileSpecifier string `json:"provisioning_profile_specifier"`
	ProvisioningProfile          string `json:"provisioning_profile"`
}

func getCodeSignInfoWithXcodeprojGem(projectPth, scheme, configuration, user string) (map[string]CodeSignInfo, error) {
	runner := rubyscript.New(codeSignInfoScriptContent)
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

	envsToAppend := []string{
		"project=" + projectPth,
		"scheme=" + scheme,
		"configuration=" + configuration,
		"user=" + user}
	envs := append(runCmd.GetCmd().Env, envsToAppend...)

	runCmd.SetEnvs(envs...)

	out, err := runCmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run ruby script, output: %s, error: %s", out, err)
	}

	// OutputModel ...
	type OutputModel struct {
		Data  map[string]CodeSignInfo `json:"data"`
		Error string                  `json:"error"`
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
		return map[string]string{}, errors.Wrap(err, "Failed to scan build settings")
	}

	return buildSettings, nil
}

func getTargetBuildSettingsWithXcodebuild(projectPth, target, configuration string) (map[string]string, error) {
	args := []string{"-showBuildSettings", "-project", projectPth, "-target", target}
	if configuration != "" {
		args = append(args, "-configuration", configuration)
	}

	cmd := command.New("xcodebuild", args...)
	cmd.SetDir(filepath.Dir(projectPth))

	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		if errorutil.IsExitStatusError(err) {
			return map[string]string{}, errors.Wrapf(err, "%s failed with output: %s", cmd.PrintableCommandArgs(), out)
		}
		return map[string]string{}, errors.Wrapf(err, "%s failed", cmd.PrintableCommandArgs())
	}

	return parseBuildSettingsOut(out)
}

func getBundleIDWithPlistbuddy(infoPlistPth string) (string, error) {
	plistData, err := plistutil.NewPlistDataFromFile(infoPlistPth)
	if err != nil {
		return "", err
	}

	bundleID, _ := plistData.GetString("CFBundleIdentifier")
	return bundleID, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// ResolveCodeSignInfo ...
func ResolveCodeSignInfo(projectOrWorkspacePth, scheme, configuration, user string) (map[string]CodeSignInfo, error) {
	targetCodeSignInfoMap, err := getCodeSignInfoWithXcodeprojGem(projectOrWorkspacePth, scheme, configuration, user)
	if err != nil {
		return nil, err
	}

	resolvedCodeSignInfoMap := map[string]CodeSignInfo{}
	for target, codeSignInfo := range targetCodeSignInfoMap {
		if target == "" {
			return nil, errors.New("target name is empty")
		}

		projectPth := codeSignInfo.ProjectPth
		if projectPth == "" {
			return nil, fmt.Errorf("failed to resolve which project contains target: %s", target)
		}

		configuration = codeSignInfo.Configuration

		buildSettings, err := getTargetBuildSettingsWithXcodebuild(projectPth, target, configuration)
		if err != nil {
			return nil, fmt.Errorf("failed to read project build settings, error: %s", err)
		}

		// resolve Info.plist path
		infoPlistPth := buildSettings["INFOPLIST_FILE"]
		if infoPlistPth != "" {
			projectDir := filepath.Dir(projectPth)
			infoPlistPth = filepath.Join(projectDir, infoPlistPth)
		}
		if infoPlistPth == "" && codeSignInfo.InfoPlistPth != "" && !strings.Contains(codeSignInfo.InfoPlistPth, "$") {
			infoPlistPth = codeSignInfo.InfoPlistPth
		}
		// ---

		// resolve bundle id
		// best case if it presents in the buildSettings, since it is expanded
		bundleID := buildSettings["PRODUCT_BUNDLE_IDENTIFIER"]
		if bundleID == "" && codeSignInfo.BundleIdentifier != "" && !strings.Contains(codeSignInfo.BundleIdentifier, "$") {
			// bundle id not presents in -showBuildSettings output
			// use the bundle id parsed from the project file, unless it contains env var
			bundleID = codeSignInfo.BundleIdentifier
		}
		if bundleID == "" && infoPlistPth != "" {
			// try to find the bundle id in the Info.plist file, unless it contains env var
			id, err := getBundleIDWithPlistbuddy(infoPlistPth)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve bundle id, error: %s", err)
			}
			if !strings.Contains(codeSignInfo.BundleIdentifier, "$") {
				bundleID = id
			}
		}
		if bundleID == "" {
			return nil, fmt.Errorf("failed to resolve bundle id")
		}
		// ---

		provisioningStyle := firstNonEmpty(buildSettings["CODE_SIGN_STYLE"], codeSignInfo.ProvisioningStyle)
		codeSignIdentity := firstNonEmpty(buildSettings["CODE_SIGN_IDENTITY"], codeSignInfo.CodeSignIdentity)
		provisioningProfileSpecifier := firstNonEmpty(buildSettings["PROVISIONING_PROFILE_SPECIFIER"], codeSignInfo.ProvisioningProfileSpecifier)
		provisioningProfile := firstNonEmpty(buildSettings["PROVISIONING_PROFILE"], codeSignInfo.ProvisioningProfile)

		resolvedCodeSignInfo := CodeSignInfo{
			InfoPlistPth:                 infoPlistPth,
			ProjectPth:                   projectPth,
			Configuration:                configuration,
			BundleIdentifier:             bundleID,
			ProvisioningStyle:            provisioningStyle,
			CodeSignIdentity:             codeSignIdentity,
			ProvisioningProfileSpecifier: provisioningProfileSpecifier,
			ProvisioningProfile:          provisioningProfile,
		}

		resolvedCodeSignInfoMap[target] = resolvedCodeSignInfo
	}

	return resolvedCodeSignInfoMap, nil
}
