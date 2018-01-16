package tools

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/envutil"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/stretchr/testify/require"
)

func TestExportEnvironmentWithEnvman(t *testing.T) {
	key := "ExportEnvironmentWithEnvmanKey"

	tmpDir, err := pathutil.NormalizedOSTempDirPath("test")
	require.NoError(t, err)

	// envman export requires an envstore
	revokeFn, err := pathutil.RevokableChangeDir(tmpDir)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, revokeFn())
	}()

	tmpEnvStorePth := filepath.Join(tmpDir, ".envstore.yml")
	require.NoError(t, fileutil.WriteStringToFile(tmpEnvStorePth, ""))

	envstoreRevokeFn, err := envutil.RevokableSetenv("ENVMAN_ENVSTORE_PATH", tmpEnvStorePth)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, envstoreRevokeFn())
	}()
	// ---

	{
		// envstor should be clear
		cmd := command.New("envman", "print")
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		require.NoError(t, err, out)
		require.Equal(t, "", out)
	}

	value := "test"
	require.NoError(t, ExportEnvironmentWithEnvman(key, value))

	// envstore should contain ExportEnvironmentWithEnvmanKey env var
	cmd := command.New("envman", "print")
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	require.NoError(t, err, out)
	require.Equal(t, fmt.Sprintf("%s: %s", key, value), out)
}
