package xcodecommand

// BuildForTestingParams describes an `xcodebuild build-for-testing` invocation.
// Zero-valued fields are omitted.
type BuildForTestingParams struct {
	ProjectPath       string // .xcodeproj, .xcworkspace or a Swift package (no flag; run in its directory)
	Scheme            string
	Configuration     string
	Destination       string
	XCConfigPath      string
	SDK               string
	TestPlan          string
	Clean             bool // run clean first
	Authentication    *Authentication
	AdditionalOptions []string // the step's xcodebuild_options, shell-split
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

	return assemble(opts, params.AdditionalOptions)
}
