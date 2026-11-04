// Package filewriter writes files without changing the mode of existing ones.
package filewriter

import (
	"os"

	"github.com/bitrise-io/go-utils/v2/fileutil"
)

// WriteKeepingMode writes content to pth. An existing file keeps its mode; a new one gets
// newFileMode, and its missing parent directories are created.
//
// FileManager.Write always chmods, which fails with EPERM for a file the process can write but
// doesn't own, so it is only used to create files; WriteBytes, which does not chmod, overwrites
// existing ones.
func WriteKeepingMode(fileManager fileutil.FileManager, pth string, content []byte, newFileMode os.FileMode) error {
	if _, err := fileManager.Lstat(pth); err == nil {
		return fileManager.WriteBytes(pth, content)
	}
	return fileManager.Write(pth, string(content), newFileMode)
}
