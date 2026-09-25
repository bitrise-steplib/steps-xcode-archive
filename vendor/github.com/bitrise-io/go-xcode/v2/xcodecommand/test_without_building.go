package xcodecommand

// TestWithoutBuildingParams describes an `xcodebuild test-without-building` invocation,
// the run half of test. Zero-valued fields are omitted.
type TestWithoutBuildingParams struct {
	XCTestRun                      string // the .xctestrun file build-for-testing produced
	Destination                    string // a user -destination is added, not replaced
	ResultBundlePath               string
	TestRepetitionMode             TestRepetitionMode
	MaximumTestRepetitions         int
	RelaunchTestsForEachRepetition bool
	OnlyTesting                    []string // -only-testing:<id>; user entries are added
	SkipTesting                    []string // -skip-testing:<id>; user entries are added
	CollectTestDiagnostics         string   // a default: a user -collect-test-diagnostics replaces it
	AdditionalOptions              []string // the step's xcodebuild_options, shell-split
	Validation                     Validation
}

// TestWithoutBuilding renders params into a test-without-building Command.
func TestWithoutBuilding(params TestWithoutBuildingParams) (Command, error) {
	opts := Options{{Kind: Action, Name: ActionTestWithoutBuilding}}
	opts = appendValue(opts, "-xctestrun", params.XCTestRun)
	opts = appendValue(opts, "-destination", params.Destination)
	opts = append(opts, testRunOptions{
		resultBundlePath:               params.ResultBundlePath,
		repetitionMode:                 params.TestRepetitionMode,
		maximumRepetitions:             params.MaximumTestRepetitions,
		relaunchTestsForEachRepetition: params.RelaunchTestsForEachRepetition,
		onlyTesting:                    params.OnlyTesting,
		skipTesting:                    params.SkipTesting,
		collectTestDiagnostics:         params.CollectTestDiagnostics,
	}.options()...)

	return assemble(opts, params.AdditionalOptions, testWithoutBuildingPolicy, params.Validation)
}
