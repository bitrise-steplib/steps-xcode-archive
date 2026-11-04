package xcodecommand

import (
	"path/filepath"
	"slices"

	"github.com/bitrise-io/go-utils/v2/command"
)

const (
	xcworkspaceExtension = ".xcworkspace"
	xcodeprojExtension   = ".xcodeproj"
)

// Command is an assembled xcodebuild invocation.
type Command struct {
	args []string
}

// Args returns the xcodebuild arguments without the tool name, ready for Runner.Run.
func (c Command) Args() []string {
	return slices.Clone(c.args)
}

// Create returns a runnable command; its PrintableCommandArgs is the line to log.
func (c Command) Create(factory command.Factory, opts *command.Opts) command.Command {
	return factory.Create(toolName, c.Args(), opts)
}

// assemble merges the step's additional options over the derived flags.
func assemble(derived, user Options) (Command, error) {
	return Command{args: merge(derived, user).Args()}, nil
}

// projectOptions are the flags every scheme-building action shares.
type projectOptions struct {
	projectPath   string
	scheme        string
	configuration string
	destination   string
	xcconfigPath  string
	sdk           string
}

func (p projectOptions) options() Options {
	opts := containerOptions(p.projectPath)
	opts = appendValue(opts, "-scheme", p.scheme)
	opts = appendValue(opts, "-configuration", p.configuration)
	opts = appendDefault(opts, "-destination", p.destination)
	opts = appendValue(opts, "-xcconfig", p.xcconfigPath)
	opts = appendValue(opts, "-sdk", p.sdk)
	return opts
}

func actions(clean bool, action string) Options {
	var opts Options
	if clean {
		opts = append(opts, Option{Kind: Action, Name: ActionClean})
	}
	return append(opts, Option{Kind: Action, Name: action})
}

// containerOptions is -workspace for an .xcworkspace and -project for an .xcodeproj. A
// Swift package (Package.swift or its directory) gets no flag; run xcodebuild in it.
func containerOptions(path string) Options {
	switch filepath.Ext(filepath.Clean(path)) {
	case xcworkspaceExtension:
		return Options{{Kind: ValueOption, Name: "-workspace", Value: path}}
	case xcodeprojExtension:
		return Options{{Kind: ValueOption, Name: "-project", Value: path}}
	default:
		return nil
	}
}

func appendValue(opts Options, flag, value string) Options {
	if value == "" {
		return opts
	}
	return append(opts, Option{Kind: ValueOption, Name: flag, Value: value})
}

// appendDefault adds a value flag the Step sets on its own, which a copy in the additional
// options replaces: the build family's -destination, as steps-xcode-archive dropped its
// own for the user's. Any other derived flag stays next to the user's copy; see merge.
func appendDefault(opts Options, flag, value string) Options {
	if value == "" {
		return opts
	}
	return append(opts, Option{Kind: ValueOption, Name: flag, Value: value, fallback: true})
}

func appendAuthentication(opts Options, auth *Authentication) Options {
	if auth == nil {
		return opts
	}
	return append(opts, auth.options()...)
}
