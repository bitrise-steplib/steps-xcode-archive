package xcodecommand

// BuildParams describes an `xcodebuild build` invocation. Zero-valued fields are omitted.
type BuildParams struct {
	ProjectPath        string // .xcodeproj, .xcworkspace or a Swift package (no flag; run in its directory)
	Scheme             string
	Configuration      string
	Destination        string // a default: a -destination in AdditionalOptions replaces it
	XCConfigPath       string
	SDK                string
	DisableCodeSigning bool // CODE_SIGNING_ALLOWED=NO
	Clean              bool // run clean first
	Authentication     *Authentication
	AdditionalOptions  Options // the step's xcodebuild_options, from ParseOptions
}

// Build renders params into a build Command.
func Build(params BuildParams) (Command, error) {
	opts := actions(params.Clean, ActionBuild)
	opts = append(opts, projectOptions{
		projectPath:   params.ProjectPath,
		scheme:        params.Scheme,
		configuration: params.Configuration,
		destination:   params.Destination,
		xcconfigPath:  params.XCConfigPath,
		sdk:           params.SDK,
	}.options()...)
	opts = appendAuthentication(opts, params.Authentication)
	opts = appendCodeSigningAllowed(opts, params.DisableCodeSigning)

	return assemble(opts, params.AdditionalOptions, buildPolicy)
}

func appendCodeSigningAllowed(opts Options, disable bool) Options {
	if !disable {
		return opts
	}
	return append(opts, Option{Kind: BuildSetting, Name: "CODE_SIGNING_ALLOWED", Value: "NO"})
}

var buildPolicy = actionPolicy{
	name:       ActionBuild,
	rejections: []rejection{modeSwitching, silentLog, testOnly},
	defaults:   []string{"-destination"},
	appendable: []string{"-arch"},
}
