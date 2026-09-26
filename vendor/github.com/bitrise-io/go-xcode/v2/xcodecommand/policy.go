package xcodecommand

import "slices"

// actionPolicy is a command's policy for additional options; anything not listed passes.
type actionPolicy struct {
	name       string
	rejects    []rejection
	defaults   []string // derived keys the user's option replaces
	appendable []string // repeatable keys where derived and user entries both stay
}

// rejection is a group of flags a command refuses, with the reason for all of them.
type rejection struct {
	flags  []string
	reason string
}

func (r rejection) except(flags ...string) rejection {
	return rejection{
		flags:  slices.DeleteFunc(slices.Clone(r.flags), func(f string) bool { return slices.Contains(flags, f) }),
		reason: r.reason,
	}
}

var (
	modeSwitching = rejection{
		reason: "switches xcodebuild into another mode",
		flags: []string{
			"-exportArchive", "-showBuildSettings", "-resolvePackageDependencies", "-list", "-version",
			"-showsdks", "-showdestinations", "-showTestPlans", "-create-xcframework", "-usage", "-help",
		},
	}
	testOnly = rejection{
		reason: "applies to test actions only",
		flags: []string{
			"-testPlan", "-xctestrun", "-only-testing", "-skip-testing", "-only-test-configuration",
			"-skip-test-configuration", "-test-iterations", "-run-tests-until-failure", "-retry-tests-on-failure",
			"-test-repetition-relaunch-enabled", "-parallel-testing-enabled", "-parallel-testing-worker-count",
			"-collect-test-diagnostics", "-testLanguage", "-testRegion",
			"-maximum-concurrent-test-device-destinations", "-maximum-concurrent-test-simulator-destinations",
		},
	}
	withoutBuildingOnly = rejection{flags: []string{"-xctestrun"}, reason: "applies to test-without-building only"}
)

var (
	// The build family: a step's -destination is a fallback the user's replaces, -arch
	// is repeatable.
	archivePolicy = actionPolicy{
		name:       ActionArchive,
		rejects:    []rejection{modeSwitching, testOnly},
		defaults:   []string{"-destination"},
		appendable: []string{"-arch"},
	}
	buildPolicy = actionPolicy{
		name:       ActionBuild,
		rejects:    []rejection{modeSwitching, testOnly},
		defaults:   []string{"-destination"},
		appendable: []string{"-arch"},
	}
	analyzePolicy = actionPolicy{
		name:       ActionAnalyze,
		rejects:    []rejection{modeSwitching, testOnly},
		defaults:   []string{"-destination", "-resultBundlePath"},
		appendable: []string{"-arch"},
	}
	// build-for-testing takes the test selection flags, which it bakes into the xctestrun.
	buildForTestingPolicy = actionPolicy{
		name:       ActionBuildForTesting,
		rejects:    []rejection{modeSwitching},
		defaults:   []string{"-destination"},
		appendable: []string{"-arch"},
	}
	exportArchivePolicy = actionPolicy{
		name:    "export archive",
		rejects: []rejection{modeSwitching.except("-exportArchive"), testOnly},
	}
	resolvePackagesPolicy = actionPolicy{
		name:    "resolve packages",
		rejects: []rejection{modeSwitching.except("-resolvePackageDependencies"), testOnly},
	}
	testPolicy                = testRunPolicy(ActionTest, withoutBuildingOnly)
	testWithoutBuildingPolicy = testRunPolicy(ActionTestWithoutBuilding)
	showBuildSettingsPolicy   = actionPolicy{
		name:    "show build settings",
		rejects: []rejection{modeSwitching.except("-showBuildSettings"), testOnly},
	}
)
