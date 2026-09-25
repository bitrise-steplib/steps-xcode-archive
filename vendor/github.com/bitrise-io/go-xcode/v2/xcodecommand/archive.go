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
	AdditionalOptions []string // the step's xcodebuild_options, shell-split
	Validation        Validation
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

	return assemble(opts, params.AdditionalOptions, archivePolicy, params.Validation)
}
