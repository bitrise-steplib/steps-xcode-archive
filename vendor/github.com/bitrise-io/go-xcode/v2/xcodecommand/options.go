package xcodecommand

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	shellquote "github.com/kballard/go-shellquote"
)

// Kind classifies one xcodebuild command line argument.
type Kind int

// The argument forms xcodebuild accepts, plus Unknown for what it does not.
const (
	Switch       Kind = iota // -quiet
	ValueOption              // -destination <spec>
	ColonOption              // -only-testing:<id>
	UserDefault              // -UseModernBuildSystem=YES
	BuildSetting             // CODE_SIGNING_ALLOWED=NO
	Action                   // clean, build, archive, ...
	Unknown                  // fits no form; Name holds it verbatim
)

// issue says why an argument was kept as Unknown; Options.Diagnostics turns it into a message.
type issue int

const (
	noIssue             issue = iota
	emptyArgument             // "" from an env var that expanded to nothing
	quotedFlagWithValue       // "-destination generic/platform=iOS": xcodebuild takes it as a user default
	quotedFlag                // "-sdk macosx": xcodebuild refuses it
	invalidFlag               // "-", "-=x"
	colonWithoutValue         // "-only-testing:"
	missingValue              // "-destination" as the last argument
	bareWord                  // "Distribution" from an unquoted CODE_SIGN_IDENTITY=Apple Distribution
)

// Option is one xcodebuild argument, parsed from user input or derived from params.
type Option struct {
	Kind  Kind
	Name  string
	Value string
	issue issue // why an Unknown option was not parsed
}

// Key identifies the option when merging: the flag, the build setting name or the
// action. A user default keys as "-name=", so -flag=value never collides with -flag value.
func (o Option) Key() string {
	if o.Kind == UserDefault {
		return o.Name + "="
	}
	return o.Name
}

// String renders the option as it appears on the command line.
func (o Option) String() string {
	return strings.Join(o.args(), " ")
}

// args is the option as xcodebuild arguments: one for a switch, an action or a joined
// form, two for a flag with a value.
func (o Option) args() []string {
	switch o.Kind {
	case ValueOption:
		return []string{o.Name, o.Value}
	case ColonOption:
		return []string{o.Name + ":" + o.Value}
	case UserDefault, BuildSetting:
		return []string{o.Name + "=" + o.Value}
	default:
		return []string{o.Name}
	}
}

// Options is an ordered list of xcodebuild arguments.
type Options []Option

// Args renders the options as xcodebuild arguments.
func (opts Options) Args() []string {
	var args []string
	for _, o := range opts {
		args = append(args, o.args()...)
	}
	return args
}

// Filter returns the options for which keep is true, in order.
func (opts Options) Filter(keep func(Option) bool) Options {
	var kept Options
	for _, o := range opts {
		if keep(o) {
			kept = append(kept, o)
		}
	}
	return kept
}

var (
	// CODE_SIGNING_ALLOWED=NO, my_setting=1, OTHER_LDFLAGS= (empty value); split at the
	// first "=" like xcodebuild, so OTHER_SWIFT_FLAGS=-D A=B keeps "-D A=B".
	// Not generic/platform=iOS: "/" is not a setting name, that is a -destination value.
	buildSettingPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
	// -quiet, -only-testing, -test-repetition-relaunch-enabled, -UseModernBuildSystem;
	// not "-", "-=x" or "-destination generic/platform=iOS" quoted as one argument.
	flagNamePattern = regexp.MustCompile(`^-[A-Za-z0-9][A-Za-z0-9_.-]*$`)
	// "-destination generic/platform=iOS" as one argument: a flag, a space, then a value
	// containing "=". xcodebuild reads the whole thing as a user default and ignores it.
	flagQuotedWithValue = regexp.MustCompile(`^-[A-Za-z0-9][A-Za-z0-9_.-]*\s+[^=:\s][^=]*=`)
)

// freeFormValueFlags always take the next argument: their value may look like a build
// setting (-destination platform=iOS), an action (-scheme test) or a path named like one
// (-derivedDataPath build).
//
// This is a hint, not a grammar. A flag missing here is still parsed by the lookahead,
// and the rendered arguments are the same either way; only the diagnostics and merge keys
// can differ. It does not need updating when xcodebuild gains flags.
var freeFormValueFlags = []string{
	"-destination", "-scheme", "-target", "-configuration",
	"-testPlan", "-only-test-configuration", "-skip-test-configuration",
	"-project", "-workspace", "-xcconfig", "-archivePath",
	"-derivedDataPath", "-resultBundlePath", "-clonedSourcePackagesDirPath",
	"-packageCachePath", "-exportPath", "-exportOptionsPlist", "-xctestrun",
}

// SplitAdditionalOptions splits the xcodebuild_options input into arguments with POSIX
// shell rules, as the steps always have: quotes group a value that contains spaces.
func SplitAdditionalOptions(input string) ([]string, error) {
	args, err := shellquote.Split(input)
	if err != nil {
		return nil, fmt.Errorf("xcodebuild_options %q cannot be split like a shell command line: %w", input, err)
	}
	return args, nil
}

// ParseAdditionalOptions turns shell-split xcodebuild arguments into typed Options.
// Whatever fits no form is kept verbatim as Unknown, so the command line stays as the
// user wrote it and xcodebuild refuses it with its own message; Options.Diagnostics
// says why.
func ParseAdditionalOptions(args []string) Options {
	var opts Options

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case strings.TrimSpace(arg) == "":
			// "" (an env var that expanded to nothing): xcodebuild sees an unknown build action
			opts = append(opts, Option{Kind: Unknown, Name: arg, issue: emptyArgument})
		case strings.HasPrefix(arg, "-") && flagQuotedWithValue.MatchString(arg):
			// "-destination generic/platform=iOS" as one argument: xcodebuild reads it as a user
			// default named "destination generic/platform" and ignores it; the step's own
			// -destination wins. Kept verbatim, as xcodebuild takes it.
			opts = append(opts, Option{Kind: Unknown, Name: arg, issue: quotedFlagWithValue})
		case strings.HasPrefix(arg, "-"):
			// -quiet | -destination id=SIM | -only-testing:AppTests | -UseModernBuildSystem=NO
			opt, consumed, why := parseFlag(args[i:])
			if why != noIssue {
				opts = append(opts, Option{Kind: Unknown, Name: arg, issue: why})
				continue
			}
			opts = append(opts, opt)
			i += consumed - 1 // a flag with a value used two arguments
		case buildSettingPattern.MatchString(arg):
			// CODE_SIGNING_ALLOWED=NO
			name, value, _ := strings.Cut(arg, "=")
			opts = append(opts, Option{Kind: BuildSetting, Name: name, Value: value})
		case slices.Contains(knownActions, arg):
			// clean
			opts = append(opts, Option{Kind: Action, Name: arg})
		default:
			// "Distribution" from an unquoted CODE_SIGN_IDENTITY=Apple Distribution
			opts = append(opts, Option{Kind: Unknown, Name: arg, issue: bareWord})
		}
	}

	return opts
}

// parseFlag classifies args[0] (a "-" argument) and reports how many arguments it used;
// an issue other than noIssue means it is malformed.
func parseFlag(args []string) (opt Option, consumed int, why issue) {
	flag := args[0]

	// The first of "=" and ":" splits name from value; values may contain spaces:
	//   -UseModernBuildSystem=NO                  -> name -UseModernBuildSystem, value NO
	//   -IDEFoo=a:b                               -> name -IDEFoo, value a:b
	//   -only-testing:Suite/test=1                -> name -only-testing, value Suite/test=1
	//   -only-testing:Pulley ManagerTests         -> name -only-testing, value Pulley ManagerTests
	//   -quiet                                    -> name -quiet, no value
	name, value := flag, ""
	if sep := strings.IndexAny(flag, "=:"); sep > 0 {
		name, value = flag[:sep], flag[sep+1:]
	}
	if strings.ContainsAny(name, " \t") {
		// "-sdk macosx" quoted as one argument: xcodebuild refuses it
		return Option{}, 0, quotedFlag
	}
	if !flagNamePattern.MatchString(name) {
		// "-", "-=x"
		return Option{}, 0, invalidFlag
	}

	switch {
	case len(name) < len(flag) && flag[len(name)] == '=':
		// -UseModernBuildSystem=NO (an empty value is fine: -IDEFoo=)
		return Option{Kind: UserDefault, Name: name, Value: value}, 1, noIssue
	case len(name) < len(flag):
		// -only-testing:AppTests; "-only-testing:" alone names no test
		if value == "" {
			return Option{}, 0, colonWithoutValue
		}
		return Option{Kind: ColonOption, Name: name, Value: value}, 1, noIssue
	case slices.Contains(freeFormValueFlags, flag):
		// -destination platform=iOS Simulator,name=iPhone 15: the next argument is the value
		// even though it looks like a build setting
		if len(args) < 2 {
			return Option{}, 0, missingValue
		}
		return Option{Kind: ValueOption, Name: flag, Value: args[1]}, 2, noIssue
	case len(args) > 1 && looksLikeValue(args[1]):
		// -packageAuthorizationProvider netrc: a bare word after a flag is its value
		return Option{Kind: ValueOption, Name: flag, Value: args[1]}, 2, noIssue
	default:
		// -quiet, -skipMacroValidation -quiet, -verbose ARCHS=arm64, -parallelizeTargets clean
		return Option{Kind: Switch, Name: flag}, 1, noIssue
	}
}

// looksLikeValue: "netrc", "/tmp/dd", "YES"; not "-quiet", not "ARCHS=arm64", not "clean".
func looksLikeValue(arg string) bool {
	return strings.TrimSpace(arg) != "" &&
		!strings.HasPrefix(arg, "-") &&
		!buildSettingPattern.MatchString(arg) &&
		!slices.Contains(knownActions, arg)
}
