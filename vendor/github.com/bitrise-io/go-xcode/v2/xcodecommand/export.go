package xcodecommand

// ExportArchiveParams describes an `xcodebuild -exportArchive` invocation. Zero-valued
// fields are omitted.
type ExportArchiveParams struct {
	ArchivePath        string
	ExportPath         string
	ExportOptionsPlist string
	Authentication     *Authentication
}

// ExportArchive renders params into an export Command. It takes no additional options: the
// step's xcodebuild_options are for its build action, and no step passes them to the export.
func ExportArchive(params ExportArchiveParams) (Command, error) {
	opts := Options{{Kind: Switch, Name: "-exportArchive"}}
	opts = appendValue(opts, "-archivePath", params.ArchivePath)
	opts = appendValue(opts, "-exportPath", params.ExportPath)
	opts = appendValue(opts, "-exportOptionsPlist", params.ExportOptionsPlist)
	opts = appendAuthentication(opts, params.Authentication)

	return Command{args: opts.Args()}, nil
}
