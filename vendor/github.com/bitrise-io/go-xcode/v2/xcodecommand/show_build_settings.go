package xcodecommand

// showBuildSettingsParams describes an `xcodebuild -showBuildSettings` invocation.
// Package-local: settings are read through a provider (STEP-2169), the only caller.
// Narrow forwarded xcodebuild_options with ParseAdditionalOptions and Options.Filter first.
type showBuildSettingsParams struct {
	projectPath       string // .xcodeproj, .xcworkspace or a Swift package (no flag; run in its directory)
	target            string // -target; alternative to scheme
	scheme            string // -scheme; required for a workspace
	configuration     string
	additionalOptions []string
	validation        Validation
}

// showBuildSettings renders params into a settings query Command.
func showBuildSettings(params showBuildSettingsParams) (Command, error) {
	opts := containerOptions(params.projectPath)
	opts = appendValue(opts, "-target", params.target)
	opts = appendValue(opts, "-scheme", params.scheme)
	opts = appendValue(opts, "-configuration", params.configuration)
	opts = append(opts, Option{Kind: Switch, Name: "-showBuildSettings"})

	return assemble(opts, params.additionalOptions, showBuildSettingsSpec, params.validation)
}
