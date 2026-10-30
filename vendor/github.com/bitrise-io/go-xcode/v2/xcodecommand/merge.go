package xcodecommand

import "slices"

// merge lays the user's options over the Step's: the user's come last, in the order given,
// and a Step option the user also set is dropped when it yields. What yields: a key the
// policy declares a default (the archive -destination a step invents), a build setting
// (xcodebuild takes the last value) and a switch (a repeat changes nothing). A value option
// such as -xcconfig stays, and xcodebuild refuses the repeat ("may only be provided once");
// a key the policy declares appendable keeps every copy, and xcodebuild takes them all.
// The user's options are marked as the user's on the way, for the lint.
func merge(step, user Options, policy actionPolicy) Options {
	merged := slices.Clone(step)
	for i, o := range merged {
		// "clean", a flag the user did not touch, or test's "-skip-testing:Flaky" from
		// quarantine next to the user's "-skip-testing:Manual": kept as is
		if o.Kind == Action || !user.sets(o.key()) || slices.Contains(policy.appendable, o.key()) {
			continue
		}
		// archive: "-destination generic/platform=iOS" + user "-destination generic/platform=tvOS" -> the user's
		// analyze: "CODE_SIGNING_ALLOWED=NO" + user "CODE_SIGNING_ALLOWED=YES" -> the user's
		// "-allowProvisioningUpdates" set by the step and again by the user -> the user's
		// "-xcconfig /tmp/temp.xcconfig" + user "-xcconfig mine.xcconfig" -> both, and xcodebuild fails on it
		merged[i].dropped = slices.Contains(policy.defaults, o.key()) || o.Kind == Switch || o.Kind == BuildSetting
	}
	for _, o := range user {
		o.user = true // the lint tells the user's copies from the Step's by this mark
		merged = append(merged, o)
	}
	return merged
}
