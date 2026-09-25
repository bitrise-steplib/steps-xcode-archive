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
	AdditionalOptions  []string // the step's xcodebuild_options, shell-split
	Validation         Validation
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

	return assemble(opts, params.AdditionalOptions, buildPolicy, params.Validation)
}

// AnalyzeParams describes an `xcodebuild analyze` invocation. Zero-valued fields are omitted.
type AnalyzeParams struct {
	ProjectPath        string // .xcodeproj, .xcworkspace or a Swift package (no flag; run in its directory)
	Scheme             string
	Configuration      string
	Destination        string // a default: a -destination in AdditionalOptions replaces it
	XCConfigPath       string
	SDK                string
	ResultBundlePath   string   // a default: a -resultBundlePath in AdditionalOptions replaces it
	DisableCodeSigning bool     // CODE_SIGNING_ALLOWED=NO
	Clean              bool     // run clean first
	AdditionalOptions  []string // the step's xcodebuild_options, shell-split
	Validation         Validation
}

// Analyze renders params into an analyze Command.
func Analyze(params AnalyzeParams) (Command, error) {
	opts := actions(params.Clean, ActionAnalyze)
	opts = append(opts, projectOptions{
		projectPath:   params.ProjectPath,
		scheme:        params.Scheme,
		configuration: params.Configuration,
		destination:   params.Destination,
		xcconfigPath:  params.XCConfigPath,
		sdk:           params.SDK,
	}.options()...)
	opts = appendValue(opts, "-resultBundlePath", params.ResultBundlePath)
	opts = appendCodeSigningAllowed(opts, params.DisableCodeSigning)

	return assemble(opts, params.AdditionalOptions, analyzePolicy, params.Validation)
}

// BuildForTestingParams describes an `xcodebuild build-for-testing` invocation.
// Zero-valued fields are omitted.
type BuildForTestingParams struct {
	ProjectPath       string // .xcodeproj, .xcworkspace or a Swift package (no flag; run in its directory)
	Scheme            string
	Configuration     string
	Destination       string // a default: a -destination in AdditionalOptions replaces it
	XCConfigPath      string
	SDK               string
	TestPlan          string
	Clean             bool // run clean first
	Authentication    *Authentication
	AdditionalOptions []string // the step's xcodebuild_options, shell-split
	Validation        Validation
}

// BuildForTesting renders params into a build-for-testing Command.
func BuildForTesting(params BuildForTestingParams) (Command, error) {
	opts := actions(params.Clean, ActionBuildForTesting)
	opts = append(opts, projectOptions{
		projectPath:   params.ProjectPath,
		scheme:        params.Scheme,
		configuration: params.Configuration,
		destination:   params.Destination,
		xcconfigPath:  params.XCConfigPath,
		sdk:           params.SDK,
	}.options()...)
	opts = appendAuthentication(opts, params.Authentication)
	opts = appendValue(opts, "-testPlan", params.TestPlan)

	return assemble(opts, params.AdditionalOptions, buildForTestingPolicy, params.Validation)
}

func appendCodeSigningAllowed(opts Options, disable bool) Options {
	if !disable {
		return opts
	}
	return append(opts, Option{Kind: BuildSetting, Name: "CODE_SIGNING_ALLOWED", Value: "NO"})
}
