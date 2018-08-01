package xcodeproj

import (
	"fmt"

	"github.com/bitrise-tools/xcode-project/serialized"
)

// TargetType ...
type TargetType string

// TargetTypes
const (
	NativeTargetType    TargetType = "PBXNativeTarget"
	AggregateTargetType TargetType = "PBXAggregateTarget"
	LegacyTargetType    TargetType = "PBXLegacyTarget"
)

// Target ...
type Target struct {
	Type                   TargetType
	ID                     string
	Name                   string
	BuildConfigurationList ConfigurationList
	Dependencies           []TargetDependency
}

// DependentTargets ...
func (t Target) DependentTargets() []Target {
	var targets []Target
	for _, targetDependency := range t.Dependencies {
		childTarget := targetDependency.Target
		targets = append(targets, childTarget)

		childDependentTargets := childTarget.DependentTargets()
		targets = append(targets, childDependentTargets...)
	}

	return targets
}

func parseTarget(id string, objects serialized.Object) (Target, error) {
	rawTarget, err := objects.Object(id)
	if err != nil {
		return Target{}, err
	}

	isa, err := rawTarget.String("isa")
	if err != nil {
		return Target{}, err
	}

	var targetType TargetType
	switch isa {
	case "PBXNativeTarget":
		targetType = NativeTargetType
	case "PBXAggregateTarget":
		targetType = AggregateTargetType
	case "PBXLegacyTarget":
		targetType = LegacyTargetType
	default:
		return Target{}, fmt.Errorf("unknown target type: %s", isa)
	}

	name, err := rawTarget.String("name")
	if err != nil {
		return Target{}, err
	}

	buildConfigurationListID, err := rawTarget.String("buildConfigurationList")
	if err != nil {
		return Target{}, err
	}

	buildConfigurationList, err := parseConfigurationList(buildConfigurationListID, objects)
	if err != nil {
		return Target{}, err
	}

	dependencyIDs, err := rawTarget.StringSlice("dependencies")
	if err != nil {
		return Target{}, err
	}

	var dependencies []TargetDependency
	for _, dependencyID := range dependencyIDs {
		dependency, err := parseTargetDependency(dependencyID, objects)
		if err != nil {
			return Target{}, err
		}

		dependencies = append(dependencies, dependency)
	}

	return Target{
		Type: targetType,
		ID:   id,
		Name: name,
		BuildConfigurationList: buildConfigurationList,
		Dependencies:           dependencies,
	}, nil
}
