// Package xcodeproj reads and edits Xcode project bundles (.xcodeproj).
package xcodeproj

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-plist"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-io/go-xcode/v2/plistutil"
	"github.com/bitrise-io/go-xcode/v2/xcodeproject/serialized"
	"github.com/bitrise-io/go-xcode/v2/xcodeproject/xcscheme"
)

// XcodeProjExtension is the file extension of an Xcode project bundle.
const XcodeProjExtension = ".xcodeproj"

const pbxProjFileName = "project.pbxproj"

const (
	customAnnotationKey = plist.CustomAnnotationKey
	startKey            = plist.CustomAnnotationStartKey
	endKey              = plist.CustomAnnotationEndKey
)

// IsXcodeProj reports whether the path has the .xcodeproj extension.
func IsXcodeProj(pth string) bool {
	return filepath.Ext(pth) == XcodeProjExtension
}

// Factory opens Xcode projects with a fixed set of dependencies.
type Factory struct {
	logger           log.Logger
	buildSettings    BuildSettingsProvider
	fileManager      fileutil.FileManager
	plistFileHandler plistutil.FileHandler
	pathModifier     pathutil.PathModifier
	pathProvider     pathutil.PathProvider
	userProvider     UserProvider
	dirCreator       DirCreator
	schemeFactory    xcscheme.Factory
}

// NewFactory returns a Factory that injects the given collaborators into every project it opens.
func NewFactory(
	logger log.Logger,
	buildSettings BuildSettingsProvider,
	fileManager fileutil.FileManager,
	plistFileHandler plistutil.FileHandler,
	pathModifier pathutil.PathModifier,
	pathProvider pathutil.PathProvider,
	userProvider UserProvider,
	dirCreator DirCreator,
) Factory {
	return Factory{
		logger:           logger,
		buildSettings:    buildSettings,
		fileManager:      fileManager,
		plistFileHandler: plistFileHandler,
		pathModifier:     pathModifier,
		pathProvider:     pathProvider,
		userProvider:     userProvider,
		dirCreator:       dirCreator,
		schemeFactory:    xcscheme.NewFactory(fileManager),
	}
}

// Open reads and parses the project at pth.
func (f Factory) Open(pth string) (*XcodeProj, error) {
	absPth, err := f.pathModifier.AbsPath(pth)
	if err != nil {
		return nil, fmt.Errorf("failed to expand path (%s): %w", pth, err)
	}

	pbxProjPth := filepath.Join(absPth, pbxProjFileName)

	file, err := f.fileManager.Open(pbxProjPth)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", pbxProjPth, err)
	}
	defer func() {
		_ = file.Close()
	}()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", pbxProjPth, err)
	}

	// The buffer was read here and nobody else holds it, so it needs no copy.
	return f.parse(content, absPth)
}

// Parse builds a project from project.pbxproj content. projectPath is where the project is
// considered to live: relative build setting paths resolve against it and Save writes to it.
// content is copied, so the caller may reuse it.
func (f Factory) Parse(content []byte, projectPath string) (*XcodeProj, error) {
	// Save rebuilds the file from these bytes, so a buffer the caller changes later would corrupt it.
	return f.parse(bytes.Clone(content), projectPath)
}

func (f Factory) parse(content []byte, projectPath string) (*XcodeProj, error) {
	p, err := parsePBXProj(content)
	if err != nil {
		return nil, err
	}

	p.logger = f.logger
	p.buildSettings = f.buildSettings
	p.fileManager = f.fileManager
	p.plistFileHandler = f.plistFileHandler
	p.pathModifier = f.pathModifier
	p.pathProvider = f.pathProvider
	p.userProvider = f.userProvider
	p.dirCreator = f.dirCreator
	p.schemeFactory = f.schemeFactory

	p.Path = projectPath
	p.Name = strings.TrimSuffix(filepath.Base(projectPath), filepath.Ext(projectPath))

	return p, nil
}

// XcodeProj is a parsed Xcode project.
type XcodeProj struct {
	Name string
	Path string

	logger           log.Logger
	buildSettings    BuildSettingsProvider
	fileManager      fileutil.FileManager
	plistFileHandler plistutil.FileHandler
	pathModifier     pathutil.PathModifier
	pathProvider     pathutil.PathProvider
	userProvider     UserProvider
	dirCreator       DirCreator
	schemeFactory    xcscheme.Factory

	// rawProj is the source of truth Save writes out; every edit lands here.
	rawProj serialized.Object
	format  int
	// originalContents is what Save diffs against to rewrite only the changed objects.
	originalContents []byte

	projectID                string
	targets                  []Target
	buildConfigurations      []BuildConfiguration
	defaultConfigurationName string
	targetAttributes         serialized.Object
}

// Targets returns the project's targets, in project file order.
func (p *XcodeProj) Targets() []Target {
	return p.targets
}

// Target returns the target with the given ID.
func (p *XcodeProj) Target(id string) (Target, bool) {
	for _, target := range p.targets {
		if target.ID == id {
			return target, true
		}
	}
	return Target{}, false
}

// TargetByName returns the target with the given name.
func (p *XcodeProj) TargetByName(name string) (Target, bool) {
	for _, target := range p.targets {
		if target.Name == name {
			return target, true
		}
	}
	return Target{}, false
}

// BuildConfigurations returns the project-level build configurations.
func (p *XcodeProj) BuildConfigurations() []BuildConfiguration {
	return p.buildConfigurations
}

// DefaultConfigurationName returns the name of the project's default build configuration.
func (p *XcodeProj) DefaultConfigurationName() string {
	return p.defaultConfigurationName
}

// DependentTargetsOfTarget returns the direct and transitive dependencies of target, each once.
// Dependencies that cannot be resolved are logged and skipped.
func (p *XcodeProj) DependentTargetsOfTarget(target Target) []Target {
	// visited guards against dependency cycles in malformed project files.
	visited := map[string]bool{target.ID: true}
	return deduplicateTargets(p.dependentTargetsOfTarget(target, visited))
}

func (p *XcodeProj) dependentTargetsOfTarget(target Target, visited map[string]bool) []Target {
	var dependents []Target

	for _, dependencyTargetID := range target.dependencyTargetIDs {
		child, ok := p.Target(dependencyTargetID)
		if !ok {
			p.logger.Warnf("couldn't find dependency %s of target %s (%s), skipping", dependencyTargetID, target.Name, target.ID)
			continue
		}

		if visited[child.ID] {
			continue
		}
		visited[child.ID] = true

		dependents = append(dependents, child)
		dependents = append(dependents, p.dependentTargetsOfTarget(child, visited)...)
	}

	return dependents
}

// TargetDevelopmentTeam returns the DevelopmentTeam recorded in TargetAttributes for the target.
func (p *XcodeProj) TargetDevelopmentTeam(targetID string) (string, bool) {
	if p.targetAttributes == nil {
		return "", false
	}

	attributes, ok := p.targetAttributes.Object(targetID)
	if !ok {
		return "", false
	}

	developmentTeam, ok := attributes.String("DevelopmentTeam")
	if !ok || developmentTeam == "" {
		return "", false
	}

	return developmentTeam, true
}

func deduplicateTargets(targets []Target) []Target {
	seen := map[string]bool{}
	unique := make([]Target, 0, len(targets))

	for _, target := range targets {
		if seen[target.ID] {
			continue
		}
		seen[target.ID] = true
		unique = append(unique, target)
	}

	return unique
}
