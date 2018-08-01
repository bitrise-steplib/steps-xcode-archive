package xcodeproj

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/pathutil"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-tools/xcode-project/serialized"
	"github.com/bitrise-tools/xcode-project/xcscheme"
	"howett.net/plist"
)

// XcodeProj ...
type XcodeProj struct {
	Proj Proj

	Name string
	Path string
}

// TargetCodeSignEntitlementsPath ...
func (p XcodeProj) TargetCodeSignEntitlementsPath(target, configuration string) (string, error) {
	buildSettings, err := p.TargetBuildSettings(target, configuration, "")
	if err != nil {
		return "", err
	}

	relPth, err := buildSettings.String("CODE_SIGN_ENTITLEMENTS")
	if err != nil {
		return "", err
	}

	return filepath.Join(filepath.Dir(p.Path), relPth), nil
}

// TargetCodeSignEntitlements ...
func (p XcodeProj) TargetCodeSignEntitlements(target, configuration string) (serialized.Object, error) {
	codeSignEntitlementsPth, err := p.TargetCodeSignEntitlementsPath(target, configuration)
	if err != nil {
		return nil, err
	}

	codeSignEntitlementsContent, err := fileutil.ReadBytesFromFile(codeSignEntitlementsPth)
	if err != nil {
		return nil, err
	}

	var codeSignEntitlements serialized.Object
	if _, err := plist.Unmarshal([]byte(codeSignEntitlementsContent), &codeSignEntitlements); err != nil {
		return nil, err
	}

	return codeSignEntitlements, nil
}

// TargetInformationPropertyListPath ...
func (p XcodeProj) TargetInformationPropertyListPath(target, configuration string) (string, error) {
	buildSettings, err := p.TargetBuildSettings(target, configuration, "")
	if err != nil {
		return "", err
	}

	relPth, err := buildSettings.String("INFOPLIST_FILE")
	if err != nil {
		return "", err
	}

	return filepath.Join(filepath.Dir(p.Path), relPth), nil
}

// TargetInformationPropertyList ...
func (p XcodeProj) TargetInformationPropertyList(target, configuration string) (serialized.Object, error) {
	informationPropertyListPth, err := p.TargetInformationPropertyListPath(target, configuration)
	if err != nil {
		return nil, err
	}

	informationPropertyListContent, err := fileutil.ReadBytesFromFile(informationPropertyListPth)
	if err != nil {
		return nil, err
	}

	var informationPropertyList serialized.Object
	if _, err := plist.Unmarshal([]byte(informationPropertyListContent), &informationPropertyList); err != nil {
		return nil, err
	}

	return informationPropertyList, nil
}

// TargetBundleID ...
func (p XcodeProj) TargetBundleID(target, configuration string) (string, error) {
	buildSettings, err := p.TargetBuildSettings(target, configuration, "")
	if err != nil {
		return "", err
	}

	bundleID, err := buildSettings.String("PRODUCT_BUNDLE_IDENTIFIER")
	if err != nil && !serialized.IsKeyNotFoundError(err) {
		return "", err
	}

	if bundleID != "" {
		return bundleID, nil
	}

	informationPropertyList, err := p.TargetInformationPropertyList(target, configuration)
	if err != nil {
		return "", err
	}

	bundleID, err = informationPropertyList.String("CFBundleIdentifier")
	if err != nil {
		return "", err
	}

	if bundleID == "" {
		return "", errors.New("no PRODUCT_BUNDLE_IDENTIFIER build settings nor CFBundleIdentifier information property found")
	}

	return bundleID, nil
}

// TargetBuildSettings ...
func (p XcodeProj) TargetBuildSettings(target, configuration, sdk string) (serialized.Object, error) {
	return showBuildSettings(p.Path, target, configuration, sdk)
}

// Scheme ...
func (p XcodeProj) Scheme(name string) (xcscheme.Scheme, bool) {
	schemes, err := p.Schemes()
	if err != nil {
		return xcscheme.Scheme{}, false
	}

	for _, scheme := range schemes {
		if scheme.Name == name {
			return scheme, true
		}
	}

	return xcscheme.Scheme{}, false
}

// Schemes ...
func (p XcodeProj) Schemes() ([]xcscheme.Scheme, error) {
	pattern := filepath.Join(p.Path, "xcshareddata", "xcschemes", "*.xcscheme")
	pths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	var schemes []xcscheme.Scheme
	for _, pth := range pths {
		scheme, err := xcscheme.Open(pth)
		if err != nil {
			return nil, err
		}
		schemes = append(schemes, scheme)
	}

	return schemes, nil
}

// Open ...
func Open(pth string) (XcodeProj, error) {
	absPth, err := pathutil.AbsPath(pth)
	if err != nil {
		return XcodeProj{}, err
	}

	pbxProjPth := filepath.Join(absPth, "project.pbxproj")

	b, err := fileutil.ReadBytesFromFile(pbxProjPth)
	if err != nil {
		return XcodeProj{}, err
	}

	var raw serialized.Object
	if _, err := plist.Unmarshal(b, &raw); err != nil {
		return XcodeProj{}, fmt.Errorf("failed to generate json from Pbxproj - error: %s", err)
	}

	objects, err := raw.Object("objects")
	if err != nil {
		return XcodeProj{}, err
	}

	projectID := ""
	for id := range objects {
		object, err := objects.Object(id)
		if err != nil {
			return XcodeProj{}, err
		}

		objectISA, err := object.String("isa")
		if err != nil {
			return XcodeProj{}, err
		}

		if objectISA == "PBXProject" {
			projectID = id
			break
		}
	}

	p, err := parseProj(projectID, objects)
	if err != nil {
		return XcodeProj{}, nil
	}

	return XcodeProj{
		Proj: p,
		Path: absPth,
		Name: strings.TrimSuffix(filepath.Base(absPth), filepath.Ext(absPth)),
	}, nil
}

// IsXcodeProj ...
func IsXcodeProj(pth string) bool {
	return filepath.Ext(pth) == ".xcodeproj"
}
