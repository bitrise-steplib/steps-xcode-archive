package xcodeproj

import (
	"fmt"

	"github.com/bitrise-io/go-plist"
	"github.com/bitrise-io/go-xcode/v2/xcodeproject/serialized"
)

const pbxProjectISA = "PBXProject"

func parsePBXProj(content []byte, base baseProj) (*PBXProj, error) {
	rawProj, format, err := decodePBXProj(content)
	if err != nil {
		return nil, err
	}

	objects, ok := rawProj.Object("objects")
	if !ok {
		return nil, fmt.Errorf("failed to parse project.pbxproj: no objects")
	}

	projectID, rawProject, err := findProject(objects)
	if err != nil {
		return nil, err
	}

	buildConfigurationListID, ok := rawProject.String("buildConfigurationList")
	if !ok {
		return nil, fmt.Errorf("project %s has no buildConfigurationList", projectID)
	}

	buildConfigurations, defaultConfigurationName, err := parseConfigurationList(buildConfigurationListID, objects)
	if err != nil {
		return nil, fmt.Errorf("failed to parse project build configuration list: %w", err)
	}

	targets, buildPhaseIDs, err := parseTargets(rawProject, objects)
	if err != nil {
		return nil, err
	}

	var targetAttributes serialized.Object
	if attributes, ok := rawProject.Object("attributes"); ok {
		targetAttributes, _ = attributes.Object("TargetAttributes")
	}

	base.targets = targets
	base.buildConfigurations = buildConfigurations
	base.defaultConfigurationName = defaultConfigurationName

	return &PBXProj{
		baseProj:         base,
		rawProj:          rawProj,
		format:           format,
		originalContents: content,
		projectID:        projectID,
		targetAttributes: targetAttributes,
		buildPhaseIDs:    buildPhaseIDs,
	}, nil
}

func decodePBXProj(content []byte) (serialized.Object, int, error) {
	var rawProj serialized.Object

	format, err := plist.UnmarshalWithCustomAnnotation(content, &rawProj)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal project.pbxproj: %w", err)
	}

	return removeCustomInfoObject(rawProj), format, nil
}

func findProject(objects serialized.Object) (string, serialized.Object, error) {
	for id := range objects {
		object, ok := objects.Object(id)
		if !ok {
			continue
		}

		isa, ok := object.String("isa")
		if !ok {
			continue
		}

		if isa == pbxProjectISA {
			return id, object, nil
		}
	}

	return "", nil, fmt.Errorf("failed to find PBXProject in project.pbxproj")
}

// parseTargets returns the project's targets and the build phase IDs of each, by target ID.
func parseTargets(rawProject, objects serialized.Object) ([]Target, map[string][]string, error) {
	targetIDs, ok := rawProject.StringSlice("targets")
	if !ok {
		return nil, nil, fmt.Errorf("project has no targets")
	}

	var targets []Target
	buildPhaseIDs := map[string][]string{}
	for _, targetID := range targetIDs {
		rawTarget, ok := objects.Object(targetID)
		if !ok {
			// The targets list can reference targets that were removed.
			continue
		}

		target, err := parseTarget(targetID, objects)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse target %s: %w", targetID, err)
		}

		targets = append(targets, target)
		if ids, ok := rawTarget.StringSlice("buildPhases"); ok {
			buildPhaseIDs[targetID] = ids
		}
	}

	return targets, buildPhaseIDs, nil
}

// removeCustomInfoObject strips the byte positions UnmarshalWithCustomAnnotation stores under
// customAnnotationKey in every dictionary, so they don't show up as keys or get written back.
// The positions exist for save only because the file on disk should change only where we changed it,
// so perObjectModify decodes them again and replaces only the changed objects.
func removeCustomInfoObject(o serialized.Object) serialized.Object {
	for _, v := range o {
		removeCustomInfo(v)
	}
	return o
}

func removeCustomInfo(o any) any {
	switch container := o.(type) {
	case map[string]any:
		delete(container, customAnnotationKey)
		for _, value := range container {
			removeCustomInfo(value)
		}
		return container
	case []any:
		for _, element := range container {
			removeCustomInfo(element)
		}
		return container
	default:
		return o
	}
}
