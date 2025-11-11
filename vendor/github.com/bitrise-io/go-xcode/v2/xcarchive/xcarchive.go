package xcarchive

import (
	"path/filepath"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
)

// ArchiveReader ...
type ArchiveReader struct {
	pathChecker pathutil.PathChecker
	logger      log.Logger
}

// NewArchiveReader ...
func NewArchiveReader(pathChecker pathutil.PathChecker, logger log.Logger) ArchiveReader {
	return ArchiveReader{
		pathChecker: pathChecker,
		logger:      logger,
	}
}

// IsMacOS try to find the Contents dir under the .app/.
// If its finds it the archive is macOS. If it does not the archive is iOS.
func (r ArchiveReader) IsMacOS(archPath string) (bool, error) {
	r.logger.Debugf("Checking archive is MacOS or iOS")
	infoPlistPath := filepath.Join(archPath, "Info.plist")

	plist, err := newPlistDataFromFile(infoPlistPath)
	if err != nil {
		return false, err
	}

	appProperties, found := plist.GetMapStringInterface("ApplicationProperties")
	if !found {
		return false, err
	}

	applicationPath, found := appProperties.GetString("ApplicationPath")
	if !found {
		return false, err
	}

	applicationPath = filepath.Join(archPath, "Products", applicationPath)
	contentsPath := filepath.Join(applicationPath, "Contents")

	exist, err := r.pathChecker.IsDirExists(contentsPath)
	if err != nil {
		return false, err
	}

	return exist, nil
}
