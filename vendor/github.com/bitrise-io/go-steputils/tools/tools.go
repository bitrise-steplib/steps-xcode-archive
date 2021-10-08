package tools

import (
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/env"
	"strings"
)

// TODO remove
var temporaryFactory = command.NewFactory(env.NewRepository())

// ExportEnvironmentWithEnvman ...
func ExportEnvironmentWithEnvman(key, value string) error {
	cmd := temporaryFactory.Create("envman", []string{"add", "--key", key}, &command.Opts{Stdin: strings.NewReader(value)})
	return cmd.Run()
}
