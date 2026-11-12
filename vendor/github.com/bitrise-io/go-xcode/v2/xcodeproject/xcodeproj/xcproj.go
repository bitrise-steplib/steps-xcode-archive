package xcodeproj

import (
	"fmt"
	"maps"
	"path"
	"path/filepath"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/v2/internal/filewriter"
	"github.com/bitrise-io/go-xcode/v2/xcodeproject/serialized"
	"github.com/bitrise-io/go-xcode/v2/xcodeproject/xcodeproj/internal/xcproj"
)

var (
	xcprojTargetKinds = map[string]targetKind{
		xcproj.TargetKindNative:              nativeTargetKind,
		xcproj.TargetKindAggregate:           aggregateTargetKind,
		xcproj.TargetKindExternalBuildSystem: externalBuildSystemTargetKind,
	}
)

// XCProj is a project whose project file is project.xcproj, the JSON format.
type XCProj struct {
	baseProj

	content []byte
	editor  *xcproj.Editor
	files   []xcproj.Reference
}

func (f Factory) newXCProj(content []byte, projectPath string) (*XCProj, error) {
	return parseXCProj(content, f.newBaseProj(projectPath))
}

// ForceCodeSign implements XcodeProj. It leaves legacy-team-id and legacy-provisioning-style
// alone, as Xcode ignores them.
func (p *XCProj) ForceCodeSign(opts ForceCodeSignOptions) error {
	target, ok := p.TargetByName(opts.TargetName)
	if !ok {
		return fmt.Errorf("failed to find target with name: %s", opts.TargetName)
	}

	buildSettings, err := targetBuildSettings(target, opts.Configuration)
	if err != nil {
		return err
	}

	for key, value := range forcedCodeSignBuildSettings(opts) {
		for _, sdkKey := range keysForAllSDKs(buildSettings, key) {
			if err := p.SetBuildSetting(opts.TargetName, opts.Configuration, sdkKey, value); err != nil {
				return err
			}
		}
	}

	return nil
}

// SetBuildSetting implements XcodeProj. Only string values are supported.
//
// TODO: a key with a [config=...] condition is rejected here but accepted by
// PBXProj.SetBuildSetting; make the two formats agree.
func (p *XCProj) SetBuildSetting(targetName, configurationName, key string, value any) error {
	stringValue, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to set build setting %s: a %T value is not supported for project.xcproj, only a string", key, value)
	}

	target, ok := p.TargetByName(targetName)
	if !ok {
		return fmt.Errorf("failed to find target with name: %s", targetName)
	}

	buildSettings, err := targetBuildSettings(target, configurationName)
	if err != nil {
		return err
	}

	if p.editor == nil {
		configurationNames := make([]string, 0, len(p.buildConfigurations))
		for _, configuration := range p.buildConfigurations {
			configurationNames = append(configurationNames, configuration.Name)
		}

		editor, err := xcproj.NewEditor(p.content, configurationNames)
		if err != nil {
			return err
		}
		p.editor = editor
	}

	if err := p.editor.SetTargetBuildSetting(targetName, configurationName, key, stringValue); err != nil {
		return err
	}
	buildSettings[key] = stringValue

	return nil
}

// Save implements XcodeProj.
func (p *XCProj) Save() error {
	content := p.content
	if p.editor != nil {
		content = p.editor.Content()
	}

	pth := filepath.Join(p.path, xcProjFileName)
	if err := filewriter.WriteKeepingMode(p.fileManager, pth, content, newProjectFileMode); err != nil {
		return fmt.Errorf("failed to write %s: %w", pth, err)
	}

	return nil
}

func parseXCProj(content []byte, base baseProj) (*XCProj, error) {
	project, err := xcproj.Decode(content)
	if err != nil {
		return nil, err
	}

	base.targets = xcprojTargets(project, base.logger)
	base.buildConfigurations = xcprojBuildConfigurations(project.Configurations, project.BuildSettings)
	base.defaultConfigurationName = project.DefaultConfiguration

	return &XCProj{
		baseProj: base,
		content:  content,
		files:    project.Files,
	}, nil
}

func xcprojTargets(project xcproj.Project, logger log.Logger) []Target {
	idsByName := make(map[string]string, len(project.Targets))
	for _, target := range project.Targets {
		idsByName[target.Name] = target.ID
	}
	filePathsByID := xcprojFilePathsByID(project.Files)

	targets := make([]Target, 0, len(project.Targets))
	for _, target := range project.Targets {
		var dependencyTargetIDs []string
		for _, name := range target.Dependencies {
			dependencyTargetIDs = append(dependencyTargetIDs, idsByName[name])
		}

		var productPath string
		if len(target.Product) > 0 {
			// The last element is the file name: "product": "Products/App.app" decodes to [Products App.app].
			productPath = xcprojProductName(target.Name, target.Product[len(target.Product)-1])
		} else if filePath, ok := filePathsByID[target.ProductID]; ok {
			productPath = xcprojProductName(target.Name, path.Base(filePath))
		} else if target.ProductID != "" {
			logger.Warnf("product of target %s (%s) is referenced by ID %s, its path is unknown", target.Name, target.ID, target.ProductID)
		}

		targets = append(targets, Target{
			ID:                       target.ID,
			Name:                     target.Name,
			BuildConfigurations:      xcprojBuildConfigurations(project.Configurations, target.BuildSettings),
			DefaultConfigurationName: project.DefaultConfiguration,
			kind:                     xcprojTargetKinds[target.Kind],
			productType:              target.ProductType,
			productPath:              productPath,
			dependencyTargetIDs:      dependencyTargetIDs,
		})
	}

	return targets
}

func xcprojFilePathsByID(references []xcproj.Reference) map[string]string {
	pathsByID := map[string]string{}
	for _, reference := range references {
		if reference.ID != "" {
			pathsByID[reference.ID] = reference.Path
		}
		maps.Copy(pathsByID, xcprojFilePathsByID(reference.Children))
	}
	return pathsByID
}

// xcprojProductName falls back to the target name, PRODUCT_NAME's default, for a product file with
// no name, such as <PRODUCTS>/.app.
func xcprojProductName(targetName, fileName string) string {
	if extension := path.Ext(fileName); extension != "" && fileName == extension {
		return targetName + extension
	}
	return fileName
}

func xcprojBuildConfigurations(configurations []xcproj.Configuration, settings serialized.Object) []BuildConfiguration {
	buildConfigurations := make([]BuildConfiguration, 0, len(configurations))
	for _, configuration := range configurations {
		buildConfigurations = append(buildConfigurations, BuildConfiguration{
			ID:            configuration.ID,
			Name:          configuration.Name,
			buildSettings: xcproj.SettingsForConfiguration(settings, configuration.Name),
		})
	}
	return buildConfigurations
}
