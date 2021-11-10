package xcodebuild

import "github.com/bitrise-io/go-utils/command"

const (
	toolName = "xcodebuild"
)

// CommandModel ...
type CommandModel interface {
	PrintableCmd() string
	Command(opts *command.Opts) command.Command
}

// AuthenticationParams ...
type AuthenticationParams struct {
	KeyID     string
	IsssuerID string
	KeyPath   string
}

func (a *AuthenticationParams) args() []string {
	return []string{
		"-allowProvisioningUpdates",
		"-authenticationKeyPath", a.KeyPath,
		"-authenticationKeyID", a.KeyID,
		"-authenticationKeyIssuerID", a.IsssuerID,
	}
}
