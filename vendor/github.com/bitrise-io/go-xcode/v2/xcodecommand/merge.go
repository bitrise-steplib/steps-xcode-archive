package xcodecommand

import (
	"fmt"
	"slices"
)

// merge lays the user's options over the derived ones; the user's come last.
func merge(derived, user Options, policy actionPolicy) (Options, []Diagnostic) {
	// "-destination id=SIM -quiet" -> {"-destination": [-destination id=SIM], "-quiet": [-quiet]}
	// Actions and unparsed leftovers stay out: they never replace a derived flag.
	userByKey := map[string]Options{}
	for _, o := range user {
		if o.Kind != Action && o.Kind != Unknown {
			userByKey[o.Key()] = append(userByKey[o.Key()], o)
		}
	}

	var merged Options
	var diagnostics []Diagnostic
	report := func(kind DiagnosticKind, format string, args ...any) {
		diagnostics = append(diagnostics, Diagnostic{Kind: kind, Message: fmt.Sprintf(format, args...)})
	}

	for _, o := range derived {
		// derived "-collect-test-diagnostics never", user "-collect-test-diagnostics=on-failure":
		// not a collision (user defaults key as "-name="), but xcodebuild ignores the "=" form.
		if shadow, ok := userByKey[o.Key()+"="]; ok && (o.Kind == ValueOption || o.Kind == Switch) {
			report(SuspiciousUserDefault, "%q is written as %q in the additional options: xcodebuild reads the \"=\" form as a user default and ignores it; use \"%s value\"", o, shadow[0], o.Name)
		}

		conflicting, ok := userByKey[o.Key()]
		// What gives way to the user's copy: a policy default (the archive -destination a step
		// invents), a build setting (xcodebuild takes the last value), a switch (a repeat
		// changes nothing). A value option such as -xcconfig does not: xcodebuild refuses it.
		yields := slices.Contains(policy.defaults, o.Key()) || o.Kind == Switch || o.Kind == BuildSetting

		switch {
		case o.Kind == Action || !ok:
			// "clean", or a flag the user did not touch: kept as is
			merged = append(merged, o)
		case slices.Contains(policy.appendable, o.Key()):
			// test: "-skip-testing:Flaky" from quarantine + "-skip-testing:Manual" from the user -> both
			merged = append(merged, o)
		case yields && slices.Equal(conflicting.Args(), o.args()):
			// "-allowProvisioningUpdates" set by the step and again by the user -> once, with a note
			report(RedundantOption, "%q is already set by %s; it can be removed from the additional options", o, policy.name)
		case yields:
			// archive: "-destination generic/platform=iOS" + user "-destination generic/platform=tvOS" -> the user's
			// analyze: "CODE_SIGNING_ALLOWED=NO" + user "CODE_SIGNING_ALLOWED=YES" -> the user's
			report(Override, "%q replaced by additional option %s", o, conflicting)
		default:
			// "-xcconfig /tmp/temp.xcconfig" + user "-xcconfig mine.xcconfig" -> both, and xcodebuild fails on it
			merged = append(merged, o)
			report(RepeatedOption, "%q is set by %s and again as additional option %s; xcodebuild refuses a repeated option", o, policy.name, conflicting)
		}
	}

	return append(merged, user...), diagnostics
}
