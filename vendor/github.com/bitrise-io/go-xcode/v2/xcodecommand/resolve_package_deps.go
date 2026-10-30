package xcodecommand

import (
	"fmt"
	"path/filepath"
)

// ResolvePackagesParams describes an `xcodebuild -resolvePackageDependencies` invocation.
// Zero-valued fields are omitted; a workspace needs a scheme.
type ResolvePackagesParams struct {
	ProjectPath       string // .xcodeproj, .xcworkspace or a Swift package (no flag; run in its directory)
	Scheme            string
	Configuration     string
	AdditionalOptions Options // the step's xcodebuild_options, from ParseOptions, as given to its main command
}

// ResolvePackages renders params into a package resolution Command; run it with a Runner.
//
// A step resolves packages as a preparation for its main command and passes it the same
// xcodebuild_options. Options that do not apply to resolution (build actions, test-only
// and mode-switching flags, anything unparseable) are left out, and the Command reports no
// diagnostics: the options still reach the main command, which reports any that are wrong
// there, once.
func ResolvePackages(params ResolvePackagesParams) (Command, error) {
	if filepath.Ext(filepath.Clean(params.ProjectPath)) == xcworkspaceExtension && params.Scheme == "" {
		// xcodebuild: "If you specify a workspace then you must also specify a scheme."
		return Command{}, fmt.Errorf("resolving the packages of workspace %s needs a scheme", params.ProjectPath)
	}
	opts := containerOptions(params.ProjectPath)
	opts = appendValue(opts, "-scheme", params.Scheme)
	opts = appendValue(opts, "-configuration", params.Configuration)
	opts = append(opts, Option{Kind: Switch, Name: "-resolvePackageDependencies"})

	// clean archive -test-iterations 2 -skipMacroValidation FOO=1 -> -skipMacroValidation FOO=1
	user := params.AdditionalOptions.Filter(func(o Option) bool {
		_, rejected := resolvePackagesPolicy.rejects(o)
		return o.Kind != Action && o.Kind != Unknown && !rejected
	})

	return Command{args: merge(opts, user, resolvePackagesPolicy).Args()}, nil
}

var resolvePackagesPolicy = actionPolicy{
	name:       "resolve packages",
	rejections: []rejection{modeSwitching.except("-resolvePackageDependencies"), testOnlyRefused, testOnly},
}
