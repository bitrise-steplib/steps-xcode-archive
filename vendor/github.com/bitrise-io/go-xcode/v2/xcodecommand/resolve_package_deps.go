package xcodecommand

// ResolvePackagesParams describes an `xcodebuild -resolvePackageDependencies` invocation.
// Zero-valued fields are omitted; a workspace needs a scheme.
type ResolvePackagesParams struct {
	ProjectPath       string // .xcodeproj, .xcworkspace or a Swift package (no flag; run in its directory)
	Scheme            string
	Configuration     string
	AdditionalOptions []string // the step's xcodebuild_options, shell-split
	Validation        Validation
}

// ResolvePackages renders params into a package resolution Command; run it with a Runner.
func ResolvePackages(params ResolvePackagesParams) (Command, error) {
	opts := containerOptions(params.ProjectPath)
	opts = appendValue(opts, "-scheme", params.Scheme)
	opts = appendValue(opts, "-configuration", params.Configuration)
	opts = append(opts, Option{Kind: Switch, Name: "-resolvePackageDependencies"})

	return assemble(opts, params.AdditionalOptions, resolvePackagesSpec, params.Validation)
}
