package xcodecommand

import (
	"fmt"
	"regexp"
	"strings"
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

// Option is one xcodebuild argument, parsed from user input or derived from params.
type Option struct {
	Kind  Kind
	Name  string
	Value string
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
	return strings.Join(o.render(), " ")
}

func (o Option) render() []string {
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
		args = append(args, o.render()...)
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
	// NAME=value, any case, split at the first "=" like xcodebuild.
	buildSettingPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
	flagNamePattern     = regexp.MustCompile(`^-[A-Za-z0-9][A-Za-z0-9_.-]*$`)
	upperCaseFlag       = regexp.MustCompile(`^-[A-Z][A-Z0-9_]*$`)
)

// freeFormValueFlags always take the next argument: their value may look like a build
// setting (-destination platform=iOS), an action (-scheme test) or a path named like one
// (-derivedDataPath build).
var freeFormValueFlags = map[string]bool{
	"-destination": true, "-scheme": true, "-target": true, "-configuration": true,
	"-testPlan": true, "-only-test-configuration": true, "-skip-test-configuration": true,
	"-project": true, "-workspace": true, "-xcconfig": true, "-archivePath": true,
	"-derivedDataPath": true, "-resultBundlePath": true, "-clonedSourcePackagesDirPath": true,
	"-packageCachePath": true, "-exportPath": true, "-exportOptionsPlist": true, "-xctestrun": true,
}

// ParseAdditionalOptions turns shell-split xcodebuild arguments into typed Options.
// Forms: -flag, -flag value, -flag:value, -key=value, NAME=value, build action. A flag
// takes the next argument as its value unless that is a flag, a setting or an action.
// Anything else is kept verbatim as Unknown and reported.
func ParseAdditionalOptions(args []string) (Options, []Diagnostic) {
	var opts Options
	var diagnostics []Diagnostic

	malformed := func(arg, why string) {
		opts = append(opts, Option{Kind: Unknown, Name: arg})
		diagnostics = append(diagnostics, Diagnostic{Kind: MalformedOption, Message: fmt.Sprintf("%q %s", arg, why)})
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case strings.TrimSpace(arg) == "":
			malformed(arg, "is empty")
		case strings.HasPrefix(arg, "-"):
			opt, consumed, why := parseFlag(args[i:])
			if why != "" {
				malformed(arg, why)
				continue
			}
			if opt.Kind == UserDefault && upperCaseFlag.MatchString(opt.Name) {
				diagnostics = append(diagnostics, Diagnostic{Kind: SuspiciousUserDefault, Message: fmt.Sprintf("%q looks like the build setting %s=%s written with a leading dash; xcodebuild accepts it as a user default and the setting never applies", arg, opt.Name[1:], opt.Value)})
			}
			opts = append(opts, opt)
			i += consumed - 1
		case buildSettingPattern.MatchString(arg):
			name, value, _ := strings.Cut(arg, "=")
			opts = append(opts, Option{Kind: BuildSetting, Name: name, Value: value})
		case knownActions[arg]:
			opts = append(opts, Option{Kind: Action, Name: arg})
		default:
			malformed(arg, "is not a -flag, -flag value, -flag:value, -key=value, NAME=value or a build action; xcodebuild treats it as an unknown build action")
		}
	}

	return opts, diagnostics
}

// parseFlag classifies args[0] (a "-" argument) and reports how many arguments it used;
// a non-empty why means it is malformed.
func parseFlag(args []string) (opt Option, consumed int, why string) {
	flag := args[0]
	if strings.ContainsAny(flag, " \t") {
		return Option{}, 0, "contains whitespace: quote only the value, not the flag and the value together"
	}

	// Whichever of "=" and ":" comes first decides the form (-only-testing:Suite/test=1, -IDEFoo=a:b).
	eq, colon := strings.Index(flag, "="), strings.Index(flag, ":")
	switch {
	case eq > 0 && (colon < 0 || eq < colon):
		name, value := flag[:eq], flag[eq+1:]
		if !flagNamePattern.MatchString(name) {
			return Option{}, 0, "is not a valid -key=value user default"
		}
		return Option{Kind: UserDefault, Name: name, Value: value}, 1, ""
	case colon > 0:
		name, value := flag[:colon], flag[colon+1:]
		if !flagNamePattern.MatchString(name) || value == "" {
			return Option{}, 0, "is not a valid -flag:value option"
		}
		return Option{Kind: ColonOption, Name: name, Value: value}, 1, ""
	}

	if !flagNamePattern.MatchString(flag) {
		return Option{}, 0, "is not a valid flag"
	}

	hasNext := len(args) > 1
	switch {
	case freeFormValueFlags[flag]:
		if !hasNext {
			return Option{}, 0, "requires a value"
		}
		return Option{Kind: ValueOption, Name: flag, Value: args[1]}, 2, ""
	case hasNext && looksLikeValue(args[1]):
		return Option{Kind: ValueOption, Name: flag, Value: args[1]}, 2, ""
	default:
		return Option{Kind: Switch, Name: flag}, 1, ""
	}
}

func looksLikeValue(arg string) bool {
	return strings.TrimSpace(arg) != "" &&
		!strings.HasPrefix(arg, "-") &&
		!buildSettingPattern.MatchString(arg) &&
		!knownActions[arg]
}
