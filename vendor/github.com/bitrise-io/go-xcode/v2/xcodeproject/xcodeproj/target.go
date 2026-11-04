package xcodeproj

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/bitrise-io/go-xcode/v2/xcodeproject/serialized"
)

const (
	nativeTargetISA    = "PBXNativeTarget"
	aggregateTargetISA = "PBXAggregateTarget"
	legacyTargetISA    = "PBXLegacyTarget"
)

// targetKind is what builds a target. The zero value is no kind.
type targetKind int

const (
	nativeTargetKind targetKind = iota + 1
	aggregateTargetKind
	externalBuildSystemTargetKind
)

const appClipProductType = "com.apple.product-type.application.on-demand-install-capable"

// Target is a build target of the project.
type Target struct {
	ID   string
	Name string

	BuildConfigurations      []BuildConfiguration
	DefaultConfigurationName string

	kind        targetKind
	productType string
	// productPath is the path of the product file; only its base name, such as App.app, is used.
	productPath         string
	dependencyTargetIDs []string
}

// DependsOn reports whether the target directly depends on the target with the given ID.
func (t Target) DependsOn(targetID string) bool {
	return slices.Contains(t.dependencyTargetIDs, targetID)
}

// IsExecutableProduct reports whether the target produces an .app or .appex bundle.
func (t Target) IsExecutableProduct() bool {
	return t.isAppProduct() || t.isAppExtensionProduct()
}

// IsUITestProduct reports whether the target is a UI test bundle.
func (t Target) IsUITestProduct() bool {
	return filepath.Ext(t.productType) == ".ui-testing"
}

// IsAppClipProduct reports whether the target produces an App Clip.
func (t Target) IsAppClipProduct() bool {
	return t.productType == appClipProductType
}

func (t Target) isAppProduct() bool {
	return filepath.Ext(t.productPath) == ".app"
}

func (t Target) isAppExtensionProduct() bool {
	return filepath.Ext(t.productPath) == ".appex"
}

func (t Target) isNativeTarget() bool {
	return t.kind == nativeTargetKind
}

// Based on https://github.com/CocoaPods/Xcodeproj/blob/907c81763a7660978fda93b2f38f05de0cbb51ad/lib/xcodeproj/project/object/native_target.rb#L470
func (t Target) isTest() bool {
	return t.isTestProduct() ||
		t.IsUITestProduct() ||
		t.productType == "com.apple.product-type.bundle" // OCTest bundle
}

func (t Target) isTestProduct() bool {
	return filepath.Ext(t.productType) == ".unit-test"
}

func parseTarget(id string, objects serialized.Object) (Target, error) {
	rawTarget, ok := objects.Object(id)
	if !ok {
		return Target{}, fmt.Errorf("target %s not found", id)
	}

	isa, ok := rawTarget.String("isa")
	if !ok {
		return Target{}, fmt.Errorf("target %s has no isa", id)
	}

	var kind targetKind
	switch isa {
	case nativeTargetISA:
		kind = nativeTargetKind
	case aggregateTargetISA:
		kind = aggregateTargetKind
	case legacyTargetISA:
		kind = externalBuildSystemTargetKind
	default:
		return Target{}, fmt.Errorf("unknown target type: %s", isa)
	}

	name, ok := rawTarget.String("name")
	if !ok {
		return Target{}, fmt.Errorf("target %s has no name", id)
	}

	productType, _ := rawTarget.String("productType")

	buildConfigurationListID, ok := rawTarget.String("buildConfigurationList")
	if !ok {
		return Target{}, fmt.Errorf("target %s has no buildConfigurationList", id)
	}

	buildConfigurations, defaultConfigurationName, err := parseConfigurationList(buildConfigurationListID, objects)
	if err != nil {
		return Target{}, fmt.Errorf("target %s: %w", id, err)
	}

	dependencyIDs, _ := rawTarget.StringSlice("dependencies")

	var dependencyTargetIDs []string
	for _, dependencyID := range dependencyIDs {
		// Only direct target dependencies carry a "target" key.
		targetID, ok := targetIDOfDependency(dependencyID, objects)
		if !ok {
			continue
		}
		dependencyTargetIDs = append(dependencyTargetIDs, targetID)
	}

	var productPath string
	if productReferenceID, ok := rawTarget.String("productReference"); ok {
		productReference, ok := objects.Object(productReferenceID)
		if !ok {
			return Target{}, fmt.Errorf("target %s references missing product %s", id, productReferenceID)
		}
		productPath, ok = productReference.String("path")
		if !ok {
			return Target{}, fmt.Errorf("target %s product %s has no path", id, productReferenceID)
		}
	}

	return Target{
		ID:                       id,
		Name:                     name,
		BuildConfigurations:      buildConfigurations,
		DefaultConfigurationName: defaultConfigurationName,
		kind:                     kind,
		productType:              productType,
		productPath:              productPath,
		dependencyTargetIDs:      dependencyTargetIDs,
	}, nil
}

func targetIDOfDependency(dependencyID string, objects serialized.Object) (string, bool) {
	rawDependency, ok := objects.Object(dependencyID)
	if !ok {
		return "", false
	}
	return rawDependency.String("target")
}
