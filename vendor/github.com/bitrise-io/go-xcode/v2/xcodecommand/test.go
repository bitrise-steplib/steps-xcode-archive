package xcodecommand

// TestParams describes an `xcodebuild test` invocation, which is build-for-testing
// followed by test-without-building; the fields are the union of the two. Zero-valued
// fields are omitted.
type TestParams struct {
	ProjectPath                    string // .xcodeproj, .xcworkspace or a Swift package (no flag; run in its directory)
	Scheme                         string
	Destination                    string // a user -destination is added, not replaced: test runs on several destinations
	TestPlan                       string
	XCConfigPath                   string
	Clean                          bool // run clean first
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

// Test renders params into a test Command.
func Test(params TestParams) (Command, error) {
	opts := containerOptions(params.ProjectPath)
	opts = appendValue(opts, "-scheme", params.Scheme)
	opts = append(opts, actions(params.Clean, ActionTest)...)
	opts = appendValue(opts, "-destination", params.Destination)
	opts = appendValue(opts, "-testPlan", params.TestPlan)
	opts = appendValue(opts, "-xcconfig", params.XCConfigPath)
	opts = append(opts, testRunOptions{
		resultBundlePath:               params.ResultBundlePath,
		repetitionMode:                 params.TestRepetitionMode,
		maximumRepetitions:             params.MaximumTestRepetitions,
		relaunchTestsForEachRepetition: params.RelaunchTestsForEachRepetition,
		onlyTesting:                    params.OnlyTesting,
		skipTesting:                    params.SkipTesting,
		collectTestDiagnostics:         params.CollectTestDiagnostics,
	}.options()...)

	return assemble(opts, params.AdditionalOptions, testPolicy, params.Validation)
}
