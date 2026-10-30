package xcodecommand

// ArchiveParams describes an `xcodebuild archive` invocation. Zero-valued fields are omitted.
type ArchiveParams struct {
	ProjectPath       string // .xcodeproj, .xcworkspace or a Swift package (no flag; run in its directory)
	Scheme            string
	Configuration     string
	Destination       string // a default: a -destination in AdditionalOptions replaces it
	XCConfigPath      string
	SDK               string
	ArchivePath       string
	Clean             bool // run clean first
	Authentication    *Authentication
	AdditionalOptions Options // the step's xcodebuild_options, from ParseOptions
}

// Archive renders params into an archive Command.
func Archive(params ArchiveParams) (Command, error) {
	opts := actions(params.Clean, ActionArchive)
	opts = append(opts, projectOptions{
		projectPath:   params.ProjectPath,
		scheme:        params.Scheme,
		configuration: params.Configuration,
		destination:   params.Destination,
		xcconfigPath:  params.XCConfigPath,
		sdk:           params.SDK,
	}.options()...)
	opts = appendValue(opts, "-archivePath", params.ArchivePath)
	opts = appendAuthentication(opts, params.Authentication)

	return assemble(opts, params.AdditionalOptions, archivePolicy)
}

// The step's -destination is a fallback the user's replaces; -arch is repeatable. The
// other build actions follow the same rule.
var archivePolicy = actionPolicy{
	name:       ActionArchive,
	rejections: []rejection{modeSwitching, testOnlyRefused, testOnly},
	defaults:   []string{"-destination"},
	appendable: []string{"-arch"},
}
