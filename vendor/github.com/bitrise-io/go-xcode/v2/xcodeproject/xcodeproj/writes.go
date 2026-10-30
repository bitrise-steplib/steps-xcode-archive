package xcodeproj

import (
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"

	"github.com/bitrise-io/go-plist"
	"github.com/bitrise-io/go-xcode/v2/internal/filewriter"
	"github.com/bitrise-io/go-xcode/v2/xcodeproject/serialized"
)

// newPBXProjFileMode is the mode of a newly created project.pbxproj; an existing one keeps its mode.
const newPBXProjFileMode = 0644

// ForceCodeSignOptions are the manual code signing settings ForceCodeSign applies.
type ForceCodeSignOptions struct {
	TargetName              string
	Configuration           string
	DevelopmentTeam         string
	CodesignIdentity        string
	ProvisioningProfileUUID string
}

// ForceCodeSign switches a target configuration to manual code signing. Call Save to persist it.
// It sets CODE_SIGN_STYLE, DEVELOPMENT_TEAM, CODE_SIGN_IDENTITY and PROVISIONING_PROFILE, including
// their existing [sdk=...] variants, clears PROVISIONING_PROFILE_SPECIFIER, and updates the
// target's TargetAttributes entry if there is one.
func (p *XcodeProj) ForceCodeSign(opts ForceCodeSignOptions) error {
	target, ok := p.TargetByName(opts.TargetName)
	if !ok {
		return fmt.Errorf("failed to find target with name: %s", opts.TargetName)
	}

	buildSettings, err := targetBuildSettings(target, opts.Configuration)
	if err != nil {
		return err
	}

	forced := map[string]string{
		"CODE_SIGN_STYLE":                "Manual",
		"DEVELOPMENT_TEAM":               opts.DevelopmentTeam,
		"CODE_SIGN_IDENTITY":             opts.CodesignIdentity,
		"PROVISIONING_PROFILE_SPECIFIER": "",
		"PROVISIONING_PROFILE":           opts.ProvisioningProfileUUID,
	}
	for key, value := range forced {
		writeBuildSettingForAllSDKs(buildSettings, key, value)
	}

	if attributes, ok := p.targetAttributes.Object(target.ID); ok {
		attributes["ProvisioningStyle"] = "Manual"
		attributes["DevelopmentTeam"] = opts.DevelopmentTeam
		attributes["DevelopmentTeamName"] = ""
	}

	return nil
}

// SetBuildSetting sets a build setting of a target configuration, such as CURRENT_PROJECT_VERSION or
// PRODUCT_BUNDLE_IDENTIFIER, in the project file. Several settings can be set before one Save
// persists them. Targets fetched earlier see the change, as they share the build settings.
func (p *XcodeProj) SetBuildSetting(targetName, configurationName, key string, value any) error {
	target, ok := p.TargetByName(targetName)
	if !ok {
		return fmt.Errorf("failed to find target with name: %s", targetName)
	}

	buildSettings, err := targetBuildSettings(target, configurationName)
	if err != nil {
		return err
	}

	buildSettings[key] = value

	return nil
}

// Save writes project.pbxproj, rewriting only the changed objects so the rest of the file stays
// byte for byte. If that is not possible, it writes the whole file and logs a warning.
func (p *XcodeProj) Save() error {
	pth := filepath.Join(p.Path, pbxProjFileName)

	content, err := p.perObjectModify()
	if err != nil {
		p.logger.Warnf("failed to modify project in-place: %v", err)

		content, err = plist.MarshalIndent(p.rawProj, p.format, "\t")
		if err != nil {
			return fmt.Errorf("failed to marshal .pbxproj: %w", err)
		}
	}

	if err := filewriter.WriteKeepingMode(p.fileManager, pth, content, newPBXProjFileMode); err != nil {
		return fmt.Errorf("failed to write %s: %w", pth, err)
	}

	return nil
}

func targetBuildSettings(target Target, configurationName string) (serialized.Object, error) {
	for _, configuration := range target.BuildConfigurations {
		if configuration.Name == configurationName {
			return configuration.buildSettings, nil
		}
	}

	return nil, fmt.Errorf("failed to find build configuration %s of target %s", configurationName, target.Name)
}

// writeBuildSettingForAllSDKs also updates the existing [sdk=...] variants of key.
func writeBuildSettingForAllSDKs(buildSettings serialized.Object, key, value string) {
	buildSettings[key] = value

	sdkVariant := regexp.MustCompile(fmt.Sprintf(`^%s\[sdk=.*\]$`, regexp.QuoteMeta(key)))
	for existingKey := range buildSettings {
		if sdkVariant.MatchString(existingKey) {
			buildSettings[existingKey] = value
		}
	}
}

type objectChange struct {
	start, end int
	content    []byte
}

// perObjectModify splices re-serialised changed objects into the original bytes.
// The original and annotated trees are decoded from originalContents here rather than kept from
// parse time, so an opened project holds no extra copies of the tree; Save is called rarely.
func (p *XcodeProj) perObjectModify() ([]byte, error) {
	var annotated serialized.Object
	if _, err := plist.UnmarshalWithCustomAnnotation(p.originalContents, &annotated); err != nil {
		return nil, fmt.Errorf("failed to decode original project: %w", err)
	}

	original, _, err := decodePBXProj(p.originalContents)
	if err != nil {
		return nil, err
	}

	objectsModified, ok := p.rawProj.Object("objects")
	if !ok {
		return nil, fmt.Errorf("failed to parse project: no objects")
	}
	objectsOriginal, ok := original.Object("objects")
	if !ok {
		return nil, fmt.Errorf("failed to parse original project: no objects")
	}
	objectsAnnotated, ok := annotated.Object("objects")
	if !ok {
		return nil, fmt.Errorf("failed to parse annotated project: no objects")
	}

	var changes []objectChange
	for id := range objectsModified {
		objectModified, ok := objectsModified.Object(id)
		if !ok {
			return nil, fmt.Errorf("object %s is not a dictionary", id)
		}

		objectOriginal, ok := objectsOriginal.Object(id)
		if !ok {
			return nil, fmt.Errorf("new object added, not in original project: %s", id)
		}

		if reflect.DeepEqual(objectOriginal, objectModified) {
			continue
		}

		objectAnnotated, ok := objectsAnnotated.Object(id)
		if !ok {
			return nil, fmt.Errorf("new object added, not in original annotated project: %s", id)
		}

		position, ok := objectAnnotated.Object(customAnnotationKey)
		if !ok {
			return nil, fmt.Errorf("no raw object position available for %s", id)
		}
		start, ok := position.Int64(startKey)
		if !ok {
			return nil, fmt.Errorf("no raw object start position available for %s", id)
		}
		end, ok := position.Int64(endKey)
		if !ok {
			return nil, fmt.Errorf("no raw object end position available for %s", id)
		}

		content, err := plist.MarshalIndent(objectModified, p.format, "\t")
		if err != nil {
			return nil, fmt.Errorf("could not marshal object %s: %w", id, err)
		}

		changes = append(changes, objectChange{start: int(start), end: int(end), content: content})
	}

	if len(changes) == 0 {
		return p.originalContents, nil
	}

	sort.Slice(changes, func(i, j int) bool {
		if changes[i].start == changes[j].start {
			return changes[i].end < changes[j].end
		}
		return changes[i].start < changes[j].start
	})

	var result []byte
	previousEnd := 0
	for i, change := range changes {
		if i < len(changes)-1 && change.end >= changes[i+1].start {
			return nil, fmt.Errorf("overlapping changes: %d, %d", change.end, changes[i+1].start)
		}

		result = append(result, p.originalContents[previousEnd:change.start]...)
		result = append(result, change.content...)
		previousEnd = change.end
	}

	if previousEnd <= len(p.originalContents)-1 {
		result = append(result, p.originalContents[previousEnd:]...)
	}

	return result, nil
}
