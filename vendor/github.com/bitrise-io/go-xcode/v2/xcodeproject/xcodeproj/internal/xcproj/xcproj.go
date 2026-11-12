// Package xcproj reads and edits project.xcproj, the JSON project file Xcode 27 writes inside
// an .xcodeproj bundle in place of project.pbxproj.
package xcproj

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tailscale/hujson"

	"github.com/bitrise-io/go-xcode/v2/xcodeproject/serialized"
)

// The target kinds. A target written without a kind is native.
const (
	TargetKindNative              = "native"
	TargetKindAggregate           = "aggregate"
	TargetKindExternalBuildSystem = "external-build-system"
)

// A reference written without a kind is a file reference.
const (
	ReferenceKindFile   = "file-reference"
	ReferenceKindGroup  = "group"
	ReferenceKindFolder = "folder"
)

// BuildPhaseKindResources names the Copy Resources phase in a build phase reference.
const BuildPhaseKindResources = "resources"

const (
	localTargetDependencyKind = "localTarget"

	// productTypePrefix is left out of a product type written as product-type. A product type
	// without it is written whole as full-product-type.
	productTypePrefix = "com.apple.product-type."

	groupTreeIDPrefix = "id:"
)

// Project is the part of project.xcproj that go-xcode reads.
type Project struct {
	DefaultConfiguration string
	Configurations       []Configuration

	// BuildSettings holds every configuration in one map: a key carries its conditions, as in
	// CODE_SIGN_STYLE[config=Release].
	BuildSettings serialized.Object

	Targets []Target

	Files []Reference
}

// Reference is a file reference, a group or a folder of the file tree.
type Reference struct {
	ID   string
	Kind string

	// Path is relative to the parent group, unless it starts with / or ~, or with a base such as
	// <PROJECT>/. A group without one takes its parent's directory.
	Path string

	// Targets is set only for a folder.
	Targets []string

	// BuildPhases leaves out a build phase referenced by object ID: Xcode writes one only when the
	// name would be ambiguous, and build phases are not decoded.
	BuildPhases []BuildPhaseReference

	Children []Reference
}

// BuildPhaseReference is the target and kind part of a build phase name path.
type BuildPhaseReference struct {
	Target string
	Kind   string
}

// Configuration is one build configuration of the project.
type Configuration struct {
	// ID is usually empty: Xcode writes one only when the name alone would be ambiguous.
	ID   string
	Name string
}

// Target is one target of the project.
type Target struct {
	ID   string
	Name string
	Kind string

	// ProductType is the full pbxproj product type, such as com.apple.product-type.application.
	ProductType string

	// Product is the product's name path in the group tree, such as [Products App.app].
	Product []string
	// ProductID is set in place of Product when the name path would be ambiguous.
	ProductID string

	// TestHostTarget is the name of the app target a test target runs in.
	TestHostTarget string

	// Dependencies are the names of the project's own targets this target depends on.
	// Dependencies on other projects and on Swift packages are left out.
	Dependencies []string

	// BuildSettings is keyed like Project.BuildSettings.
	BuildSettings serialized.Object
}

type rawProject struct {
	DefaultConfiguration string             `json:"default-configuration"`
	Configurations       []rawConfiguration `json:"configurations"`
	BuildSettings        serialized.Object  `json:"build-settings"`
	Targets              []rawTarget        `json:"targets"`
	Files                []rawReference     `json:"files"`
}

// rawConfiguration is written as a bare name unless it has an ID or an xcconfig file.
type rawConfiguration struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type rawTarget struct {
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	Kind            string                `json:"kind"`
	ProductType     string                `json:"product-type"`
	FullProductType string                `json:"full-product-type"`
	Product         rawGroupTreeReference `json:"product"`
	TestHostTarget  string                `json:"test-host-target"`
	Dependencies    []rawDependency       `json:"dependencies"`
	BuildSettings   serialized.Object     `json:"build-settings"`
}

// rawReference holds target-membership raw, because a folder lists target names and a file
// reference lists build files.
type rawReference struct {
	ID               string            `json:"id"`
	Kind             string            `json:"kind"`
	Path             string            `json:"path"`
	TargetMembership []json.RawMessage `json:"target-membership"`
	Children         []rawReference    `json:"children"`
}

// rawBuildFile is written as a bare build phase reference unless it carries build file
// properties.
type rawBuildFile struct {
	BuildPhase rawGroupTreeReference `json:"build-phase"`
}

// rawGroupTreeReference is written as an id:-prefixed object ID, a /-joined name path, or, when
// a name contains /, an array of names.
type rawGroupTreeReference struct {
	ID       string
	NamePath []string
}

// rawNamePathComponent is written as an object only when the name contains /.
type rawNamePathComponent struct {
	Name string `json:"name"`
}

// rawDependency is written as a bare target name unless it carries platform filters or points
// outside the project.
type rawDependency struct {
	Kind   string `json:"kind"`
	Target string `json:"target"`
}

// Decode parses the content of a project.xcproj file.
func Decode(content []byte) (Project, error) {
	standard, err := hujson.Standardize(bytes.Clone(content))
	if err != nil {
		return Project{}, fmt.Errorf("failed to parse project.xcproj (Xcode writes JSON with comments and trailing commas; other JSON5 syntax, such as unquoted keys or single-quoted strings, is not supported): %w", err)
	}

	var raw rawProject
	if err := json.Unmarshal(standard, &raw); err != nil {
		return Project{}, fmt.Errorf("failed to decode project.xcproj: %w", err)
	}

	return raw.project()
}

// UnmarshalJSON accepts a bare name or an object.
func (c *rawConfiguration) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &c.Name); err == nil {
		return nil
	}

	type configurationObject rawConfiguration
	return json.Unmarshal(data, (*configurationObject)(c))
}

// UnmarshalJSON accepts a bare target name or an object.
func (d *rawDependency) UnmarshalJSON(data []byte) error {
	d.Kind = localTargetDependencyKind
	if err := json.Unmarshal(data, &d.Target); err == nil {
		return nil
	}

	type dependencyObject rawDependency
	return json.Unmarshal(data, (*dependencyObject)(d))
}

// UnmarshalJSON accepts a string or an array of name path components.
func (r *rawGroupTreeReference) UnmarshalJSON(data []byte) error {
	var reference string
	if err := json.Unmarshal(data, &reference); err == nil {
		if id, ok := strings.CutPrefix(reference, groupTreeIDPrefix); ok {
			r.ID = id
		} else if reference != "" {
			r.NamePath = strings.Split(reference, "/")
		}
		return nil
	}

	var components []rawNamePathComponent
	if err := json.Unmarshal(data, &components); err != nil {
		return err
	}
	r.NamePath = make([]string, 0, len(components))
	for _, component := range components {
		r.NamePath = append(r.NamePath, component.Name)
	}
	return nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (b *rawBuildFile) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &b.BuildPhase); err == nil {
		return nil
	}

	type buildFileObject rawBuildFile
	return json.Unmarshal(data, (*buildFileObject)(b))
}

// UnmarshalJSON accepts a bare name or an object.
func (c *rawNamePathComponent) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &c.Name); err == nil {
		return nil
	}

	type componentObject rawNamePathComponent
	return json.Unmarshal(data, (*componentObject)(c))
}

func (r rawProject) project() (Project, error) {
	configurations := make([]Configuration, 0, len(r.Configurations))
	for i, rawConfig := range r.Configurations {
		if rawConfig.Name == "" {
			return Project{}, fmt.Errorf("configuration %d has no name", i)
		}
		configurations = append(configurations, Configuration(rawConfig))
	}

	targets := make([]Target, 0, len(r.Targets))
	names := make(map[string]bool, len(r.Targets))
	for i, rawTarget := range r.Targets {
		target, err := rawTarget.target()
		if err != nil {
			return Project{}, fmt.Errorf("target %d: %w", i, err)
		}
		if names[target.Name] {
			return Project{}, fmt.Errorf("target %d: another target is also named %s", i, target.Name)
		}
		names[target.Name] = true
		targets = append(targets, target)
	}
	for _, target := range targets {
		for _, dependency := range target.Dependencies {
			if !names[dependency] {
				return Project{}, fmt.Errorf("target %s depends on unknown target %s", target.Name, dependency)
			}
		}
	}

	files, err := references(r.Files)
	if err != nil {
		return Project{}, err
	}

	return Project{
		DefaultConfiguration: r.DefaultConfiguration,
		Configurations:       configurations,
		BuildSettings:        nonNil(r.BuildSettings),
		Targets:              targets,
		Files:                files,
	}, nil
}

func (r rawTarget) target() (Target, error) {
	if r.Name == "" {
		return Target{}, fmt.Errorf("target has no name")
	}
	if r.ID == "" {
		return Target{}, fmt.Errorf("target %s has no id", r.Name)
	}

	kind := r.Kind
	switch kind {
	case "":
		kind = TargetKindNative
	case TargetKindNative, TargetKindAggregate, TargetKindExternalBuildSystem:
	default:
		return Target{}, fmt.Errorf("target %s has unknown kind: %s", r.Name, kind)
	}

	productType := r.FullProductType
	if r.ProductType != "" {
		productType = productTypePrefix + r.ProductType
	}

	var dependencies []string
	for _, dependency := range r.Dependencies {
		if dependency.Kind != localTargetDependencyKind {
			continue
		}
		if dependency.Target == "" {
			return Target{}, fmt.Errorf("target %s has a dependency with no target", r.Name)
		}
		dependencies = append(dependencies, dependency.Target)
	}

	return Target{
		ID:             r.ID,
		Name:           r.Name,
		Kind:           kind,
		ProductType:    productType,
		Product:        r.Product.NamePath,
		ProductID:      r.Product.ID,
		TestHostTarget: r.TestHostTarget,
		Dependencies:   dependencies,
		BuildSettings:  nonNil(r.BuildSettings),
	}, nil
}

func references(rawReferences []rawReference) ([]Reference, error) {
	if rawReferences == nil {
		return nil, nil
	}

	references := make([]Reference, 0, len(rawReferences))
	for _, rawReference := range rawReferences {
		reference, err := rawReference.reference()
		if err != nil {
			return nil, err
		}
		references = append(references, reference)
	}
	return references, nil
}

func (r rawReference) reference() (Reference, error) {
	reference := Reference{ID: r.ID, Kind: cmp.Or(r.Kind, ReferenceKindFile), Path: r.Path}

	for _, membership := range r.TargetMembership {
		if reference.Kind == ReferenceKindFolder {
			var target string
			if err := json.Unmarshal(membership, &target); err != nil {
				return Reference{}, fmt.Errorf("folder %s has an invalid target membership: %w", r.Path, err)
			}
			reference.Targets = append(reference.Targets, target)
			continue
		}

		var buildFile rawBuildFile
		if err := json.Unmarshal(membership, &buildFile); err != nil {
			return Reference{}, fmt.Errorf("file %s has an invalid target membership: %w", r.Path, err)
		}
		if buildFile.BuildPhase.ID != "" {
			continue
		}
		namePath := buildFile.BuildPhase.NamePath
		// The name path is the target, the build phase kind and, when the kind is not unique, the
		// build phase name.
		if len(namePath) < 2 || len(namePath) > 3 {
			return Reference{}, fmt.Errorf("file %s has an invalid build phase reference: %s", r.Path, membership)
		}
		reference.BuildPhases = append(reference.BuildPhases, BuildPhaseReference{Target: namePath[0], Kind: namePath[1]})
	}

	children, err := references(r.Children)
	if err != nil {
		return Reference{}, err
	}
	reference.Children = children

	return reference, nil
}

// nonNil keeps an absent settings map from reading differently than an empty one.
func nonNil(settings serialized.Object) serialized.Object {
	if settings == nil {
		return serialized.Object{}
	}
	return settings
}
