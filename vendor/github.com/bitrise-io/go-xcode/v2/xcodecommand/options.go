package xcodecommand

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"

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
	Unknown                  // fits no form; Name holds it verbatim, and xcodebuild fails on it with its own message
)

// Option is one xcodebuild argument, parsed from user input or derived from params.
type Option struct {
	Kind     Kind
	Name     string
	Value    string
	tookFlag bool // a value flag that took the next flag as its value, as xcodebuild does
	user     bool // from xcodebuild_options, marked by the merge; the zero value is the Step's own
	dropped  bool // the merge left it off the command line: the user set it too
}

// key identifies the option when merging: the flag, the build setting name or the
// action. A user default keys as "-name=", so -flag=value never collides with -flag value.
func (o Option) key() string {
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

// Args renders the options as xcodebuild arguments, without the ones the merge dropped.
func (opts Options) Args() []string {
	var args []string
	for _, o := range opts {
		if !o.dropped {
			args = append(args, o.args()...)
		}
	}
	return args
}

// sets says whether a flag or build setting with the key is among the options. Actions
// and unparsed arguments never replace a derived flag.
func (opts Options) sets(key string) bool {
	return slices.ContainsFunc(opts, func(o Option) bool { return o.Kind != Action && o.Kind != Unknown && o.key() == key })
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
	// not "-" or "-:x", nor "-sdk macosx" quoted as one argument.
	flagNamePattern = regexp.MustCompile(`^-[A-Za-z0-9][A-Za-z0-9_.-]*$`)
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
	"-packageCachePath", "-exportPath", "-exportOptionsPlist", "-xctestrun", "-testProductsPath",
	"-resultStreamPath", "-localizationPath", "-importPath", "-codesizeProfileOutputDir", "-test-enumeration-output-path",
}

// ParseOptions reads the xcodebuild_options input. It splits the input with POSIX shell
// rules, as the steps always have (quotes group a value that contains spaces), and reads
// the arguments as xcodebuild does. Only a quoting error fails: whatever fits no form is
// kept verbatim as Unknown, so the command line stays as the user wrote it and xcodebuild
// refuses it with its own message.
func ParseOptions(input string) (Options, error) {
	args, err := shellquote.Split(input)
	if err != nil {
		return nil, fmt.Errorf("xcodebuild_options %q cannot be split like a shell command line: %w", input, err)
	}
	return parse(args), nil
}

// parse turns shell-split arguments into typed Options. Whatever fits no form is kept
// verbatim as Unknown, so the command line stays as the user wrote it.
func parse(args []string) Options {
	var opts Options
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case strings.HasPrefix(arg, "-"):
			// -quiet | -destination id=SIM | -only-testing:AppTests | -UseModernBuildSystem=NO
			opt, consumed := parseFlag(args[i:])
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
			// "" from an env var that expanded to nothing, "Distribution" from an unquoted
			// CODE_SIGN_IDENTITY=Apple Distribution: xcodebuild fails on an unknown build action
			opts = append(opts, Option{Kind: Unknown, Name: arg})
		}
	}

	return opts
}

// parseFlag classifies args[0] (a "-" argument) and reports how many arguments it used.
// What xcodebuild refuses comes back as Unknown.
func parseFlag(args []string) (Option, int) {
	flag := args[0]
	unknown := Option{Kind: Unknown, Name: flag}

	// The first of "=" and ":" splits name from value; values may contain spaces:
	//   -UseModernBuildSystem=NO                  -> name -UseModernBuildSystem, value NO
	//   -IDEFoo=a:b                               -> name -IDEFoo, value a:b
	//   -only-testing:Suite/test=1                -> name -only-testing, value Suite/test=1
	//   -only-testing:Pulley ManagerTests         -> name -only-testing, value Pulley ManagerTests
	//   -quiet                                    -> name -quiet, no value
	name, sep, value := flag, byte(0), ""
	if i := strings.IndexAny(flag, "=:"); i > 0 {
		name, sep, value = flag[:i], flag[i], flag[i+1:]
	}

	switch {
	case sep == '=':
		// -UseModernBuildSystem=NO (an empty value is fine: -IDEFoo=). "-destination
		// generic/platform=iOS" quoted as one argument lands here too: xcodebuild reads it as
		// a user default named "destination generic/platform" and ignores it.
		return Option{Kind: UserDefault, Name: name, Value: value}, 1
	case strings.IndexFunc(name, unicode.IsSpace) >= 0 || !flagNamePattern.MatchString(name):
		// "-sdk macosx" quoted as one argument, "-", "-:x": xcodebuild refuses it
		return unknown, 1
	case sep == ':':
		// -only-testing:AppTests; "-only-testing:" alone names no test, and xcodebuild refuses it
		if value == "" {
			return unknown, 1
		}
		return Option{Kind: ColonOption, Name: name, Value: value}, 1
	case slices.Contains(freeFormValueFlags, flag):
		// -destination platform=iOS Simulator,name=iPhone 15: the next argument is the value
		// even though it looks like a build setting. A flag is taken as the value too, as
		// xcodebuild does ("-derivedDataPath -quiet" writes derived data to ./-quiet and
		// drops the switch). Only -destination checks its argument and refuses a flag, and
		// every flag refuses to be the last argument ("requires an argument"). Verified on
		// Xcode 26.5 and 27.0.
		switch {
		case len(args) < 2 || (flag == "-destination" && strings.HasPrefix(args[1], "-")):
			return unknown, 1
		case strings.HasPrefix(args[1], "-"):
			return Option{Kind: ValueOption, Name: flag, Value: args[1], tookFlag: true}, 2
		}
		return Option{Kind: ValueOption, Name: flag, Value: args[1]}, 2
	case len(args) > 1 && looksLikeValue(args[1]):
		// -packageAuthorizationProvider netrc: a bare word after a flag is its value
		return Option{Kind: ValueOption, Name: flag, Value: args[1]}, 2
	default:
		// -quiet, -skipMacroValidation -quiet, -verbose ARCHS=arm64, -parallelizeTargets clean
		return Option{Kind: Switch, Name: flag}, 1
	}
}

// looksLikeValue: "netrc", "/tmp/dd", "YES"; not "-quiet", not "ARCHS=arm64", not "clean".
func looksLikeValue(arg string) bool {
	return strings.TrimSpace(arg) != "" &&
		!strings.HasPrefix(arg, "-") &&
		!buildSettingPattern.MatchString(arg) &&
		!slices.Contains(knownActions, arg)
}
