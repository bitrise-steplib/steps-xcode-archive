package xcscheme

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

// BuildableReference ...
type BuildableReference struct {
	BuildableIdentifier string `xml:"BuildableIdentifier,attr"`
	BlueprintIdentifier string `xml:"BlueprintIdentifier,attr"`
	BuildableName       string `xml:"BuildableName,attr"`
	BlueprintName       string `xml:"BlueprintName,attr"`
	ReferencedContainer string `xml:"ReferencedContainer,attr"`
}

// IsAppReference ...
func (r BuildableReference) IsAppReference() bool {
	return filepath.Ext(r.BuildableName) == ".app"
}

func (r BuildableReference) isTestProduct() bool {
	return filepath.Ext(r.BuildableName) == ".xctest"
}

// ReferencedContainerAbsPath ...
func (r BuildableReference) ReferencedContainerAbsPath(schemeContainerDir string) (string, error) {
	s := strings.Split(r.ReferencedContainer, ":")
	if len(s) != 2 {
		return "", fmt.Errorf("unknown referenced container (%s)", r.ReferencedContainer)
	}

	base := s[1]
	absPth := filepath.Join(schemeContainerDir, base)

	return pathutil.AbsPath(absPth)
}

// BuildActionEntry ...
type BuildActionEntry struct {
	BuildForTesting   string `xml:"buildForTesting,attr"`
	BuildForRunning   string `xml:"buildForRunning,attr"`
	BuildForProfiling string `xml:"buildForProfiling,attr"`
	BuildForArchiving string `xml:"buildForArchiving,attr"`
	BuildForAnalyzing string `xml:"buildForAnalyzing,attr"`

	BuildableReference BuildableReference
}

// BuildAction ...
type BuildAction struct {
	ParallelizeBuildables     string             `xml:"parallelizeBuildables,attr"`
	BuildImplicitDependencies string             `xml:"buildImplicitDependencies,attr"`
	BuildActionEntries        []BuildActionEntry `xml:"BuildActionEntries>BuildActionEntry"`
}

// TestableReference ...
type TestableReference struct {
	Skipped        string `xml:"skipped,attr"`
	Parallelizable string `xml:"parallelizable,attr,omitempty"`

	BuildableReference BuildableReference
}

func (r TestableReference) isTestable() bool {
	return r.Skipped == "NO" && r.BuildableReference.isTestProduct()
}

// TestPlanReference ...
type TestPlanReference struct {
	Reference string `xml:"reference,attr,omitempty"`
	Default   string `xml:"default,attr,omitempty"`
}

// IsDefault ...
func (r TestPlanReference) IsDefault() bool {
	return r.Default == "YES"
}

// Name ...
func (r TestPlanReference) Name() string {
	// reference = "container:FullTests.xctestplan"
	idx := strings.Index(r.Reference, ":")
	testPlanFileName := r.Reference[idx+1:]
	return strings.TrimSuffix(testPlanFileName, filepath.Ext(testPlanFileName))
}

// MacroExpansion ...
type MacroExpansion struct {
	BuildableReference BuildableReference
}

// AdditionalOptions ...
type AdditionalOptions struct {
}

// TestPlans ...
type TestPlans struct {
	TestPlanReferences []TestPlanReference `xml:"TestPlanReference,omitempty"`
}

// TestAction ...
type TestAction struct {
	BuildConfiguration           string `xml:"buildConfiguration,attr"`
	SelectedDebuggerIdentifier   string `xml:"selectedDebuggerIdentifier,attr"`
	SelectedLauncherIdentifier   string `xml:"selectedLauncherIdentifier,attr"`
	ShouldUseLaunchSchemeArgsEnv string `xml:"shouldUseLaunchSchemeArgsEnv,attr"`

	// TODO: This property means that a TestPlan belongs to this test action.
	//   As long as the related testPlan has default settings it is not created as a separate TestPlan file.
	//   If any default test plan setting is changed, Xcode creates the TestPlan file, adds a TestPlans entry to the scheme and removes this property from the TestAction.
	//   Code working with test plans should be updated to consider this new property.
	ShouldAutocreateTestPlan string `xml:"shouldAutocreateTestPlan,attr,omitempty"`

	Testables         []TestableReference `xml:"Testables>TestableReference"`
	TestPlans         *TestPlans
	MacroExpansion    MacroExpansion
	AdditionalOptions AdditionalOptions
}

// BuildableProductRunnable ...
type BuildableProductRunnable struct {
	RunnableDebuggingMode string `xml:"runnableDebuggingMode,attr"`
	BuildableReference    BuildableReference
}

// LaunchAction ...
type LaunchAction struct {
	BuildConfiguration             string `xml:"buildConfiguration,attr"`
	SelectedDebuggerIdentifier     string `xml:"selectedDebuggerIdentifier,attr"`
	SelectedLauncherIdentifier     string `xml:"selectedLauncherIdentifier,attr"`
	LaunchStyle                    string `xml:"launchStyle,attr"`
	UseCustomWorkingDirectory      string `xml:"useCustomWorkingDirectory,attr"`
	IgnoresPersistentStateOnLaunch string `xml:"ignoresPersistentStateOnLaunch,attr"`
	DebugDocumentVersioning        string `xml:"debugDocumentVersioning,attr"`
	DebugServiceExtension          string `xml:"debugServiceExtension,attr"`
	AllowLocationSimulation        string `xml:"allowLocationSimulation,attr"`
	BuildableProductRunnable       BuildableProductRunnable
	AdditionalOptions              AdditionalOptions
}

// ProfileAction ...
type ProfileAction struct {
	BuildConfiguration           string `xml:"buildConfiguration,attr"`
	ShouldUseLaunchSchemeArgsEnv string `xml:"shouldUseLaunchSchemeArgsEnv,attr"`
	SavedToolIdentifier          string `xml:"savedToolIdentifier,attr"`
	UseCustomWorkingDirectory    string `xml:"useCustomWorkingDirectory,attr"`
	DebugDocumentVersioning      string `xml:"debugDocumentVersioning,attr"`
	BuildableProductRunnable     BuildableProductRunnable
}

// AnalyzeAction ...
type AnalyzeAction struct {
	BuildConfiguration string `xml:"buildConfiguration,attr"`
}

// ArchiveAction ...
type ArchiveAction struct {
	BuildConfiguration       string `xml:"buildConfiguration,attr"`
	RevealArchiveInOrganizer string `xml:"revealArchiveInOrganizer,attr"`
}

// Scheme ...
type Scheme struct {
	// The last known Xcode version.
	LastUpgradeVersion string `xml:"LastUpgradeVersion,attr"`
	// The version of `.xcscheme` files supported.
	Version string `xml:"version,attr"`

	BuildAction   BuildAction
	TestAction    TestAction
	LaunchAction  LaunchAction
	ProfileAction ProfileAction
	AnalyzeAction AnalyzeAction
	ArchiveAction ArchiveAction

	Name     string `xml:"-"`
	Path     string `xml:"-"`
	IsShared bool   `xml:"-"`
}

// Open ...
func Open(pth string) (Scheme, error) {
	var start = time.Now()

	f, err := os.Open(pth)
	if err != nil {
		return Scheme{}, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Warnf("Failed to close scheme: %s: %s", pth, err)
		}
	}()

	scheme, err := parse(f)
	if err != nil {
		return Scheme{}, fmt.Errorf("failed to unmarshal scheme file: %s: %s", pth, err)
	}

	scheme.Name = strings.TrimSuffix(filepath.Base(pth), filepath.Ext(pth))
	scheme.Path = pth

	log.Printf("Read %s scheme in %s.", scheme.Name, time.Since(start).Round(time.Second))

	return scheme, nil
}

func parse(reader io.Reader) (scheme Scheme, err error) {
	err = xml.NewDecoder(reader).Decode(&scheme)
	return
}

// XMLToken ...
type XMLToken int

const (
	// XMLStart ...
	XMLStart XMLToken = 1
	// XMLEnd ...
	XMLEnd XMLToken = 2
	// XMLAttribute ...
	XMLAttribute XMLToken = 3
)

// Marshal ...
func (s Scheme) Marshal() ([]byte, error) {
	contents, err := xml.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Scheme: %v", err)
	}

	contentsNewline := strings.ReplaceAll(string(contents), "><", ">\n<")

	// Place XML Attributes on separate lines
	re := regexp.MustCompile(`\s([^=<>/]*)\s?=\s?"([^=<>/]*)"`)
	contentsNewline = re.ReplaceAllString(contentsNewline, "\n$1 = \"$2\"")

	var contentsIndented string

	indent := 0
	for _, line := range strings.Split(contentsNewline, "\n") {
		currentLine := XMLAttribute
		if strings.HasPrefix(line, "</") {
			currentLine = XMLEnd
		} else if strings.HasPrefix(line, "<") {
			currentLine = XMLStart
		}

		if currentLine == XMLEnd && indent != 0 {
			indent--
		}

		contentsIndented += strings.Repeat("   ", indent)
		contentsIndented += line + "\n"

		if currentLine == XMLStart {
			indent++
		}
	}

	return []byte(xml.Header + contentsIndented), nil
}

// AppBuildActionEntry ...
func (s Scheme) AppBuildActionEntry() (BuildActionEntry, bool) {
	var entry BuildActionEntry
	for _, e := range s.BuildAction.BuildActionEntries {
		if e.BuildForArchiving != "YES" {
			continue
		}
		if !e.BuildableReference.IsAppReference() {
			continue
		}
		entry = e
		break
	}

	return entry, (entry.BuildableReference.BlueprintIdentifier != "")
}

// IsTestable returns true if Test is a valid action
func (s Scheme) IsTestable() bool {
	for _, testEntry := range s.TestAction.Testables {
		if testEntry.isTestable() {
			return true
		}
	}

	return false
}

// DefaultTestPlan ...
func (s Scheme) DefaultTestPlan() *TestPlanReference {
	if s.TestAction.TestPlans == nil {
		return nil
	}

	testPlans := *s.TestAction.TestPlans

	for _, testPlanRef := range testPlans.TestPlanReferences {
		if testPlanRef.IsDefault() {
			return &testPlanRef
		}
	}
	return nil
}
