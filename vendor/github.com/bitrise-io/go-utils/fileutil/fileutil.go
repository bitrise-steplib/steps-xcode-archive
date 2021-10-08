package fileutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/bitrise-io/go-utils/pathutil"
)

// FileManager ...
type FileManager interface {
	Remove(path string) error
	RemoveAll(path string) error
	Write(path string, value string, mode os.FileMode) error
}

type fileManager struct{}

// NewFileManager ...
func NewFileManager() FileManager {
	return fileManager{}
}

// Remove ...
func (fileManager) Remove(path string) error {
	return os.Remove(path)
}

// RemoveAll ...
func (fileManager) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

// Write ...
func (fileManager) Write(path string, value string, mode os.FileMode) error {
	if err := ensureSavePath(path); err != nil {
		return err
	}

	if err := WriteStringToFile(path, value); err != nil {
		return err
	}

	if err := os.Chmod(path, mode); err != nil {
		return err
	}
	return nil
}

func ensureSavePath(savePath string) error {
	dirPath := filepath.Dir(savePath)
	return os.MkdirAll(dirPath, 0700)
}

// WriteStringToFile ...
func WriteStringToFile(pth string, fileCont string) error {
	return WriteBytesToFile(pth, []byte(fileCont))
}

// WriteStringToFileWithPermission ...
func WriteStringToFileWithPermission(pth string, fileCont string, perm os.FileMode) error {
	return WriteBytesToFileWithPermission(pth, []byte(fileCont), perm)
}

// WriteBytesToFileWithPermission ...
func WriteBytesToFileWithPermission(pth string, fileCont []byte, perm os.FileMode) error {
	if pth == "" {
		return errors.New("No path provided")
	}

	var file *os.File
	var err error
	if perm == 0 {
		file, err = os.Create(pth)
	} else {
		// same as os.Create, but with a specified permission
		//  the flags are copy-pasted from the official
		//  os.Create func: https://golang.org/src/os/file.go?s=7327:7366#L244
		file, err = os.OpenFile(pth, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
	}
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Println(" [!] Failed to close file:", err)
		}
	}()

	if _, err := file.Write(fileCont); err != nil {
		return err
	}

	return nil
}

// WriteBytesToFile ...
func WriteBytesToFile(pth string, fileCont []byte) error {
	return WriteBytesToFileWithPermission(pth, fileCont, 0)
}

// WriteJSONToFile ...
func WriteJSONToFile(pth string, fileCont interface{}) error {
	bytes, err := json.Marshal(fileCont)
	if err != nil {
		return fmt.Errorf("failed to JSON marshal the provided object: %+v", err)
	}
	return WriteBytesToFile(pth, bytes)
}

// AppendStringToFile ...
func AppendStringToFile(pth string, fileCont string) error {
	return AppendBytesToFile(pth, []byte(fileCont))
}

// AppendBytesToFile ...
func AppendBytesToFile(pth string, fileCont []byte) error {
	if pth == "" {
		return errors.New("No path provided")
	}

	var file *os.File
	filePerm, err := GetFilePermissions(pth)
	if err != nil {
		// create the file
		file, err = os.Create(pth)
	} else {
		// open for append
		file, err = os.OpenFile(pth, os.O_APPEND|os.O_CREATE|os.O_WRONLY, filePerm)
	}
	if err != nil {
		// failed to create or open-for-append the file
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Println(" [!] Failed to close file:", err)
		}
	}()

	if _, err := file.Write(fileCont); err != nil {
		return err
	}

	return nil
}

// ReadBytesFromFile ...
func ReadBytesFromFile(pth string) ([]byte, error) {
	if isExists, err := pathutil.IsPathExists(pth); err != nil {
		return []byte{}, err
	} else if !isExists {
		return []byte{}, fmt.Errorf("No file found at path: %s", pth)
	}

	bytes, err := ioutil.ReadFile(pth)
	if err != nil {
		return []byte{}, err
	}
	return bytes, nil
}

// ReadStringFromFile ...
func ReadStringFromFile(pth string) (string, error) {
	contBytes, err := ReadBytesFromFile(pth)
	if err != nil {
		return "", err
	}
	return string(contBytes), nil
}

// GetFileModeOfFile ...
//  this is the "permissions" info, which can be passed directly to
//  functions like WriteBytesToFileWithPermission or os.OpenFile
func GetFileModeOfFile(pth string) (os.FileMode, error) {
	finfo, err := os.Lstat(pth)
	if err != nil {
		return 0, err
	}
	return finfo.Mode(), nil
}

// GetFilePermissions ...
// - alias of: GetFileModeOfFile
//  this is the "permissions" info, which can be passed directly to
//  functions like WriteBytesToFileWithPermission or os.OpenFile
func GetFilePermissions(filePth string) (os.FileMode, error) {
	return GetFileModeOfFile(filePth)
}
