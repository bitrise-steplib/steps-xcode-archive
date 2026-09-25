package xcodecommand

import "fmt"

// actionSpec is a command's policy for additional options; anything not listed passes.
type actionSpec struct {
	name       string
	rejected   map[string]string // flag -> reason
	defaults   map[string]bool   // derived keys the user's option replaces
	appendable map[string]bool   // repeatable keys where derived and user entries both stay
}

var modeSwitchingFlags = map[string]string{
	"-exportArchive":              "switches xcodebuild into export mode",
	"-showBuildSettings":          "switches xcodebuild into build settings listing mode",
	"-resolvePackageDependencies": "switches xcodebuild into package resolution mode",
	"-list":                       "switches xcodebuild into listing mode",
	"-version":                    "switches xcodebuild into version printing mode",
	"-showsdks":                   "switches xcodebuild into SDK listing mode",
	"-showdestinations":           "switches xcodebuild into destination listing mode",
	"-showTestPlans":              "switches xcodebuild into test plan listing mode",
	"-create-xcframework":         "switches xcodebuild into xcframework creation mode",
	"-usage":                      "switches xcodebuild into help mode",
	"-help":                       "switches xcodebuild into help mode",
}

var testOnlyFlags = map[string]string{
	"-xctestrun":                                      "applies to test-without-building only",
	"-only-testing":                                   "applies to test actions only",
	"-skip-testing":                                   "applies to test actions only",
	"-only-test-configuration":                        "applies to test actions only",
	"-skip-test-configuration":                        "applies to test actions only",
	"-test-iterations":                                "applies to test actions only",
	"-run-tests-until-failure":                        "applies to test actions only",
	"-retry-tests-on-failure":                         "applies to test actions only",
	"-test-repetition-relaunch-enabled":               "applies to test actions only",
	"-parallel-testing-enabled":                       "applies to test actions only",
	"-parallel-testing-worker-count":                  "applies to test actions only",
	"-collect-test-diagnostics":                       "applies to test actions only",
	"-testLanguage":                                   "applies to test actions only",
	"-testRegion":                                     "applies to test actions only",
	"-maximum-concurrent-test-device-destinations":    "applies to test actions only",
	"-maximum-concurrent-test-simulator-destinations": "applies to test actions only",
}

var (
	// The build family: a step's -destination is a fallback the user's replaces, -arch
	// is repeatable.
	archiveSpec = actionSpec{
		name:       ActionArchive,
		rejected:   union(modeSwitchingFlags, testOnlyFlags, map[string]string{"-testPlan": "applies to test and build-for-testing only"}),
		defaults:   set("-destination"),
		appendable: set("-arch"),
	}
	buildSpec = actionSpec{
		name:       ActionBuild,
		rejected:   union(modeSwitchingFlags, testOnlyFlags, map[string]string{"-testPlan": "applies to test and build-for-testing only"}),
		defaults:   set("-destination"),
		appendable: set("-arch"),
	}
	analyzeSpec = actionSpec{
		name:       ActionAnalyze,
		rejected:   union(modeSwitchingFlags, testOnlyFlags, map[string]string{"-testPlan": "applies to test and build-for-testing only"}),
		defaults:   set("-destination", "-resultBundlePath"),
		appendable: set("-arch"),
	}
	// build-for-testing takes the test selection flags, which it bakes into the xctestrun.
	buildForTestingSpec = actionSpec{
		name:       ActionBuildForTesting,
		rejected:   modeSwitchingFlags,
		defaults:   set("-destination"),
		appendable: set("-arch"),
	}
	exportArchiveSpec = actionSpec{
		name:     "export archive",
		rejected: union(without(modeSwitchingFlags, "-exportArchive"), testOnlyFlags),
	}
	resolvePackagesSpec = actionSpec{
		name:     "resolve packages",
		rejected: union(without(modeSwitchingFlags, "-resolvePackageDependencies"), testOnlyFlags),
	}
	// test derives -skip-testing from quarantined tests and -destination from the simulator
	// input; the user's entries are added to both. -collect-test-diagnostics is a default.
	testSpec = actionSpec{
		name:       ActionTest,
		rejected:   modeSwitchingFlags,
		defaults:   set("-collect-test-diagnostics"),
		appendable: set("-skip-testing", "-destination", "-arch"),
	}
	showBuildSettingsSpec = actionSpec{
		name:     "show build settings",
		rejected: union(without(modeSwitchingFlags, "-showBuildSettings"), testOnlyFlags),
	}
)

// check reports the options the command refuses; build actions always are.
func (s actionSpec) check(opts Options) []Diagnostic {
	var diagnostics []Diagnostic
	for _, o := range opts {
		switch o.Kind {
		case Action:
			diagnostics = append(diagnostics, Diagnostic{Kind: ActionInOptions, Message: fmt.Sprintf("%q is a build action, and %s sets its own actions", o.Name, s.name)})
		case Switch, ValueOption, ColonOption:
			if reason, ok := s.rejected[o.Name]; ok {
				diagnostics = append(diagnostics, Diagnostic{Kind: RejectedOption, Message: fmt.Sprintf("%q is not valid for %s: %s", o.String(), s.name, reason)})
			}
		}
	}
	return diagnostics
}

func union(maps ...map[string]string) map[string]string {
	out := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

func without(m map[string]string, keys ...string) map[string]string {
	out := union(m)
	for _, k := range keys {
		delete(out, k)
	}
	return out
}

func set(keys ...string) map[string]bool {
	out := map[string]bool{}
	for _, k := range keys {
		out[k] = true
	}
	return out
}
