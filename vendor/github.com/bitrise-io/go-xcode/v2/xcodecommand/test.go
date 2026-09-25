package xcodecommand

import "strconv"

// TestRepetitionMode is the -run-tests-until-failure / -retry-tests-on-failure choice.
type TestRepetitionMode string

// Test repetition modes.
const (
	TestRepetitionNone           TestRepetitionMode = "none"
	TestRepetitionUntilFailure   TestRepetitionMode = "until_failure"
	TestRepetitionRetryOnFailure TestRepetitionMode = "retry_on_failure"
)

// TestParams describes an `xcodebuild test` invocation. Zero-valued fields are omitted.
// Argument order follows the xcode-test step.
type TestParams struct {
	ProjectPath                    string // .xcodeproj, .xcworkspace or a Swift package (no flag; run in its directory)
	Scheme                         string
	Destination                    string // a user -destination is added, not replaced: test runs on several destinations
	TestPlan                       string
	ResultBundlePath               string
	TestRepetitionMode             TestRepetitionMode
	MaximumTestRepetitions         int
	RelaunchTestsForEachRepetition bool
	XCConfigPath                   string
	Clean                          bool     // run clean first
	SkipTesting                    []string // -skip-testing:<id>; user entries are added, not replaced
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
	opts = appendValue(opts, "-resultBundlePath", params.ResultBundlePath)

	switch params.TestRepetitionMode {
	case TestRepetitionUntilFailure:
		opts = append(opts, Option{Kind: Switch, Name: "-run-tests-until-failure"})
	case TestRepetitionRetryOnFailure:
		opts = append(opts, Option{Kind: Switch, Name: "-retry-tests-on-failure"})
	}
	if params.TestRepetitionMode != "" && params.TestRepetitionMode != TestRepetitionNone {
		opts = appendValue(opts, "-test-iterations", strconv.Itoa(params.MaximumTestRepetitions))
	}
	if params.RelaunchTestsForEachRepetition {
		opts = appendValue(opts, "-test-repetition-relaunch-enabled", "YES")
	}

	opts = appendValue(opts, "-xcconfig", params.XCConfigPath)
	for _, test := range params.SkipTesting {
		opts = append(opts, Option{Kind: ColonOption, Name: "-skip-testing", Value: test})
	}
	opts = appendValue(opts, "-collect-test-diagnostics", params.CollectTestDiagnostics)

	return assemble(opts, params.AdditionalOptions, testSpec, params.Validation)
}
