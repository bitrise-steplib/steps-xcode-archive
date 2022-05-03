package xcodeproj

import "github.com/bitrise-io/go-xcode/xcodeproject/serialized"

// TargetDependency is a reference to another Target that is a dependency of a given Target
type TargetDependency struct {
	ID       string
	TargetID string
}

func parseTargetDependency(id string, objects serialized.Object) (TargetDependency, error) {
	rawTargetDependency, err := objects.Object(id)
	if err != nil {
		return TargetDependency{}, err
	}

	targetID, err := rawTargetDependency.String("target")
	if err != nil {
		return TargetDependency{}, err
	}

	return TargetDependency{
		ID:       id,
		TargetID: targetID,
	}, nil
}
