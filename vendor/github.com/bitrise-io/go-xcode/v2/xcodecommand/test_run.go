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

// testRunOptions are the flags of the run half shared by test and test-without-building.
type testRunOptions struct {
	resultBundlePath               string
	repetitionMode                 TestRepetitionMode
	maximumRepetitions             int
	relaunchTestsForEachRepetition bool
	onlyTesting                    []string
	skipTesting                    []string
	collectTestDiagnostics         string
}

func (r testRunOptions) options() Options {
	opts := appendValue(nil, "-resultBundlePath", r.resultBundlePath)

	switch r.repetitionMode {
	case TestRepetitionUntilFailure:
		opts = append(opts, Option{Kind: Switch, Name: "-run-tests-until-failure"})
	case TestRepetitionRetryOnFailure:
		opts = append(opts, Option{Kind: Switch, Name: "-retry-tests-on-failure"})
	}
	if r.repetitionMode != "" && r.repetitionMode != TestRepetitionNone {
		opts = appendValue(opts, "-test-iterations", strconv.Itoa(r.maximumRepetitions))
	}
	if r.relaunchTestsForEachRepetition {
		opts = appendValue(opts, "-test-repetition-relaunch-enabled", "YES")
	}

	for _, id := range r.onlyTesting {
		opts = append(opts, Option{Kind: ColonOption, Name: "-only-testing", Value: id})
	}
	for _, id := range r.skipTesting {
		opts = append(opts, Option{Kind: ColonOption, Name: "-skip-testing", Value: id})
	}
	return appendValue(opts, "-collect-test-diagnostics", r.collectTestDiagnostics)
}

// testRunPolicy is the policy shared by the test actions: selection flags and -destination
// are appendable (xcodebuild applies -only-testing before -skip-testing, and runs on
// every destination), -collect-test-diagnostics is a default.
func testRunPolicy(name string, extra ...rejection) actionPolicy {
	return actionPolicy{
		name:       name,
		rejections: append([]rejection{modeSwitching}, extra...),
		defaults:   []string{"-collect-test-diagnostics"},
		appendable: []string{"-only-testing", "-skip-testing", "-only-test-configuration", "-skip-test-configuration", "-destination", "-arch"},
	}
}
