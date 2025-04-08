package xcodeversion

import (
	"fmt"

	"github.com/bitrise-io/go-utils/v2/command"
)

// Version ...
type Version struct {
	Version      string
	BuildVersion string
	MajorVersion int64
}

// Reader ...
type Reader interface {
	GetVersion() (Version, error)
}

type reader struct {
	commandFactory command.Factory
}

// NewXcodeVersionProvider ...
func NewXcodeVersionProvider(commandFactory command.Factory) Reader {
	return &reader{
		commandFactory: commandFactory,
	}
}

// GetVersion ...
func (b *reader) GetVersion() (Version, error) {
	cmd := b.commandFactory.Create("xcodebuild", []string{"-version"}, &command.Opts{})

	outStr, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return Version{}, fmt.Errorf("xcodebuild -version failed: %s, output: %s", err, outStr)
	}

	return getXcodeVersionFromXcodebuildOutput(outStr)
}
