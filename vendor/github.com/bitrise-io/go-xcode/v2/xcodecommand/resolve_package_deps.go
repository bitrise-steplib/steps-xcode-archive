package xcodecommand

// ResolvePackagesParams describes an `xcodebuild -resolvePackageDependencies` invocation.
// Zero-valued fields are omitted; a workspace needs a scheme.
type ResolvePackagesParams struct {
	ProjectPath       string // .xcodeproj, .xcworkspace or a Swift package (no flag; run in its directory)
	Scheme            string
	Configuration     string
	AdditionalOptions []string // the step's xcodebuild_options, shell-split, as given to its main command
}

// ResolvePackages renders params into a package resolution Command; run it with a Runner.
//
// A step resolves packages as a preparation for its main command and passes it the same
// xcodebuild_options. Options that do not apply to resolution (build actions, test-only
// and mode-switching flags, anything unparseable) are left out, and the Command reports no
// diagnostics: the main command reports those options, once.
func ResolvePackages(params ResolvePackagesParams) (Command, error) {
	opts := containerOptions(params.ProjectPath)
	opts = appendValue(opts, "-scheme", params.Scheme)
	opts = appendValue(opts, "-configuration", params.Configuration)
	opts = append(opts, Option{Kind: Switch, Name: "-resolvePackageDependencies"})

	// clean archive -test-iterations 2 -skipMacroValidation FOO=1 -> -skipMacroValidation FOO=1
	user := ParseAdditionalOptions(params.AdditionalOptions).Filter(func(o Option) bool {
		_, rejected := resolvePackagesPolicy.rejects(o)
		return o.Kind != Action && o.Kind != Unknown && !rejected
	})
	merged, _ := merge(opts, user, resolvePackagesPolicy)

	return Command{args: merged.Args()}, nil
}
