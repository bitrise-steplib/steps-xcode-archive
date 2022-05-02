package utils

import (
	"fmt"
	"path/filepath"
	"strings"

	v1command "github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/v2/command"
)

func zip(cmdFactory command.Factory, sourceDir, destinationZipPth string) error {
	parentDir := filepath.Dir(sourceDir)
	dirName := filepath.Base(sourceDir)
	cmd := cmdFactory.Create("/usr/bin/zip", []string{"-rTy", destinationZipPth, dirName}, &command.Opts{Dir: parentDir})
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to zip dir: %s, output: %s, error: %s", sourceDir, out, err)
	}

	return nil
}

func exportEnvironmentWithEnvman(cmdFactory command.Factory, keyStr, valueStr string) error {
	cmd := cmdFactory.Create("envman", []string{"add", "--key", keyStr}, &command.Opts{Stdin: strings.NewReader(valueStr)})
	return cmd.Run()
}

// ExportOutputDir ...
func ExportOutputDir(cmdFactory command.Factory, sourceDirPth, destinationDirPth, envKey string) error {
	if sourceDirPth != destinationDirPth {
		if err := v1command.CopyDir(sourceDirPth, destinationDirPth, true); err != nil {
			return err
		}
	}

	return exportEnvironmentWithEnvman(cmdFactory, envKey, destinationDirPth)
}

// ExportOutputFile ...
func ExportOutputFile(cmdFactory command.Factory, sourcePth, destinationPth, envKey string) error {
	if sourcePth != destinationPth {
		if err := v1command.CopyFile(sourcePth, destinationPth); err != nil {
			return err
		}
	}

	return exportEnvironmentWithEnvman(cmdFactory, envKey, destinationPth)
}

// ExportOutputFileContent ...
func ExportOutputFileContent(cmdFactory command.Factory, content, destinationPth, envKey string) error {
	if err := fileutil.WriteStringToFile(destinationPth, content); err != nil {
		return err
	}

	return ExportOutputFile(cmdFactory, destinationPth, destinationPth, envKey)
}

// ExportOutputDirAsZip ...
func ExportOutputDirAsZip(cmdFactory command.Factory, sourceDirPth, destinationPth, envKey string) error {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("__export_tmp_dir__")
	if err != nil {
		return err
	}

	base := filepath.Base(sourceDirPth)
	tmpZipFilePth := filepath.Join(tmpDir, base+".zip")

	if err := zip(cmdFactory, sourceDirPth, tmpZipFilePth); err != nil {
		return err
	}

	return ExportOutputFile(cmdFactory, tmpZipFilePth, destinationPth, envKey)
}
