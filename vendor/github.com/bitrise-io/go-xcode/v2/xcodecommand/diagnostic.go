package xcodecommand

// This file turns what the parser and the merge decided into messages for the step log.
// Nothing here changes the command line: deleting it, the second return of merge and
// Command.Diagnostics leaves the parser and the merge as they are.

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

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

// Diagnostic is one finding about the additional options, for the step log.
type Diagnostic struct {
	Kind    DiagnosticKind
	Message string
}

func (d Diagnostic) String() string {
	return d.Message
}

// Validation says what a constructor does with diagnostics.
type Validation int

// Validation modes.
const (
	// Warn (default) passes every option through and reports; this matches what the
	// steps did before and keeps working setups that trip a check working.
	Warn Validation = iota
	// Fail returns an error for the first non-informational diagnostic.
	Fail
)

func (k DiagnosticKind) informational() bool {
	return k == Override || k == RedundantOption
}

func firstFailure(diagnostics []Diagnostic) error {
	for _, d := range diagnostics {
		if !d.Kind.informational() {
			return fmt.Errorf("invalid additional option: %s", d.Message)
		}
	}
	return nil
}

// lint reports everything about a command's additional options that the user should see.
func lint(user, derived Options, collisions []collision, policy actionPolicy) []Diagnostic {
	var diagnostics []Diagnostic
	diagnostics = append(diagnostics, user.Diagnostics()...)
	diagnostics = append(diagnostics, lintPolicy(user, policy)...)
	diagnostics = append(diagnostics, lintStepInputs(derived, user)...)
	diagnostics = append(diagnostics, lintShadows(derived, user)...)
	diagnostics = append(diagnostics, lintCollisions(collisions)...)
	return diagnostics
}

// -ENABLE_BITCODE, -MARKETING_VERSION: a build setting typed with a dash, which
// xcodebuild accepts as a user default and silently ignores. Not -IDEFoo.
var upperCaseFlag = regexp.MustCompile(`^-[A-Z][A-Z0-9_]*$`)

// Diagnostics reports what the parser could not read and what xcodebuild would silently
// ignore, each with its fix. A step logs these before its first xcodebuild call.
func (opts Options) Diagnostics() []Diagnostic {
	var diagnostics []Diagnostic
	for _, o := range opts {
		switch {
		case o.Kind == Unknown && o.issue == quotedFlagWithValue:
			diagnostics = append(diagnostics, Diagnostic{Kind: SuspiciousUserDefault, Message: fmt.Sprintf("%q is a flag quoted together with its value. xcodebuild reads it as a user default and ignores it, so today's build runs without it. Remove it to keep that, or use %s to apply it.", o.Name, unquoteFlag(o.Name))})
		case o.Kind == Unknown:
			diagnostics = append(diagnostics, Diagnostic{Kind: MalformedOption, Message: fmt.Sprintf("%q %s", o.Name, malformedMessage(o))})
		case o.Kind == UserDefault && upperCaseFlag.MatchString(o.Name):
			// -ENABLE_BITCODE=NO: the user meant ENABLE_BITCODE=NO
			setting := o.Name[1:] + "=" + shellQuoted(o.Value)
			diagnostics = append(diagnostics, Diagnostic{Kind: SuspiciousUserDefault, Message: fmt.Sprintf("%q looks like the build setting %s with a leading dash. xcodebuild reads it as a user default and ignores it, so today's build runs without it. Remove it to keep that, or use %s to apply it.", o, setting, setting)})
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
		return "has no value after the colon. Add the value or remove the flag."
	case missingValue:
		return "has no value. Add one or remove the flag."
	default:
		return `is not a flag, a NAME=value build setting or a build action. xcodebuild treats it as an unknown build action. Quote a value with spaces, for example CODE_SIGN_IDENTITY="Apple Distribution", or remove it.`
	}
}

// lintPolicy reports the options the command refuses; build actions always are.
func lintPolicy(user Options, policy actionPolicy) []Diagnostic {
	var diagnostics []Diagnostic
	for _, o := range user {
		switch o.Kind {
		case Action:
			diagnostics = append(diagnostics, Diagnostic{Kind: ActionInOptions, Message: fmt.Sprintf("%q is a build action. The %s command sets its own actions. Remove it.", o.Name, policy.name)})
		default:
			if r, rejected := policy.rejects(o); rejected {
				diagnostics = append(diagnostics, Diagnostic{Kind: RejectedOption, Message: fmt.Sprintf("%q %s and is not valid for %s. Remove it.", o.String(), r.reason, policy.name)})
			}
		}
	}
	return diagnostics
}

// stepInputFlags only work together with what a step derives from its own inputs. Passed
// alone in the additional options they are reported, so the user reaches for the input.
var stepInputFlags = map[string]string{
	"-allowProvisioningUpdates": "cannot update provisioning profiles on its own. xcodebuild needs the App Store Connect API key flags, which the Step adds when its automatic code signing is enabled. Remove it and enable automatic code signing instead.",
}

// lintStepInputs reports user options from stepInputFlags that the command did not derive
// itself. When it did, the merge reports them as redundant instead.
func lintStepInputs(derived, user Options) []Diagnostic {
	var diagnostics []Diagnostic
	for _, o := range user {
		hint, ok := stepInputFlags[o.Key()]
		if !ok || slices.ContainsFunc(derived, func(d Option) bool { return d.Key() == o.Key() }) {
			continue
		}
		diagnostics = append(diagnostics, Diagnostic{Kind: PreferStepInput, Message: fmt.Sprintf("%q %s", o, hint)})
	}
	return diagnostics
}

// lintShadows reports a user "-flag=value" next to a derived "-flag value". It is not a
// collision (a user default keys as "-flag="), but xcodebuild ignores the "=" form, so the
// user's value never applies.
func lintShadows(derived, user Options) []Diagnostic {
	var diagnostics []Diagnostic
	for _, o := range derived {
		if o.Kind != ValueOption && o.Kind != Switch {
			continue
		}
		for _, u := range user {
			if u.Kind == UserDefault && u.Name == o.Name {
				diagnostics = append(diagnostics, Diagnostic{Kind: SuspiciousUserDefault, Message: fmt.Sprintf("%q is written with \"=\". xcodebuild reads it as a user default and ignores it, so today's build uses the Step's %q. Remove it to keep that, or use %s %s to apply it.", u, o, o.Name, shellQuoted(u.Value))})
				break
			}
		}
	}
	return diagnostics
}

// lintCollisions describes what the merge did with each derived option the user set too.
func lintCollisions(collisions []collision) []Diagnostic {
	var diagnostics []Diagnostic
	for _, c := range collisions {
		switch c.kind {
		case redundant:
			diagnostics = append(diagnostics, Diagnostic{Kind: RedundantOption, Message: fmt.Sprintf("%q is already set by the Step. Remove it.", c.derived)})
		case overridden:
			diagnostics = append(diagnostics, Diagnostic{Kind: Override, Message: fmt.Sprintf("%q replaces the Step's default %q.", c.user.join(), c.derived)})
		case repeated:
			diagnostics = append(diagnostics, Diagnostic{Kind: RepeatedOption, Message: fmt.Sprintf("%q repeats %q, which the Step sets. xcodebuild refuses a repeated option. Remove it, or change the Step input instead.", c.user.join(), c.derived)})
		}
	}
	return diagnostics
}

// join renders the options as one command line fragment, for messages.
func (opts Options) join() string {
	return strings.Join(opts.Args(), " ")
}

// shellQuoted renders a value as it has to be written in xcodebuild_options: the inverse
// of SplitAdditionalOptions.
func shellQuoted(value string) string {
	return shellquote.Join(value)
}

// unquoteFlag turns "-destination 'generic/platform=iOS'" (one argument) into the form the
// user meant: the flag, then the value quoted on its own.
func unquoteFlag(arg string) string {
	flag, value, _ := strings.Cut(arg, " ")
	value = strings.Trim(strings.TrimSpace(value), "'\"")
	return flag + " " + shellQuoted(value)
}
