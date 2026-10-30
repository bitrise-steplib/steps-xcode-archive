package xcodecommand

import "errors"

// TestWithoutBuildingParams describes an `xcodebuild test-without-building` invocation,
// the run half of test. It runs either an xctestrun (XCTestRun) or a scheme, whose test
// bundles xcodebuild finds in the build root (Scheme, with ProjectPath and TestPlan), not
// both. Zero-valued fields are omitted.
type TestWithoutBuildingParams struct {
	XCTestRun                      string // the .xctestrun file build-for-testing produced
	ProjectPath                    string // .xcodeproj, .xcworkspace or a Swift package (no flag; run in its directory)
	Scheme                         string
	TestPlan                       string
	Destination                    string // a user -destination is added, not replaced
	ResultBundlePath               string
	TestRepetitionMode             TestRepetitionMode
	MaximumTestRepetitions         int
	RelaunchTestsForEachRepetition bool
	OnlyTesting                    []string // -only-testing:<id>; user entries are added
	SkipTesting                    []string // -skip-testing:<id>; user entries are added
	CollectTestDiagnostics         string   // a default: a user -collect-test-diagnostics replaces it
	AdditionalOptions              Options  // the step's xcodebuild_options, from ParseOptions
}

// TestWithoutBuilding renders params into a test-without-building Command.
func TestWithoutBuilding(params TestWithoutBuildingParams) (Command, error) {
	switch {
	case params.XCTestRun != "" && (params.ProjectPath != "" || params.Scheme != "" || params.TestPlan != ""):
		// xcodebuild: "Cannot use -xctestrun with -project, -workspace or -scheme options."
		return Command{}, errors.New("test-without-building runs an xctestrun or a scheme, not both")
	case params.XCTestRun == "" && params.Scheme == "":
		return Command{}, errors.New("test-without-building needs an xctestrun or a scheme")
	}
	// -xctestrun App.xctestrun: test-without-building -xctestrun App.xctestrun -destination ...
	// a scheme, as test renders it: -project App.xcodeproj -scheme App test-without-building ...
	opts := containerOptions(params.ProjectPath)
	opts = appendValue(opts, "-scheme", params.Scheme)
	opts = append(opts, Option{Kind: Action, Name: ActionTestWithoutBuilding})
	opts = appendValue(opts, "-xctestrun", params.XCTestRun)
	opts = appendValue(opts, "-destination", params.Destination)
	opts = appendValue(opts, "-testPlan", params.TestPlan)
	run, err := testRunOptions{
		resultBundlePath:               params.ResultBundlePath,
		repetitionMode:                 params.TestRepetitionMode,
		maximumRepetitions:             params.MaximumTestRepetitions,
		relaunchTestsForEachRepetition: params.RelaunchTestsForEachRepetition,
		onlyTesting:                    params.OnlyTesting,
		skipTesting:                    params.SkipTesting,
		collectTestDiagnostics:         params.CollectTestDiagnostics,
	}.options()
	if err != nil {
		return Command{}, err
	}
	opts = append(opts, run...)

	return assemble(opts, params.AdditionalOptions, testWithoutBuildingPolicy)
}

var testWithoutBuildingPolicy = testRunPolicy(ActionTestWithoutBuilding, listsTests)
