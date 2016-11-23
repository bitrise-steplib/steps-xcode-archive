package utils

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
)

func zip(sourceDir, destinationZipPth string) error {
	parentDir := filepath.Dir(sourceDir)
	dirName := filepath.Base(sourceDir)
	cmd := cmdex.NewCommand("/usr/bin/zip", "-rTy", destinationZipPth, dirName)
	cmd.SetDir(parentDir)
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to zip dir: %s, output: %s, error: %s", sourceDir, out, err)
	}

	return nil
}

func exportEnvironmentWithEnvman(keyStr, valueStr string) error {
	cmd := cmdex.NewCommand("envman", "add", "--key", keyStr)
	cmd.SetStdin(strings.NewReader(valueStr))
	return cmd.Run()
}

// ExportOutputDir ...
func ExportOutputDir(sourceDirPth, destinationDirPth, envKey string) error {
	if sourceDirPth != destinationDirPth {
		if err := cmdex.CopyDir(sourceDirPth, destinationDirPth, true); err != nil {
			return err
		}
	}

	return exportEnvironmentWithEnvman(envKey, destinationDirPth)
}

// ExportOutputFile ...
func ExportOutputFile(sourcePth, destinationPth, envKey string) error {
	if sourcePth != destinationPth {
		if err := cmdex.CopyFile(sourcePth, destinationPth); err != nil {
			return err
		}
	}

	return exportEnvironmentWithEnvman(envKey, destinationPth)
}

// ExportOutputFileContent ...
func ExportOutputFileContent(content, destinationPth, envKey string) error {
	if err := fileutil.WriteStringToFile(destinationPth, content); err != nil {
		return err
	}

	return ExportOutputFile(destinationPth, destinationPth, envKey)
}

// ExportOutputDirAsZip ...
func ExportOutputDirAsZip(sourceDirPth, destinationPth, envKey string) error {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("__export_tmp_dir__")
	if err != nil {
		return err
	}

	base := filepath.Base(sourceDirPth)
	tmpZipFilePth := filepath.Join(tmpDir, base+".zip")

	if err := zip(sourceDirPth, tmpZipFilePth); err != nil {
		return err
	}

	return ExportOutputFile(tmpZipFilePth, destinationPth, envKey)
}
