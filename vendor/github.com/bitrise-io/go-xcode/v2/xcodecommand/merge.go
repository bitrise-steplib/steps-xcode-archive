package xcodecommand

// placed is an option on an assembled command line: whose it is, and whether the merge
// left it off the line. Both are the merge's decisions, not the option's own.
type placed struct {
	Option
	user    bool // from xcodebuild_options; the zero value is the Step's own
	dropped bool // left off the command line: the user set it too, and the Step's copy yielded
}

// merged is an assembled command line: the Step's options, then the user's, in the order
// given. Its Args leaves the dropped ones out; an Options renders every one of its own.
type merged []placed

// options is the command line's options without their marks, for the rules that read
// options alone.
func (m merged) options() Options {
	opts := make(Options, len(m))
	for i, p := range m {
		opts[i] = p.Option
	}
	return opts
}

// Args renders the command line as xcodebuild arguments.
func (m merged) Args() []string {
	var args []string
	for _, p := range m {
		if !p.dropped {
			args = append(args, p.args()...)
		}
	}
	return args
}

// merge lays the user's options over the Step's: the user's come last, in the order given,
// and a Step option the user also set (Options.sets: a copy of the same form) is dropped
// when it yields. What yields: a Step default (appendDefault), a build setting (xcodebuild
// takes the last value) and a switch (a repeat changes nothing). Every other Step option
// stays next to the user's copy: a flag xcodebuild takes every copy of, such as test's
// -destination or -skip-testing, works with both, and a single-valued one such as
// -xcconfig makes xcodebuild refuse the line ("may only be provided once").
func merge(step, user Options) merged {
	var line merged
	for _, o := range step {
		p := placed{Option: o}
		// "clean", a flag the user did not touch, or test's "-skip-testing:Flaky" from
		// quarantine next to the user's "-skip-testing:Manual": kept as is
		if user.sets(o) {
			// archive: "-destination generic/platform=iOS" + user "-destination generic/platform=tvOS" -> the user's
			// analyze: "CODE_SIGNING_ALLOWED=NO" + user "CODE_SIGNING_ALLOWED=YES" -> the user's
			// "-allowProvisioningUpdates" set by the step and again by the user -> the user's
			// "-xcconfig /tmp/temp.xcconfig" + user "-xcconfig mine.xcconfig" -> both, and xcodebuild fails on it
			p.dropped = o.fallback || o.Kind == Switch || o.Kind == BuildSetting
		}
		line = append(line, p)
	}
	for _, o := range user {
		line = append(line, placed{Option: o, user: true})
	}
	return line
}
