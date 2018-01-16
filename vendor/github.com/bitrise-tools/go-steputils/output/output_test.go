package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bitrise-io/go-utils/envutil"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/stretchr/testify/require"
)

func TestZipAndExportOutputDir(t *testing.T) {
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

	sourceDir := filepath.Join(tmpDir, "source")
	require.NoError(t, os.MkdirAll(sourceDir, 0777))

	destinationZip := filepath.Join(tmpDir, "destination.zip")

	envKey := "EXPORTED_ZIP_PATH"
	require.NoError(t, ZipAndExportOutput(sourceDir, destinationZip, envKey))

	// destination should exist
	exist, err := pathutil.IsPathExists(destinationZip)
	require.NoError(t, err)
	require.Equal(t, true, exist, tmpDir)

	// destination should be exported
	envstoreContent, err := fileutil.ReadStringFromFile(tmpEnvStorePth)
	require.NoError(t, err)
	t.Logf("envstoreContent: %s\n", envstoreContent)
	require.Equal(t, true, strings.Contains(envstoreContent, "- "+envKey+": "+destinationZip), envstoreContent)
}

func TestExportOutputFileContent(t *testing.T) {
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

	sourceFileContent := "test"

	destinationFile := filepath.Join(tmpDir, "destination")

	envKey := "EXPORTED_FILE_PATH"
	require.NoError(t, ExportOutputFileContent(sourceFileContent, destinationFile, envKey))

	// destination should exist
	exist, err := pathutil.IsPathExists(destinationFile)
	require.NoError(t, err)
	require.Equal(t, true, exist)

	// destination should contain the source content
	content, err := fileutil.ReadStringFromFile(destinationFile)
	require.NoError(t, err)
	require.Equal(t, sourceFileContent, content)

	// destination should be exported
	envstoreContent, err := fileutil.ReadStringFromFile(os.Getenv("ENVMAN_ENVSTORE_PATH"))
	require.NoError(t, err)
	require.Equal(t, true, strings.Contains(envstoreContent, "- "+envKey+": "+destinationFile), envstoreContent)

	require.NoError(t, revokeFn())
}

func TestExportOutputFile(t *testing.T) {
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

	sourceFile := filepath.Join(tmpDir, "source")
	require.NoError(t, fileutil.WriteStringToFile(sourceFile, ""))

	destinationFile := filepath.Join(tmpDir, "destination")

	envKey := "EXPORTED_FILE_PATH"
	require.NoError(t, ExportOutputFile(sourceFile, destinationFile, envKey))

	// destination should exist
	exist, err := pathutil.IsPathExists(destinationFile)
	require.NoError(t, err)
	require.Equal(t, true, exist)

	// destination should be exported
	envstoreContent, err := fileutil.ReadStringFromFile(os.Getenv("ENVMAN_ENVSTORE_PATH"))
	require.NoError(t, err)
	require.Equal(t, true, strings.Contains(envstoreContent, "- "+envKey+": "+destinationFile), envstoreContent)

	require.NoError(t, revokeFn())
}

func TestExportOutputDir(t *testing.T) {
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

	sourceDir := filepath.Join(tmpDir, "source")
	require.NoError(t, os.MkdirAll(sourceDir, 0777))

	destinationDir := filepath.Join(tmpDir, "destination")

	envKey := "EXPORTED_DIR_PATH"
	require.NoError(t, ExportOutputDir(sourceDir, destinationDir, envKey))

	// destination should exist
	exist, err := pathutil.IsDirExists(destinationDir)
	require.NoError(t, err)
	require.Equal(t, true, exist)

	// destination should be exported
	envstoreContent, err := fileutil.ReadStringFromFile(os.Getenv("ENVMAN_ENVSTORE_PATH"))
	require.NoError(t, err)
	require.Equal(t, true, strings.Contains(envstoreContent, "- "+envKey+": "+destinationDir), envstoreContent)

	require.NoError(t, revokeFn())
}
