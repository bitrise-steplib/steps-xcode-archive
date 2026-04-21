package buildcache

import (
	"testing"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/stretchr/testify/assert"
)

func TestWrappingFactory_RewritesXcodebuild(t *testing.T) {
	inner := command.NewFactory(env.NewRepository())
	wrapped := NewWrappingCommandFactory(inner, "/opt/bin/bitrise-build-cache")

	cmd := wrapped.Create("xcodebuild", []string{"-project", "App.xcodeproj", "archive"}, nil)

	printed := cmd.PrintableCommandArgs()
	assert.Contains(t, printed, "/opt/bin/bitrise-build-cache")
	assert.Contains(t, printed, "react-native")
	assert.Contains(t, printed, "run")
	assert.Contains(t, printed, "xcodebuild")
	assert.Contains(t, printed, "-project")
	assert.Contains(t, printed, "archive")
}

func TestWrappingFactory_PassesThroughOtherBinaries(t *testing.T) {
	inner := command.NewFactory(env.NewRepository())
	wrapped := NewWrappingCommandFactory(inner, "/opt/bin/bitrise-build-cache")

	cmd := wrapped.Create("xcbeautify", []string{"--disable-colored-output"}, nil)

	printed := cmd.PrintableCommandArgs()
	assert.Contains(t, printed, "xcbeautify")
	assert.NotContains(t, printed, "bitrise-build-cache")
	assert.NotContains(t, printed, "react-native")
}
