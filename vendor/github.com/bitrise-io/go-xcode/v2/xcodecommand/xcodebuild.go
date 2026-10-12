package xcodecommand

const toolName = "xcodebuild"

// Build actions xcodebuild accepts (Xcode 16).
const (
	ActionBuild               = "build"
	ActionBuildForTesting     = "build-for-testing"
	ActionAnalyze             = "analyze"
	ActionArchive             = "archive"
	ActionTest                = "test"
	ActionTestWithoutBuilding = "test-without-building"
	ActionDocBuild            = "docbuild"
	ActionInstallSrc          = "installsrc"
	ActionInstall             = "install"
	ActionClean               = "clean"
)

var knownActions = []string{
	ActionBuild, ActionBuildForTesting, ActionAnalyze, ActionArchive,
	ActionTest, ActionTestWithoutBuilding, ActionDocBuild,
	ActionInstallSrc, ActionInstall, ActionClean,
}

// Authentication is the App Store Connect API key for -allowProvisioningUpdates.
type Authentication struct {
	KeyPath  string
	KeyID    string
	IssuerID string
}

func (a Authentication) options() Options {
	return Options{
		{Kind: Switch, Name: "-allowProvisioningUpdates"},
		{Kind: ValueOption, Name: "-authenticationKeyPath", Value: a.KeyPath},
		{Kind: ValueOption, Name: "-authenticationKeyID", Value: a.KeyID},
		{Kind: ValueOption, Name: "-authenticationKeyIssuerID", Value: a.IssuerID},
	}
}
