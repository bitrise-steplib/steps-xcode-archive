package step

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
)

type outputExporter struct {
	fileManager  fileutil.FileManager
	pathProvider pathutil.PathProvider
	cmdFactory   command.Factory
	logger       log.Logger
}

func (e outputExporter) zip(sourceDir, destinationZipPth string) error {
	e.logger.TPrintf("Will zip directory path: %s", sourceDir)

	parentDir := filepath.Dir(sourceDir)
	dirName := filepath.Base(sourceDir)
	cmd := e.cmdFactory.Create("/usr/bin/zip", []string{"-rTy", destinationZipPth, dirName}, &command.Opts{Dir: parentDir})
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to zip dir: %s, output: %s, error: %s", sourceDir, out, err)
	}

	e.logger.TPrintf("Directory zipped.")

	return nil
}

func (e outputExporter) exportEnvironmentWithEnvman(keyStr, valueStr string) error {
	cmd := e.cmdFactory.Create("envman", []string{"add", "--key", keyStr}, &command.Opts{Stdin: strings.NewReader(valueStr)})
	return cmd.Run()
}

func (e outputExporter) ExportOutputDir(sourceDirPth, destinationDirPth, envKey string) error {
	if sourceDirPth != destinationDirPth {
		e.logger.TPrintf("Copying export output")

		if err := e.fileManager.CopyDir(sourceDirPth, destinationDirPth, &fileutil.CopyOptions{Overwrite: true}); err != nil {
			return err
		}

		e.logger.TPrintf("Copied export output to %s", destinationDirPth)
	}

	return e.exportEnvironmentWithEnvman(envKey, destinationDirPth)
}

func (e outputExporter) ExportOutputFile(sourcePth, destinationPth, envKey string) error {
	if sourcePth != destinationPth {
		if err := e.fileManager.CopyFile(sourcePth, destinationPth, &fileutil.CopyOptions{Overwrite: true}); err != nil {
			return err
		}
	}

	return e.exportEnvironmentWithEnvman(envKey, destinationPth)
}

func (e outputExporter) ExportOutputFileContent(content, destinationPth, envKey string) error {
	if err := e.fileManager.Write(destinationPth, content, 0644); err != nil {
		return err
	}

	return e.ExportOutputFile(destinationPth, destinationPth, envKey)
}

func (e outputExporter) ExportOutputDirAsZip(sourceDirPth, destinationPth, envKey string) error {
	tmpDir, err := e.pathProvider.CreateTempDir("__export_tmp_dir__")
	if err != nil {
		return err
	}

	base := filepath.Base(sourceDirPth)
	tmpZipFilePth := filepath.Join(tmpDir, base+".zip")

	if err := e.zip(sourceDirPth, tmpZipFilePth); err != nil {
		return err
	}

	return e.ExportOutputFile(tmpZipFilePth, destinationPth, envKey)
}

// ExportDSYMs copies each dSYM into dsymDir preserving its basename
// (matches v1 CopyDir(..., isOnlyContent=false) semantics).
func (e outputExporter) ExportDSYMs(dsymDir string, dsyms []string) error {
	for _, dsym := range dsyms {
		dst := filepath.Join(dsymDir, filepath.Base(dsym))
		if err := e.fileManager.CopyDir(dsym, dst, &fileutil.CopyOptions{Overwrite: true}); err != nil {
			return fmt.Errorf("could not copy (%s) to directory (%s): %s", dsym, dsymDir, err)
		}
	}
	return nil
}
