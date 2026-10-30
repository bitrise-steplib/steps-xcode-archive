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
	MalformedOption       DiagnosticKind = iota // unclassifiable argument, passed through; xcodebuild rejects it
	RejectedOption                              // switches xcodebuild's mode or belongs to another action
	ActionInOptions                             // a build action; the command owns its action list
	SuspiciousUserDefault                       // -NAME=value that looks like a build setting; xcodebuild ignores it silently
	RepeatedOption                              // a value option set by the command and again by the user; xcodebuild refuses it
	PreferStepInput                             // a flag that only works with what the step's inputs provide; use the input
	Override                                    // informational: a default or build setting yielded to the user's
	RedundantOption                             // informational: identical to what the command already sets
)

// Informational says whether the kind is a note rather than a problem: the command line
// does what the user meant. A step logs the rest as warnings, or fails on them.
func (k DiagnosticKind) Informational() bool {
	return k == Override || k == RedundantOption
}

// Diagnostic is one finding about the additional options, for the step log.
type Diagnostic struct {
	Kind    DiagnosticKind
	Option  Option // the user's option the finding is about
	Message string
}

func (d Diagnostic) String() string {
	return d.Message
}

// report is a Diagnostic about o with a formatted message.
func report(kind DiagnosticKind, o Option, format string, args ...any) Diagnostic {
	return Diagnostic{Kind: kind, Option: o, Message: fmt.Sprintf(format, args...)}
}

// -ENABLE_BITCODE, -MARKETING_VERSION: a build setting typed with a dash, which
// xcodebuild accepts as a user default and silently ignores. Not -IDEFoo.
var upperCaseFlag = regexp.MustCompile(`^-[A-Z][A-Z0-9_]*$`)

// Diagnostics reports what the parser could not read and what xcodebuild would silently
// ignore, each with its fix. A step logs these when it reads its inputs, before its first
// xcodebuild call; a Command reports only its own findings.
func (opts Options) Diagnostics() []Diagnostic {
	var diagnostics []Diagnostic
	for _, o := range opts {
		switch {
		case o.Kind == Unknown && o.issue == quotedFlagWithValue:
			diagnostics = append(diagnostics, report(SuspiciousUserDefault, o, "%q is a flag quoted together with its value. xcodebuild reads it as a user default and ignores it, so it has no effect. Use %s to apply it, or remove it.", o.Name, unquoteFlag(o.Name)))
		case o.Kind == Unknown:
			diagnostics = append(diagnostics, report(MalformedOption, o, "%q %s", o.Name, malformedMessage(o)))
		case o.Kind == UserDefault && upperCaseFlag.MatchString(o.Name):
			// -ENABLE_BITCODE=NO: the user meant ENABLE_BITCODE=NO
			setting := o.Name[1:] + "=" + shellQuoted(o.Value)
			diagnostics = append(diagnostics, report(SuspiciousUserDefault, o, "%q looks like the build setting %s with a leading dash. xcodebuild reads it as a user default and ignores it, so it has no effect. Use %s to apply it, or remove it.", o, setting, setting))
		}
	}
	return diagnostics
}

func malformedMessage(o Option) string {
	switch o.issue {
	case emptyArgument:
		return "is empty. xcodebuild treats it as an unknown build action. Remove it."
	case quotedFlag:
		return fmt.Sprintf("is a flag quoted together with its value. xcodebuild refuses it. Use %s instead.", unquoteFlag(o.Name))
	case invalidFlag:
		return "is not a valid flag. xcodebuild refuses it. Remove it."
	case colonWithoutValue:
		return "has no value after the colon. xcodebuild refuses it. Add the value, or remove it."
	case missingValue:
		return "has no value. xcodebuild refuses it. Add one, or remove it."
	default:
		return `is not a flag, a NAME=value build setting or a build action. xcodebuild treats it as an unknown build action. Quote a value with spaces, for example CODE_SIGN_IDENTITY="Apple Distribution", or remove it.`
	}
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
	hint      string
	commands  []string // the commands whose Steps have the input; empty means all
	satisfied []string // value flags that, all set by the user too, make the flag work without the input
}

// appliesTo says whether the command's Step has the input.
func (in stepInput) appliesTo(command string) bool {
	return len(in.commands) == 0 || slices.Contains(in.commands, command)
}

var testRunCommands = []string{ActionTest, ActionTestWithoutBuilding}

const testRepetitionHint = "belongs to the Step's test repetition inputs (test_repetition_mode, maximum_test_repetitions, relaunch_tests_for_each_repetition). Remove it and set those inputs instead."

var stepInputFlags = map[string]stepInput{
	"-allowProvisioningUpdates": {
		hint:      "cannot update provisioning profiles on its own. xcodebuild needs the App Store Connect API key flags, which the Step adds when its automatic code signing is enabled. Remove it and enable automatic code signing instead.",
		satisfied: []string{"-authenticationKeyPath", "-authenticationKeyID", "-authenticationKeyIssuerID"},
	},
	// xcode-test and xcode-test-without-building render these from their inputs.
	"-test-iterations":                  {hint: testRepetitionHint, commands: testRunCommands},
	"-retry-tests-on-failure":           {hint: testRepetitionHint, commands: testRunCommands},
	"-run-tests-until-failure":          {hint: testRepetitionHint, commands: testRunCommands},
	"-test-repetition-relaunch-enabled": {hint: testRepetitionHint, commands: testRunCommands},
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
// copy, then each "-flag=value" the flag shadows, then whether a Step input should set it
// instead. Every fact is in the group, so no rule needs another pass.
func (g flagGroup) findings(all Options, policy actionPolicy) []Diagnostic {
	if len(g.user) == 0 {
		return nil
	}
	var diagnostics []Diagnostic

	// same: the user's copies that key as the Step's; a "-flag=value" keys apart and shadows
	var step Option
	var same Options
	if len(g.step) > 0 {
		step = g.step[0]
		same = g.user.Filter(func(o Option) bool { return o.key() == step.key() })
	}
	// the switch on the command line, the user's own or the Step's, for a value given to it
	sw, hasSwitch := g.all().last(func(o Option) bool { return o.Kind == Switch })
	switch {
	case hasSwitch && slices.ContainsFunc(g.user, valued):
		// "-allowProvisioningUpdates YES", "-allowProvisioningUpdates:YES", "-allowProvisioningUpdates=YES"
		for _, u := range g.user.Filter(valued) {
			diagnostics = append(diagnostics, valuedSwitch(u, sw))
		}
	case len(same) > 0 && step.dropped && same.allEqual(step):
		diagnostics = append(diagnostics, report(RedundantOption, same[0], "%q is already set by the Step. Remove it.", step))
	case len(same) > 0 && step.dropped:
		diagnostics = append(diagnostics, report(Override, same[0], "%q replaces the Step's default %q.", same.join(), step))
	case len(same) > 0 && !slices.Contains(policy.appendable, g.name):
		diagnostics = append(diagnostics, report(RepeatedOption, same[0], "%s is managed by the Step. %q conflicts with its %q: xcodebuild accepts the flag only once. Remove it.", g.name, same.join(), step))
	}

	if !hasSwitch {
		// "-destination=x": xcodebuild ignores the "=" form; the message names what the build uses instead
		applied, found := g.all().last(func(o Option) bool { return o.Kind == ValueOption && !o.dropped })
		for _, u := range g.user.Filter(func(o Option) bool { return o.Kind == UserDefault }) {
			switch {
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
	}

	if in, ok := stepInputFlags[g.name]; ok && in.appliesTo(policy.name) && len(g.step) == 0 && !setsAll(all, in.satisfied) {
		// -allowProvisioningUpdates without the Step's API key flags, or the user's own
		for _, u := range g.user {
			if u.key() == g.name {
				diagnostics = append(diagnostics, report(PreferStepInput, u, "%q %s", u, in.hint))
			}
		}
	}
	return diagnostics
}

// valuedSwitch reports u, a value given to sw, a switch the command line has. How
// xcodebuild reads the value depends on how it is written.
func valuedSwitch(u, sw Option) Diagnostic {
	kind, effect := MalformedOption, fmt.Sprintf("xcodebuild reads %q as a build action and fails", u.Value)
	switch u.Kind {
	case UserDefault:
		// -allowProvisioningUpdates=YES
		kind, effect = SuspiciousUserDefault, "xcodebuild reads it as a user default and ignores it"
	case ColonOption:
		// -allowProvisioningUpdates:YES
		effect = "xcodebuild refuses it as an invalid option"
	}
	set := fmt.Sprintf("The Step already sets %s", sw)
	if sw.user {
		set = fmt.Sprintf("%s is already set", sw)
	}
	return report(kind, u, "%q gives a value to %s, which takes none, so %s. %s. Remove it.", u, sw, effect, set)
}

func valued(o Option) bool { return o.Kind != Switch }

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
