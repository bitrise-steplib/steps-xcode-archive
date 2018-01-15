package input

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/stretchr/testify/require"
)

func TestValidateWithOptions(t *testing.T) {
	err := ValidateWithOptions("testinput", "tst0", "tst1", "testinput")
	require.NoError(t, err)

	err = ValidateWithOptions("testinput", "test", "input")
	require.EqualError(t, err, "invalid parameter: testinput, available: [test input]")

	err = ValidateWithOptions("testinput")
	require.EqualError(t, err, "invalid parameter: testinput, available: []")

	err = ValidateWithOptions("", "param1", "param2")
	require.EqualError(t, err, "parameter not specified")
}

func TestValidateIfNotEmpty(t *testing.T) {
	err := ValidateIfNotEmpty("testinput")
	require.NoError(t, err)

	err = ValidateIfNotEmpty("")
	require.EqualError(t, err, "parameter not specified")
}

func TestSecureInput(t *testing.T) {
	output := SecureInput("testinput")
	require.Equal(t, "***", output)

	output = SecureInput("")
	require.Equal(t, "", output)
}

func TestValidateIfPathExists(t *testing.T) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("test")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(tmpDir))
	}()

	t.Log("no error - if dir exist")
	{
		err := ValidateIfPathExists(tmpDir)
		require.NoError(t, err)
	}

	t.Log("no error - if file exist")
	{
		pth := filepath.Join(tmpDir, "test")
		require.NoError(t, fileutil.WriteStringToFile(pth, ""))

		err := ValidateIfPathExists(pth)
		require.NoError(t, err)
	}

	t.Log("error - if path does not exist")
	{
		err := ValidateIfPathExists("/not/exists/for/sure")
		require.EqualError(t, err, "path not exist at: /not/exists/for/sure")
	}
}

func TestValidateIfDirExists(t *testing.T) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("test")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(tmpDir))
	}()

	t.Log("no error - if dir exist")
	{
		err := ValidateIfDirExists(tmpDir)
		require.NoError(t, err)
	}

	t.Log("error - if dir does not exist")
	{
		err := ValidateIfDirExists("/not/exists/for/sure")
		require.EqualError(t, err, "dir not exist at: /not/exists/for/sure")
	}

	t.Log("error - if path is a file path")
	{
		pth := filepath.Join(tmpDir, "test")
		require.NoError(t, fileutil.WriteStringToFile(pth, ""))

		err := ValidateIfDirExists(pth)
		require.EqualError(t, err, "dir not exist at: "+pth)
	}

}
