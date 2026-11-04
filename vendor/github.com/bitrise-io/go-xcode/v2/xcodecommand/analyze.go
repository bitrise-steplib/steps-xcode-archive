package xcodecommand

// AnalyzeParams describes an `xcodebuild analyze` invocation. Zero-valued fields are omitted.
type AnalyzeParams struct {
	ProjectPath        string // .xcodeproj, .xcworkspace or a Swift package (no flag; run in its directory)
	Scheme             string
	Configuration      string
	Destination        string
	XCConfigPath       string
	SDK                string
	ResultBundlePath   string
	DisableCodeSigning bool     // CODE_SIGNING_ALLOWED=NO
	Clean              bool     // run clean first
	AdditionalOptions  []string // the step's xcodebuild_options, shell-split
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

	return assemble(opts, params.AdditionalOptions)
}
