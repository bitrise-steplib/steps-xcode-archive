package xcconfig

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/pathutil"
)

// Writer ...
type Writer interface {
	Write(input string) (string, error)
}

type writer struct {
	pathProvider pathutil.PathProvider
	fileManager  fileutil.FileManager
	pathChecker  pathutil.PathChecker
	pathModifier pathutil.PathModifier
}

// NewWriter ...
func NewWriter(pathProvider pathutil.PathProvider, fileManager fileutil.FileManager, pathChecker pathutil.PathChecker, pathModifier pathutil.PathModifier) Writer {
	return &writer{pathProvider: pathProvider, fileManager: fileManager, pathChecker: pathChecker, pathModifier: pathModifier}
}

// Write writes the contents of input into a xcconfig file if
// the provided content is not already a path to xcconfig file.
// If the content is a valid path to xcconfig, it will validate the path,
// and return the path. It returns error if it cannot finalize a xcconfig
// file and/or its path.
func (w writer) Write(input string) (string, error) {
	if w.isPath(input) {
		xcconfigPath, err := w.pathModifier.AbsPath(input)
		if err != nil {
			return "", fmt.Errorf("failed to convert xcconfig file path (%s) to absolute path: %w", input, err)
		}

		pathExists, err := w.pathChecker.IsPathExists(xcconfigPath)
		if err != nil {
			return "", err
		}
		if !pathExists {
			return "", fmt.Errorf("provided xcconfig file path doesn't exist: %s", input)
		}
		return xcconfigPath, nil
	}

	dir, err := w.pathProvider.CreateTempDir("")
	if err != nil {
		return "", fmt.Errorf("unable to create temp dir for writing XCConfig: %v", err)
	}
	xcconfigPath := filepath.Join(dir, "temp.xcconfig")
	if err = w.fileManager.Write(xcconfigPath, input, 0644); err != nil {
		return "", fmt.Errorf("unable to write XCConfig content into file: %v", err)
	}
	return xcconfigPath, nil
}

func (w writer) isPath(input string) bool {
	return strings.HasSuffix(input, ".xcconfig")
}
