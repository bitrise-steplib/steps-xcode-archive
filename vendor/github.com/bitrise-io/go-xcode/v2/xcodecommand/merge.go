package xcodecommand

import "slices"

// collision records what merge did with a derived option the user set too.
type collision struct {
	kind    collisionKind
	derived Option
	user    Options // the user's copies of the key, in order
}

type collisionKind int

const (
	redundant  collisionKind = iota // the derived copy is dropped; the user's is identical
	overridden                      // the derived copy is dropped for the user's
	repeated                        // both stay; xcodebuild refuses the repeat
)

// merge lays the user's options over the derived ones; the user's come last. The
// collisions say which derived options were dropped or left repeated, for the lint.
func merge(derived, user Options, policy actionPolicy) (Options, []collision) {
	// "-destination id=SIM -quiet" -> {"-destination": [-destination id=SIM], "-quiet": [-quiet]}
	// Actions and unparsed leftovers stay out: they never replace a derived flag.
	userByKey := map[string]Options{}
	for _, o := range user {
		if o.Kind != Action && o.Kind != Unknown {
			userByKey[o.Key()] = append(userByKey[o.Key()], o)
		}
	}

	var merged Options
	var collisions []collision

	for _, o := range derived {
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
			// "-allowProvisioningUpdates" set by the step and again by the user -> once
			collisions = append(collisions, collision{redundant, o, conflicting})
		case yields:
			// archive: "-destination generic/platform=iOS" + user "-destination generic/platform=tvOS" -> the user's
			// analyze: "CODE_SIGNING_ALLOWED=NO" + user "CODE_SIGNING_ALLOWED=YES" -> the user's
			collisions = append(collisions, collision{overridden, o, conflicting})
		default:
			// "-xcconfig /tmp/temp.xcconfig" + user "-xcconfig mine.xcconfig" -> both, and xcodebuild fails on it
			merged = append(merged, o)
			collisions = append(collisions, collision{repeated, o, conflicting})
		}
	}

	return append(merged, user...), collisions
}
