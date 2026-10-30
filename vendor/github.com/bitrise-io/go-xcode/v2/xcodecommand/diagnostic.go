package xcodecommand

// This file turns what the parser and the merge decided into messages for the step log.
// Nothing here changes the command line.

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"

	shellquote "github.com/kballard/go-shellquote"
)

// DiagnosticKind classifies a finding about a command's additional options.
type DiagnosticKind int

// Diagnostic kinds.
const (
	MissingValue          DiagnosticKind = iota // a value flag with no value; xcodebuild takes the next flag as its value
	RejectedOption                              // a mode switch, or a flag of another action; xcodebuild ignores it here
	ActionInOptions                             // a build action; the command owns its action list
	SuspiciousUserDefault                       // a -name=value xcodebuild reads as a user default and ignores: a build setting with a dash, a flag quoted with its value, a flag written with "="
	PreferStepInput                             // a flag a Step input renders; use the input
	Override                                    // informational: a default or build setting yielded to the user's
	RedundantOption                             // informational: identical to what the command already sets
	LeftOut                                     // informational: options a preparatory command leaves out; the Step's main command gets them
)

// Diagnostic is one finding about the additional options, for the step log. It is
// informational when the command line does what the user meant: a note. Otherwise
// xcodebuild accepts the option and does something other than meant: a warning. What
// xcodebuild refuses is not reported; the build fails on it with xcodebuild's message.
type Diagnostic struct {
	Kind          DiagnosticKind
	Option        Option // the user's option the finding is about
	Message       string
	Informational bool
}

func (d Diagnostic) String() string {
	return d.Message
}

// report is a Diagnostic about o with a formatted message.
func report(kind DiagnosticKind, o Option, format string, args ...any) Diagnostic {
	informational := kind == Override || kind == RedundantOption || kind == LeftOut
	return Diagnostic{Kind: kind, Option: o, Message: fmt.Sprintf(format, args...), Informational: informational}
}

// -ENABLE_BITCODE, -MARKETING_VERSION: a build setting typed with a dash, which
// xcodebuild accepts as a user default and silently ignores. Not -IDEFoo.
var upperCaseFlag = regexp.MustCompile(`^-[A-Z][A-Z0-9_]*$`)

// Diagnostics reports what xcodebuild silently does with an option other than what the user
// meant, each with its fix. An argument xcodebuild refuses gets no line: the build fails on
// it with xcodebuild's own message. A step logs these when it reads its inputs, before its
// first xcodebuild call; a Command reports only its own findings.
func (opts Options) Diagnostics() []Diagnostic {
	var diagnostics []Diagnostic
	for _, o := range opts {
		switch {
		case o.Kind == UserDefault && strings.IndexFunc(o.Name, unicode.IsSpace) >= 0:
			// "-destination generic/platform=iOS" as one argument: a user default named
			// "destination generic/platform", which xcodebuild ignores
			diagnostics = append(diagnostics, report(SuspiciousUserDefault, o, "%q is a flag quoted together with its value. xcodebuild reads it as a user default and ignores it, so it has no effect. Use %s to apply it, or remove it.", o, unquoteFlag(o.String())))
		case o.tookFlag:
			// -derivedDataPath -quiet: xcodebuild takes -quiet as the path and the switch is lost
			diagnostics = append(diagnostics, report(MissingValue, o, "%q has no value, so xcodebuild takes %q as its value and that flag does not apply. Add a value, or remove it.", o.Name, o.Value))
		case o.Kind == UserDefault && upperCaseFlag.MatchString(o.Name):
			// -ENABLE_BITCODE=NO: the user meant ENABLE_BITCODE=NO
			setting := o.Name[1:] + "=" + shellQuoted(o.Value)
			diagnostics = append(diagnostics, report(SuspiciousUserDefault, o, "%q looks like the build setting %s with a leading dash. xcodebuild reads it as a user default and ignores it, so it has no effect. Use %s to apply it, or remove it.", o, setting, setting))
		}
	}
	return diagnostics
}

// lint reports what the command found in the user's options: per option, what its policy
// refuses; per flag, what became of the user's copies next to the Step's. The parser's
// findings are not repeated; Options.Diagnostics has them.
func lint(merged Options, policy actionPolicy) []Diagnostic {
	var diagnostics []Diagnostic
	for _, o := range merged {
		if d, refused := policy.refusal(o); o.user && refused {
			diagnostics = append(diagnostics, d)
		}
	}
	for _, g := range merged.byFlag() {
		diagnostics = append(diagnostics, g.findings(merged, policy)...)
	}
	return diagnostics
}

// refusal reports an option the command refuses: a build action, or a flag its policy
// rejects.
func (p actionPolicy) refusal(o Option) (Diagnostic, bool) {
	if o.Kind == Action {
		return report(ActionInOptions, o, "%q is a build action. The Step sets the actions itself. Remove it.", o.Name), true
	}
	if r, rejected := p.rejects(o); rejected {
		return report(RejectedOption, o, "%q does not work with %s. %s Remove it.", o, p.name, r.reason), true
	}
	return Diagnostic{}, false
}

// stepInput is a flag a Step input sets: passed in the additional options instead, it is
// reported so the user reaches for the input.
type stepInput struct {
	hint          string
	commands      []string // the commands whose Steps have the input; empty means all
	satisfied     []string // value flags that, all set by the user too, make the flag work without the input
	informational bool     // the flag works as given; the input is only the tidier place for it
}

// appliesTo says whether the command's Step has the input.
func (in stepInput) appliesTo(command string) bool {
	return len(in.commands) == 0 || slices.Contains(in.commands, command)
}

var testRunCommands = []string{ActionTest, ActionTestWithoutBuilding}

const testRepetitionHint = "belongs to the Step's test repetition inputs (test_repetition_mode, maximum_test_repetitions, relaunch_tests_for_each_repetition). Remove it and set those inputs instead."

var stepInputFlags = map[string]stepInput{
	"-allowProvisioningUpdates": {
		hint:      "cannot update provisioning profiles on its own. xcodebuild needs the App Store Connect API key flags, which the Step adds when its automatic code signing is enabled. Remove it and enable automatic code signing instead. See https://docs.bitrise.io/en/code-signing/ios-code-signing/managing-ios-code-signing-files---automatic-provisioning.html",
		satisfied: []string{"-authenticationKeyPath", "-authenticationKeyID", "-authenticationKeyIssuerID"},
	},
	// xcode-test and xcode-test-without-building render these from their inputs. Passed in
	// the options they work the same, so the hint is a note.
	"-test-iterations":                  {hint: testRepetitionHint, commands: testRunCommands, informational: true},
	"-retry-tests-on-failure":           {hint: testRepetitionHint, commands: testRunCommands, informational: true},
	"-run-tests-until-failure":          {hint: testRepetitionHint, commands: testRunCommands, informational: true},
	"-test-repetition-relaunch-enabled": {hint: testRepetitionHint, commands: testRunCommands, informational: true},
}

// setsAll says whether opts gives every one of the value flags a value. It is false for
// none: a stepInput without satisfied flags always needs its input.
func setsAll(opts Options, flags []string) bool {
	if len(flags) == 0 {
		return false
	}
	for _, flag := range flags {
		if !slices.ContainsFunc(opts, func(o Option) bool { return o.Kind == ValueOption && o.Name == flag && o.Value != "" }) {
			return false
		}
	}
	return true
}

// flagGroup is every option on the command line that names one flag or build setting, in
// every form the user can write it: -quiet, -quiet YES, -quiet:YES and -quiet=YES all
// belong to "-quiet". step holds the Step's copies, dropped or not, user the user's.
type flagGroup struct {
	name       string
	step, user Options
}

// byFlag groups the flags and build settings by name, in order of first appearance.
// Actions and unparsed arguments have no group.
func (opts Options) byFlag() []flagGroup {
	var groups []flagGroup
	at := map[string]int{}
	for _, o := range opts {
		if o.Kind == Action || o.Kind == Unknown {
			continue
		}
		i, ok := at[o.Name]
		if !ok {
			i = len(groups)
			at[o.Name] = i
			groups = append(groups, flagGroup{name: o.Name})
		}
		if o.user {
			groups[i].user = append(groups[i].user, o)
		} else {
			groups[i].step = append(groups[i].step, o)
		}
	}
	return groups
}

// ignoredUserDefault opens the message about a "-flag=value" the user meant as "-flag value".
const ignoredUserDefault = "%q is written with \"=\". xcodebuild reads it as a user default and ignores it, "

// findings says what became of a flag the user set: what the merge did with the Step's
// copy, then each "-flag=value" form of it, then whether a Step input should set it
// instead. A copy xcodebuild refuses gets no line, since the build fails on it with
// xcodebuild's own message: a repeated value option ("may only be provided once"), a
// value given to a switch as "-quiet YES" (an unknown build action) or "-quiet:YES" (an
// invalid option). Every fact is in the group, so no rule needs another pass.
func (g flagGroup) findings(all Options, policy actionPolicy) []Diagnostic {
	if len(g.user) == 0 {
		return nil
	}
	var diagnostics []Diagnostic

	// The Step's copy yielded to the user's copies of the same form: identical ones are
	// redundant, a different one overrides it. A copy of another form ("-quiet YES" next to
	// the Step's -quiet) is one xcodebuild refuses.
	if len(g.step) > 0 && g.step[0].dropped {
		step := g.step[0]
		same := g.user.Filter(func(o Option) bool { return o.Kind == step.Kind && o.key() == step.key() })
		switch {
		case len(same) == 0:
		case same.allEqual(step):
			diagnostics = append(diagnostics, report(RedundantOption, same[0], "%q is already set by the Step. Remove it.", step))
		default:
			diagnostics = append(diagnostics, report(Override, same[0], "%q replaces the Step's default %q.", same.join(), step))
		}
	}

	// "-flag=value": xcodebuild reads it as a user default and ignores it. A switch in the
	// group, the Step's or the user's, says the flag takes no value; otherwise the message
	// names the value the build uses instead.
	sw, isSwitch := g.all().last(func(o Option) bool { return o.Kind == Switch })
	applied, found := g.all().last(func(o Option) bool { return o.Kind == ValueOption && !o.dropped })
	for _, u := range g.user.Filter(func(o Option) bool { return o.Kind == UserDefault }) {
		switch {
		case isSwitch:
			// -allowProvisioningUpdates=YES next to the Step's -allowProvisioningUpdates; -quiet -quiet=YES
			diagnostics = append(diagnostics, report(SuspiciousUserDefault, u, "%q gives a value to %s, which takes none, so xcodebuild reads it as a user default and ignores it. %s. Remove it.", u, sw.Name, alreadySet(sw)))
		case found && applied.user:
			// -destination generic/platform=tvOS -destination=generic/platform=watchOS
			diagnostics = append(diagnostics, report(SuspiciousUserDefault, u, ignoredUserDefault+"and uses your %q instead. Change that one to %s %s to apply it, or remove it.", u, applied, u.Name, shellQuoted(u.Value)))
		case found:
			diagnostics = append(diagnostics, report(SuspiciousUserDefault, u, ignoredUserDefault+"and uses the Step's %q instead. Use %s %s to apply it, or remove it.", u, applied, u.Name, shellQuoted(u.Value)))
		case slices.Contains(freeFormValueFlags, u.Name):
			// -destination=generic/platform=iOS with no -destination anywhere
			diagnostics = append(diagnostics, report(SuspiciousUserDefault, u, ignoredUserDefault+"so it has no effect. Use %s %s to apply it, or remove it.", u, u.Name, shellQuoted(u.Value)))
		}
	}

	// A flag a Step input renders, passed here instead: -allowProvisioningUpdates without the
	// Step's API key flags, or the user's own.
	if in, ok := stepInputFlags[g.name]; ok && in.appliesTo(policy.name) && len(g.step) == 0 && !setsAll(all, in.satisfied) {
		for _, u := range g.user {
			if u.key() == g.name {
				d := report(PreferStepInput, u, "%q %s", u, in.hint)
				d.Informational = in.informational
				diagnostics = append(diagnostics, d)
			}
		}
	}
	return diagnostics
}

// alreadySet says who set the switch sw, for a message about a value given to it.
func alreadySet(sw Option) string {
	if sw.user {
		return fmt.Sprintf("%s is already set", sw)
	}
	return fmt.Sprintf("The Step already sets %s", sw)
}

// all is the group in command line order: the Step's copies, then the user's.
func (g flagGroup) all() Options {
	return append(slices.Clone(g.step), g.user...)
}

// last is the last option for which keep is true.
func (opts Options) last(keep func(Option) bool) (Option, bool) {
	for i := len(opts) - 1; i >= 0; i-- {
		if keep(opts[i]) {
			return opts[i], true
		}
	}
	return Option{}, false
}

// allEqual says whether every option renders as o does.
func (opts Options) allEqual(o Option) bool {
	return !slices.ContainsFunc(opts, func(u Option) bool { return !slices.Equal(u.args(), o.args()) })
}

// join renders the options as one command line fragment, for messages.
func (opts Options) join() string {
	return strings.Join(opts.Args(), " ")
}

// quoted renders each option quoted, comma-separated, for a message about several.
func (opts Options) quoted() string {
	var parts []string
	for _, o := range opts {
		parts = append(parts, fmt.Sprintf("%q", o.String()))
	}
	return strings.Join(parts, ", ")
}

// shellQuoted renders a value as it has to be written in xcodebuild_options: the inverse
// of the split in ParseOptions.
func shellQuoted(value string) string {
	return shellquote.Join(value)
}

// unquoteFlag turns "-destination 'generic/platform=iOS'" (one argument) into the form the
// user meant: the flag, then the value quoted on its own.
func unquoteFlag(arg string) string {
	flag, value := arg, ""
	if i := strings.IndexFunc(arg, unicode.IsSpace); i >= 0 {
		// "-sdk\tmacosx" splits like "-sdk macosx": the parser takes any whitespace
		flag, value = arg[:i], arg[i:]
	}
	value = strings.Trim(strings.TrimSpace(value), "'\"")
	return flag + " " + shellQuoted(value)
}
