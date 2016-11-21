package xcodebuild

import "github.com/bitrise-io/go-utils/cmdex"

const (
	toolName = "xcodebuild"
)

// CommandModel ...
type CommandModel interface {
	PrintableCmd() string
	Command() *cmdex.CommandModel
}
