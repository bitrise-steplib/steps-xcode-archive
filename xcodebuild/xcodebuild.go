package xcodebuild

import "github.com/bitrise-io/steps-xcode-archive/xcpretty"

const (
	toolName = "xcodebuild"
)

// Model ...
type Model struct {
	projectAction string
	projectPath   string
	scheme        string
	configuration string

	isCleanBuild bool

	archivePath string

	forceDevelopmentTeam              string
	forceProvisioningProfileSpecifier string
	forceProvisioningProfile          string
	forceCodeSignIdentity             string

	// export options
	exportFormat              string
	exportPath                string
	exportProvisioningProfile string
	exportOptionsPlist        string

	customOptions []string

	prettyCmd xcpretty.Model
}

// New ...
func New() *Model {
	return &Model{}
}

// SetProjectAction ...
func (xb *Model) SetProjectAction(projectAction string) *Model {
	xb.projectAction = projectAction
	return xb
}

// SetProjectPath ...
func (xb *Model) SetProjectPath(projectPath string) *Model {
	xb.projectPath = projectPath
	return xb
}

// SetScheme ...
func (xb *Model) SetScheme(scheme string) *Model {
	xb.scheme = scheme
	return xb
}

// SetConfiguration ...
func (xb *Model) SetConfiguration(configuration string) *Model {
	xb.configuration = configuration
	return xb
}

// SetIsCleanBuild ...
func (xb *Model) SetIsCleanBuild(isCleanBuild bool) *Model {
	xb.isCleanBuild = isCleanBuild
	return xb
}

// SetArchivePath ...
func (xb *Model) SetArchivePath(archivePath string) *Model {
	xb.archivePath = archivePath
	return xb
}

// SetForceDevelopmentTeam ...
func (xb *Model) SetForceDevelopmentTeam(forceDevelopmentTeam string) *Model {
	xb.forceDevelopmentTeam = forceDevelopmentTeam
	return xb
}

// SetForceProvisioningProfileSpecifier ...
func (xb *Model) SetForceProvisioningProfileSpecifier(forceProvisioningProfileSpecifier string) *Model {
	xb.forceProvisioningProfileSpecifier = forceProvisioningProfileSpecifier
	return xb
}

// SetForceProvisioningProfile ...
func (xb *Model) SetForceProvisioningProfile(forceProvisioningProfile string) *Model {
	xb.forceProvisioningProfile = forceProvisioningProfile
	return xb
}

// SetForceCodeSignIdentity ...
func (xb *Model) SetForceCodeSignIdentity(forceCodeSignIdentity string) *Model {
	xb.forceCodeSignIdentity = forceCodeSignIdentity
	return xb
}

// SetCustomOptions ...
func (xb *Model) SetCustomOptions(customOptions []string) *Model {
	xb.customOptions = customOptions
	return xb
}
