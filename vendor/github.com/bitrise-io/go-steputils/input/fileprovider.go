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
	GetRemoteContents(source string) ([]byte, error)
	ReadLocalFile(path string) ([]byte, error)
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
	if strings.HasPrefix(path, fileSchema) { // Local file
		return fileProvider.trimmedFilePath(path)
	}

	return fileProvider.downloadFileToLocalPath(path)
}

// Contents returns the contents of remote or local URL
func (fileProvider FileProvider) Contents(srcPath string) ([]byte, error) {
	if strings.HasPrefix(srcPath, fileSchema) { // Local file
		trimmedPath, err := fileProvider.trimmedFilePath(srcPath)
		if err != nil {
			return nil, err
		}

		return fileProvider.filedownloader.ReadLocalFile(trimmedPath)
	}

	return fileProvider.filedownloader.GetRemoteContents(srcPath)
}

// Removes file:// from the begining of the path
func (fileProvider FileProvider) trimmedFilePath(path string) (string, error) {
	pth := strings.TrimPrefix(path, fileSchema)
	return pathutil.AbsPath(pth)
}

func (fileProvider FileProvider) downloadFileToLocalPath(url string) (string, error) {
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
