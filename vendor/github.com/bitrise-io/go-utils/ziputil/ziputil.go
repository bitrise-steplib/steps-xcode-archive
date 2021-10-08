package ziputil

import (
	"fmt"
	"github.com/bitrise-io/go-utils/env"
	"path/filepath"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/pathutil"
)

// ZipDir ...
func ZipDir(sourceDirPth, destinationZipPth string, isContentOnly bool) error {
	if exist, err := pathutil.IsDirExists(sourceDirPth); err != nil {
		return err
	} else if !exist {
		return fmt.Errorf("dir (%s) not exist", sourceDirPth)
	}

	workDir := filepath.Dir(sourceDirPth)
	if isContentOnly {
		workDir = sourceDirPth
	}

	zipTarget := filepath.Base(sourceDirPth)
	if isContentOnly {
		zipTarget = "."
	}

	// -r - Travel the directory structure recursively
	// -T - Test the integrity of the new zip file
	// -y - Store symbolic links as such in the zip archive, instead of compressing and storing the file referred to by the link
	opts := &command.Opts{Dir: workDir}
	factory := command.NewFactory(env.NewRepository())
	cmd := factory.Create("/usr/bin/zip", []string{"-rTy", destinationZipPth, zipTarget}, opts)
	if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
		return fmt.Errorf("command: (%s) failed, output: %s, error: %s", cmd.PrintableCommandArgs(), out, err)
	}

	return nil
}

// ZipFile ...
func ZipFile(sourceFilePth, destinationZipPth string) error {
	if exist, err := pathutil.IsPathExists(sourceFilePth); err != nil {
		return err
	} else if !exist {
		return fmt.Errorf("file (%s) not exist", sourceFilePth)
	}

	workDir := filepath.Dir(sourceFilePth)
	zipTarget := filepath.Base(sourceFilePth)

	// -T - Test the integrity of the new zip file
	// -y - Store symbolic links as such in the zip archive, instead of compressing and storing the file referred to by the link
	opts := &command.Opts{Dir: workDir}
	factory := command.NewFactory(env.NewRepository())
	cmd := factory.Create("/usr/bin/zip", []string{"-Ty", destinationZipPth, zipTarget}, opts)
	if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
		return fmt.Errorf("command: (%s) failed, output: %s, error: %s", cmd.PrintableCommandArgs(), out, err)
	}

	return nil
}

// UnZip ...
func UnZip(zip, intoDir string) error {
	factory := command.NewFactory(env.NewRepository())
	cmd := factory.Create("/usr/bin/unzip", []string{zip, "-d", intoDir}, nil)
	if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
		return fmt.Errorf("command: (%s) failed, output: %s, error: %s", cmd.PrintableCommandArgs(), out, err)
	}

	return nil
}
