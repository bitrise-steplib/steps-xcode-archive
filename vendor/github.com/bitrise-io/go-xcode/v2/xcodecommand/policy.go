package xcodecommand

import "slices"

// actionPolicy is a command's policy for additional options; anything not listed passes.
type actionPolicy struct {
	name       string
	rejections []rejection
	defaults   []string // derived keys the user's option replaces
	appendable []string // repeatable keys where derived and user entries both stay
}

// rejection is a set of flags a command refuses for one shared reason. An option naming
// any one of the flags is rejected; the reason is what its message says.
type rejection struct {
	flags  []string // any one of these
	reason string
}

// rejects says whether the command refuses the flag option o, that is whether o names a
// flag of one of its rejections, and returns that rejection for its reason.
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
	// than the command: -showBuildSettings prints the settings and builds nothing, -list
	// lists the schemes, and so on. xcodebuild accepts each next to a build action.
	modeSwitching = rejection{
		reason: "It switches xcodebuild to another task instead.",
		flags: []string{
			"-exportArchive", "-exportNotarizedApp", "-showBuildSettings", "-showBuildSettingsForIndex",
			"-resolvePackageDependencies", "-list", "-version", "-showsdks", "-showdestinations", "-showTestPlans",
			"-create-xcframework", "-exportLocalizations", "-importLocalizations", "-find-executable", "-find-library",
			"-convert-project", "-usage", "-help", "-license", "-checkFirstLaunchStatus", "-runFirstLaunch",
			"-downloadPlatform", "-downloadAllPlatforms", "-importPlatform", "-prepareDeviceSupport",
			"-downloadComponent", "-importComponent", "-deleteComponent", "-showComponent",
		},
	}
	// -json next to a build action builds, but the build log is gone: of 811 lines only the
	// 16 compiler diagnostics remain (Xcode 26.5 and 27.0). On -showBuildSettings it turns
	// the output into JSON, which the Steps' text-parsing settings readers cannot read.
	silentLog = rejection{reason: "It turns off xcodebuild's build log: only compiler diagnostics are printed.", flags: []string{"-json"}}
	// Test flags xcodebuild accepts and ignores outside a test action (verified on archive,
	// Xcode 26.5 and 27.0). The test flags it refuses there (-testPlan, -enableCodeCoverage,
	// -testLanguage, -testRegion, -testProductsPath: "only supported when testing"; -xctestrun
	// next to -scheme, and -scheme, -project, -workspace or -testProductsPath next to
	// -xctestrun) need no line: the build fails on them with xcodebuild's own message.
	testOnly = rejection{
		reason: "It applies to test actions only, and xcodebuild ignores it here.",
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
)
