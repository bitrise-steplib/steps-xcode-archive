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
	args        []string
	diagnostics []Diagnostic
}

// Args returns the xcodebuild arguments without the tool name, ready for Runner.Run.
func (c Command) Args() []string {
	return slices.Clone(c.args)
}

// Diagnostics lists what the constructor found in the additional options.
func (c Command) Diagnostics() []Diagnostic {
	return slices.Clone(c.diagnostics)
}

// Create returns a runnable command; its PrintableCommandArgs is the line to log.
func (c Command) Create(factory command.Factory, opts *command.Opts) command.Command {
	return factory.Create(toolName, c.Args(), opts)
}

// assemble parses, checks and merges the additional options over the derived ones.
func assemble(derived Options, additional []string, spec actionSpec, validation Validation) (Command, error) {
	user, diagnostics := ParseAdditionalOptions(additional)
	diagnostics = append(diagnostics, spec.check(user)...)

	merged, mergeDiagnostics := merge(derived, user, spec)
	diagnostics = append(diagnostics, mergeDiagnostics...)
	if validation == Fail {
		if err := firstFailure(diagnostics); err != nil {
			return Command{}, err
		}
	}

	return Command{args: merged.Args(), diagnostics: diagnostics}, nil
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

func (p projectOptions) render() Options {
	opts := containerOptions(p.projectPath)
	opts = appendValue(opts, "-scheme", p.scheme)
	opts = appendValue(opts, "-configuration", p.configuration)
	opts = appendValue(opts, "-destination", p.destination)
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

func appendAuthentication(opts Options, auth *Authentication) Options {
	if auth == nil {
		return opts
	}
	return append(opts, auth.options()...)
}
