package xcodecommand

import "slices"

// actionPolicy is a command's policy for additional options; anything not listed passes.
type actionPolicy struct {
	name       string
	rejections []rejection
	defaults   []string // derived keys the user's option replaces
	appendable []string // repeatable keys where derived and user entries both stay
}

// rejection is a group of flags a command refuses, with the reason for all of them.
type rejection struct {
	flags  []string
	reason string
}

// rejects says whether the command refuses the flag option o, and why.
func (s actionPolicy) rejects(o Option) (rejection, bool) {
	if o.Kind != Switch && o.Kind != ValueOption && o.Kind != ColonOption {
		return rejection{}, false
	}
	for _, r := range s.rejections {
		if slices.Contains(r.flags, o.Name) {
			return r, true
		}
	}
	return rejection{}, false
}

func (r rejection) except(flags ...string) rejection {
	return rejection{
		flags:  slices.DeleteFunc(slices.Clone(r.flags), func(f string) bool { return slices.Contains(flags, f) }),
		reason: r.reason,
	}
}

var (
	// The flags of `xcodebuild -help` (Xcode 26.5 and 27.0) that make it do something other
	// than the command. -json turns a build or archive into a silent no-op (exit 0, nothing
	// built); on -showBuildSettings it switches the output to JSON, which the text-parsing
	// build settings readers cannot read.
	modeSwitching = rejection{
		reason: "It switches xcodebuild into another mode, so the command does not run as the Step expects.",
		flags: []string{
			"-exportArchive", "-exportNotarizedApp", "-showBuildSettings", "-showBuildSettingsForIndex",
			"-resolvePackageDependencies", "-list", "-version", "-showsdks", "-showdestinations", "-showTestPlans",
			"-create-xcframework", "-exportLocalizations", "-importLocalizations", "-find-executable", "-find-library",
			"-convert-project", "-json", "-usage", "-help", "-license", "-checkFirstLaunchStatus", "-runFirstLaunch",
			"-downloadPlatform", "-downloadAllPlatforms", "-importPlatform", "-prepareDeviceSupport",
			"-downloadComponent", "-importComponent", "-deleteComponent", "-showComponent",
		},
	}
	// Test flags xcodebuild refuses outside a test action ("only supported when testing",
	// or "Cannot use -xctestrun with ... -scheme"), verified on build and archive.
	testOnlyRefused = rejection{
		reason: "It applies to test actions only, and xcodebuild refuses it here.",
		flags:  []string{"-testPlan", "-xctestrun", "-testLanguage", "-testRegion", "-testProductsPath", "-enableCodeCoverage"},
	}
	// Test flags xcodebuild accepts and ignores outside a test action.
	testOnly = rejection{
		reason: "It applies to test actions only.",
		flags: []string{
			"-only-testing", "-skip-testing", "-only-test-configuration", "-skip-test-configuration",
			"-test-iterations", "-run-tests-until-failure", "-retry-tests-on-failure", "-test-repetition-relaunch-enabled",
			"-parallel-testing-enabled", "-parallel-testing-worker-count", "-maximum-parallel-testing-workers",
			"-maximum-concurrent-test-device-destinations", "-maximum-concurrent-test-simulator-destinations",
			"-collect-test-diagnostics", "-enablePerformanceTestsDiagnostics", "-test-timeouts-enabled",
			"-default-test-execution-time-allowance", "-maximum-test-execution-time-allowance",
			"-enumerate-tests", "-test-enumeration-style", "-test-enumeration-format", "-test-enumeration-output-path",
		},
	}
	// On a test run, -enumerate-tests lists the tests and runs none, and the command succeeds.
	listsTests = rejection{reason: "It makes xcodebuild list the tests instead of running them.", flags: []string{"-enumerate-tests"}}
	// test refuses -xctestrun next to its -scheme; test-without-building refuses -testPlan
	// next to its -xctestrun (the xctestrun already carries its test plan).
	withoutBuildingOnly = rejection{reason: "It applies to test-without-building only, and xcodebuild refuses it next to -scheme.", flags: []string{"-xctestrun"}}
	withXCTestRun       = rejection{reason: "The xctestrun already selects the test plan, and xcodebuild refuses it next to -xctestrun.", flags: []string{"-testPlan"}}
)

var (
	// The build family: a step's -destination is a fallback the user's replaces, -arch
	// is repeatable.
	archivePolicy = actionPolicy{
		name:       ActionArchive,
		rejections: []rejection{modeSwitching, testOnlyRefused, testOnly},
		defaults:   []string{"-destination"},
		appendable: []string{"-arch"},
	}
	buildPolicy = actionPolicy{
		name:       ActionBuild,
		rejections: []rejection{modeSwitching, testOnlyRefused, testOnly},
		defaults:   []string{"-destination"},
		appendable: []string{"-arch"},
	}
	analyzePolicy = actionPolicy{
		name:       ActionAnalyze,
		rejections: []rejection{modeSwitching, testOnlyRefused, testOnly},
		defaults:   []string{"-destination", "-resultBundlePath"},
		appendable: []string{"-arch"},
	}
	// build-for-testing takes the test selection flags, which it bakes into the xctestrun.
	buildForTestingPolicy = actionPolicy{
		name:       ActionBuildForTesting,
		rejections: []rejection{modeSwitching},
		defaults:   []string{"-destination"},
		appendable: []string{"-arch"},
	}
	exportArchivePolicy = actionPolicy{
		name:       "export archive",
		rejections: []rejection{modeSwitching.except("-exportArchive"), testOnlyRefused, testOnly},
	}
	resolvePackagesPolicy = actionPolicy{
		name:       "resolve packages",
		rejections: []rejection{modeSwitching.except("-resolvePackageDependencies"), testOnlyRefused, testOnly},
	}
	testPolicy                = testRunPolicy(ActionTest, listsTests, withoutBuildingOnly)
	testWithoutBuildingPolicy = testRunPolicy(ActionTestWithoutBuilding, listsTests, withXCTestRun)
	showBuildSettingsPolicy   = actionPolicy{
		name:       "show build settings",
		rejections: []rejection{modeSwitching.except("-showBuildSettings"), testOnlyRefused, testOnly},
	}
)
