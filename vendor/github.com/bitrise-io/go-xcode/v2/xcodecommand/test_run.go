package xcodecommand

import (
	"fmt"
	"strconv"
)

// TestRepetitionMode says how xcodebuild repeats tests; the values are the test steps'
// test_repetition_mode input values.
type TestRepetitionMode string

// Test repetition modes. Every mode but none runs up to MaximumTestRepetitions times
// (-test-iterations).
const (
	TestRepetitionNone               TestRepetitionMode = "none"
	TestRepetitionUntilFailure       TestRepetitionMode = "until_failure"                // -run-tests-until-failure
	TestRepetitionRetryOnFailure     TestRepetitionMode = "retry_on_failure"             // -retry-tests-on-failure
	TestRepetitionUpUntilMaximumRuns TestRepetitionMode = "up_until_maximum_repetitions" // -test-iterations alone
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

// ValidateTestRepetition checks the test steps' repetition inputs: a known mode, at least
// two repetitions for a repeating mode (xcodebuild refuses fewer), and relaunching only
// with repetition (xcodebuild refuses it otherwise). Test and TestWithoutBuilding apply it;
// a step calls it while processing its inputs to fail before any work.
func ValidateTestRepetition(mode TestRepetitionMode, maximumRepetitions int, relaunchTestsForEachRepetition bool) error {
	switch mode {
	case "", TestRepetitionNone:
		if relaunchTestsForEachRepetition {
			return fmt.Errorf("relaunch_tests_for_each_repetition needs a test_repetition_mode other than %s", TestRepetitionNone)
		}
		return nil
	case TestRepetitionUntilFailure, TestRepetitionRetryOnFailure, TestRepetitionUpUntilMaximumRuns:
		if maximumRepetitions < 2 {
			return fmt.Errorf("test_repetition_mode %s needs a maximum_test_repetitions of at least 2, got %d", mode, maximumRepetitions)
		}
		return nil
	default:
		return fmt.Errorf("test_repetition_mode %q is not one of %s, %s, %s, %s", mode,
			TestRepetitionNone, TestRepetitionUntilFailure, TestRepetitionRetryOnFailure, TestRepetitionUpUntilMaximumRuns)
	}
}

func (r testRunOptions) options() (Options, error) {
	if err := ValidateTestRepetition(r.repetitionMode, r.maximumRepetitions, r.relaunchTestsForEachRepetition); err != nil {
		return nil, err
	}
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
	return appendValue(opts, "-collect-test-diagnostics", r.collectTestDiagnostics), nil
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
