package xcodeproj

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/bitrise-io/go-xcode/v2/internal/filewriter"
	"github.com/bitrise-io/go-xcode/v2/xcodeproject/xcscheme"
)

const (
	yes                         = "YES"
	no                          = "NO"
	buildableID                 = "primary"
	defaultDebugConfiguration   = "Debug"
	defaultReleaseConfiguration = "Release"
	debuggerID                  = "Xcode.DebuggerFoundation.Debugger.LLDB"
	launcherID                  = "Xcode.DebuggerFoundation.Launcher.LLDB"
)

// newSharedSchemeFileMode is the mode of a newly created scheme file; an existing one keeps its mode.
const newSharedSchemeFileMode = 0o600

// RecreateSchemes returns the schemes Xcode would create for the project, one per native, non-test
// target. It only builds them in memory; use SaveSharedScheme to write them.
func (p *XcodeProj) RecreateSchemes() []xcscheme.Scheme {
	p.logger.Printf("Recreating Xcode schemes")

	schemes := p.generateSchemes()

	p.logger.Printf("Recreated %d Xcode scheme(s)", len(schemes))

	return schemes
}

// SaveSharedScheme writes the scheme to <Path>/xcshareddata/xcschemes/<scheme name>.xcscheme,
// overwriting an existing file. Missing directories are created.
func (p *XcodeProj) SaveSharedScheme(scheme xcscheme.Scheme) error {
	dir := p.sharedSchemesDir()
	pth := filepath.Join(dir, scheme.Name+xcschemeExtension)

	content, err := scheme.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal scheme: %w", err)
	}

	if err := p.dirCreator.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create scheme directory (%s): %w", dir, err)
	}

	if err := filewriter.WriteKeepingMode(p.fileManager, pth, content, newSharedSchemeFileMode); err != nil {
		return fmt.Errorf("failed to write scheme file (%s): %w", pth, err)
	}

	return nil
}

// generateSchemes builds the schemes Xcode would autocreate: one per native, non-test target.
func (p *XcodeProj) generateSchemes() []xcscheme.Scheme {
	projectName := path.Base(p.Path)

	var schemes []xcscheme.Scheme
	for _, buildTarget := range p.targets {
		if !buildTarget.isNativeTarget() || buildTarget.isTest() {
			continue
		}

		var testTargets []Target
		for _, testTarget := range p.targets {
			if testTarget.isTest() && testTarget.DependsOn(buildTarget.ID) {
				testTargets = append(testTargets, testTarget)
			}
		}

		schemes = append(schemes, newScheme(buildTarget, testTargets, projectName))
	}

	return schemes
}

func newScheme(buildTarget Target, testTargets []Target, projectName string) xcscheme.Scheme {
	return xcscheme.Scheme{
		Name:               buildTarget.Name,
		LastUpgradeVersion: "1240",
		Version:            "1.3",
		BuildAction:        newBuildAction(buildTarget, projectName),
		TestAction:         newTestAction(buildTarget, testTargets, projectName),
		LaunchAction:       newLaunchAction(buildTarget, projectName),
		ProfileAction:      newProfileAction(buildTarget, projectName),
		AnalyzeAction:      newAnalyzeAction(buildTarget),
		ArchiveAction:      newArchiveAction(buildTarget),
	}
}

func newBuildableReference(target Target, projectName string) xcscheme.BuildableReference {
	return xcscheme.BuildableReference{
		BuildableIdentifier: buildableID,
		BlueprintIdentifier: target.ID,
		BuildableName:       path.Base(target.productPath),
		BlueprintName:       target.Name,
		ReferencedContainer: fmt.Sprintf("container:%s", projectName),
	}
}

func newBuildAction(target Target, projectName string) xcscheme.BuildAction {
	return xcscheme.BuildAction{
		ParallelizeBuildables:     yes,
		BuildImplicitDependencies: yes,
		BuildActionEntries: []xcscheme.BuildActionEntry{
			{
				BuildForTesting:    yes,
				BuildForRunning:    yes,
				BuildForProfiling:  yes,
				BuildForArchiving:  yes,
				BuildForAnalyzing:  yes,
				BuildableReference: newBuildableReference(target, projectName),
			},
		},
	}
}

func newTestableReference(target Target, projectName string) xcscheme.TestableReference {
	return xcscheme.TestableReference{
		Skipped:            no,
		BuildableReference: newBuildableReference(target, projectName),
	}
}

func newTestAction(buildTarget Target, testTargets []Target, projectName string) xcscheme.TestAction {
	if len(testTargets) == 0 {
		return xcscheme.TestAction{}
	}

	testAction := xcscheme.TestAction{
		BuildConfiguration:           debugConfigurationName(testTargets[0]),
		SelectedDebuggerIdentifier:   debuggerID,
		SelectedLauncherIdentifier:   launcherID,
		ShouldUseLaunchSchemeArgsEnv: yes,
		MacroExpansion: xcscheme.MacroExpansion{
			BuildableReference: newBuildableReference(buildTarget, projectName),
		},
		Testables: []xcscheme.TestableReference{},
	}

	for _, testTarget := range testTargets {
		testAction.Testables = append(testAction.Testables, newTestableReference(testTarget, projectName))
	}

	return testAction
}

func newBuildableProductRunnable(target Target, projectName string) xcscheme.BuildableProductRunnable {
	return xcscheme.BuildableProductRunnable{
		RunnableDebuggingMode: "0",
		BuildableReference:    newBuildableReference(target, projectName),
	}
}

func newLaunchAction(target Target, projectName string) xcscheme.LaunchAction {
	return xcscheme.LaunchAction{
		BuildConfiguration:             debugConfigurationName(target),
		SelectedDebuggerIdentifier:     debuggerID,
		SelectedLauncherIdentifier:     launcherID,
		LaunchStyle:                    "0",
		UseCustomWorkingDirectory:      no,
		IgnoresPersistentStateOnLaunch: no,
		DebugDocumentVersioning:        yes,
		DebugServiceExtension:          "internal",
		AllowLocationSimulation:        yes,
		BuildableProductRunnable:       newBuildableProductRunnable(target, projectName),
	}
}

func newProfileAction(target Target, projectName string) xcscheme.ProfileAction {
	return xcscheme.ProfileAction{
		BuildConfiguration:           releaseConfigurationName(target),
		ShouldUseLaunchSchemeArgsEnv: yes,
		UseCustomWorkingDirectory:    no,
		DebugDocumentVersioning:      yes,
		BuildableProductRunnable:     newBuildableProductRunnable(target, projectName),
	}
}

func newAnalyzeAction(target Target) xcscheme.AnalyzeAction {
	return xcscheme.AnalyzeAction{
		BuildConfiguration: debugConfigurationName(target),
	}
}

func newArchiveAction(target Target) xcscheme.ArchiveAction {
	return xcscheme.ArchiveAction{
		BuildConfiguration:       releaseConfigurationName(target),
		RevealArchiveInOrganizer: yes,
	}
}

func debugConfigurationName(target Target) string {
	return configurationNameOrDefault(target, defaultDebugConfiguration)
}

func releaseConfigurationName(target Target) string {
	return configurationNameOrDefault(target, defaultReleaseConfiguration)
}

func configurationNameOrDefault(target Target, name string) string {
	for _, configuration := range target.BuildConfigurations {
		if configuration.Name == name {
			return name
		}
	}
	return target.DefaultConfigurationName
}
