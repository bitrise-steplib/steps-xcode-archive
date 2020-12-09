package input

import (
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/pathutil"
)

const (
	fileSchema = "file://"
)

// FileDownloader ..
type FileDownloader interface {
	Get(destination, source string) error
}

// FileProvider supports retrieving the local path to a file either provided
// as a local path using `file://` scheme
// or downloading the file to a temporary location and return the path to it.
type FileProvider struct {
	filedownloader FileDownloader
}

// NewFileProvider ...
func NewFileProvider(filedownloader FileDownloader) FileProvider {
	return FileProvider{
		filedownloader: filedownloader,
	}
}

// LocalPath ...
func (fileProvider FileProvider) LocalPath(path string) (string, error) {

	var localPath string
	if strings.HasPrefix(path, fileSchema) {
		trimmedPath, err := fileProvider.trimmedFilePath(path)
		if err != nil {
			return "", err
		}
		localPath = trimmedPath
	} else {
		downloadedPath, err := fileProvider.downloadFile(path)
		if err != nil {
			return "", err
		}
		localPath = downloadedPath
	}

	return localPath, nil
}

// Removes file:// from the begining of the path
func (fileProvider FileProvider) trimmedFilePath(path string) (string, error) {
	pth := strings.TrimPrefix(path, fileSchema)
	return pathutil.AbsPath(pth)
}

func (fileProvider FileProvider) downloadFile(url string) (string, error) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("FileProviderprovider")
	if err != nil {
		return "", err
	}

	fileName, err := fileProvider.fileNameFromPathURL(url)
	if err != nil {
		return "", err
	}
	localPath := path.Join(tmpDir, fileName)
	if err := fileProvider.filedownloader.Get(localPath, url); err != nil {
		return "", err
	}

	return localPath, nil
}

// Returns the file's name from a URL that starts with
// `http://` or `https://`
func (fileProvider FileProvider) fileNameFromPathURL(urlPath string) (string, error) {
	url, err := url.Parse(urlPath)
	if err != nil {
		return "", err
	}

	return filepath.Base(url.Path), nil
}
