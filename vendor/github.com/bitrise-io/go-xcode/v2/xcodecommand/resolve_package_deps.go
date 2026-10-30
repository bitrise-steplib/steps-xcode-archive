package xcodecommand

import (
	"fmt"
	"path/filepath"
)

// ResolvePackagesParams describes an `xcodebuild -resolvePackageDependencies` invocation.
// Zero-valued fields are omitted; a workspace needs a scheme.
type ResolvePackagesParams struct {
	ProjectPath       string // .xcodeproj, .xcworkspace or a Swift package (no flag; run in its directory)
	Scheme            string
	Configuration     string
	AdditionalOptions Options // the step's xcodebuild_options, from ParseOptions, as given to its main command
}

// ResolvePackages renders params into a package resolution Command; run it with a Runner.
//
// A step resolves packages as a preparation for its main command and passes it the same
// xcodebuild_options. Options that do not apply to resolution (build actions, test-only
// and mode-switching flags, anything unparseable) are left out, and the Command reports
// them in one informational line: the options still reach the main command, which reports
// any that are wrong there.
func ResolvePackages(params ResolvePackagesParams) (Command, error) {
	if filepath.Ext(filepath.Clean(params.ProjectPath)) == xcworkspaceExtension && params.Scheme == "" {
		// xcodebuild: "If you specify a workspace then you must also specify a scheme."
		return Command{}, fmt.Errorf("resolving the packages of workspace %s needs a scheme", params.ProjectPath)
	}
	opts := containerOptions(params.ProjectPath)
	opts = appendValue(opts, "-scheme", params.Scheme)
	opts = appendValue(opts, "-configuration", params.Configuration)
	opts = append(opts, Option{Kind: Switch, Name: "-resolvePackageDependencies"})

	// clean archive -test-iterations 2 -skipMacroValidation FOO=1 -> -skipMacroValidation FOO=1
	var user, leftOut Options
	for _, o := range params.AdditionalOptions {
		o.user = true
		if _, rejected := resolvePackagesPolicy.rejects(o); o.Kind == Action || o.Kind == Unknown || rejected {
			leftOut = append(leftOut, o)
		} else {
			user = append(user, o)
		}
	}
	cmd := Command{args: merge(opts, user, resolvePackagesPolicy).Args()}
	if len(leftOut) > 0 {
		cmd.diagnostics = []Diagnostic{report(LeftOut, leftOut[0], "%s left out of package resolution; the Step's main command gets the full xcodebuild_options.", leftOut.quoted())}
	}
	return cmd, nil
}

var resolvePackagesPolicy = actionPolicy{
	name:       "resolve packages",
	rejections: []rejection{modeSwitching.except("-resolvePackageDependencies"), silentLog, testOnly},
}
