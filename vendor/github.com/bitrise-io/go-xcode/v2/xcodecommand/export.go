package xcodecommand

// ExportArchiveParams describes an `xcodebuild -exportArchive` invocation. Zero-valued
// fields are omitted.
type ExportArchiveParams struct {
	ArchivePath        string
	ExportPath         string
	ExportOptionsPlist string
	Authentication     *Authentication
	AdditionalOptions  []string // the step's xcodebuild_options, shell-split
	Validation         Validation
}

// ExportArchive renders params into an export Command.
func ExportArchive(params ExportArchiveParams) (Command, error) {
	opts := Options{{Kind: Switch, Name: "-exportArchive"}}
	opts = appendValue(opts, "-archivePath", params.ArchivePath)
	opts = appendValue(opts, "-exportPath", params.ExportPath)
	opts = appendValue(opts, "-exportOptionsPlist", params.ExportOptionsPlist)
	opts = appendAuthentication(opts, params.Authentication)

	return assemble(opts, params.AdditionalOptions, exportArchiveSpec, params.Validation)
}
