package stepconf

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/v2/filedownloader"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/pathutil"
)

const (
	fileScheme = "file://"
)

// FileProvider supports retrieving the local path to a file either provided
// as a local path using `file://` scheme or downloading the file to a
// temporary location and returning the path to it.
// Downloads use automatic retry logic via the filedownloader package.
type FileProvider interface {
	// LocalPath returns the local file path for the given path.
	// If the path uses the file:// scheme, it strips the scheme and returns the absolute path.
	// If the path is a remote URL (http:// or https://), it downloads the file to a
	// temporary directory and returns the local path to the downloaded file.
	LocalPath(ctx context.Context, path string) (string, error)

	// Contents returns a streaming reader for the file contents.
	// If the path uses the file:// scheme, it opens the local file.
	// If the path is a remote URL (http:// or https://), it fetches the remote content.
	// The caller is responsible for closing the returned io.ReadCloser.
	Contents(ctx context.Context, srcPath string) (io.ReadCloser, error)
}

type fileProvider struct {
	downloader   filedownloader.Downloader
	fileManager  fileutil.FileManager
	pathProvider pathutil.PathProvider
	pathModifier pathutil.PathModifier
}

// NewFileProvider ...
func NewFileProvider(downloader filedownloader.Downloader, fileManager fileutil.FileManager, pathProvider pathutil.PathProvider, pathModifier pathutil.PathModifier) FileProvider {
	return &fileProvider{
		downloader:   downloader,
		fileManager:  fileManager,
		pathProvider: pathProvider,
		pathModifier: pathModifier,
	}
}

// LocalPath returns the local file path for the given path.
// If the path uses the file:// scheme, it strips the scheme and returns the absolute path.
// If the path is a remote URL (http:// or https://), it downloads the file to a
// temporary directory and returns the local path to the downloaded file.
func (f *fileProvider) LocalPath(ctx context.Context, path string) (string, error) {
	if strings.HasPrefix(path, fileScheme) {
		return f.trimmedFilePath(path)
	}

	return f.downloadFileToLocalPath(ctx, path)
}

// Contents returns a streaming reader for the file contents.
// If the path uses the file:// scheme, it opens the local file.
// If the path is a remote URL (http:// or https://), it fetches the remote content.
// The caller is responsible for closing the returned io.ReadCloser.
func (f *fileProvider) Contents(ctx context.Context, srcPath string) (io.ReadCloser, error) {
	if strings.HasPrefix(srcPath, fileScheme) {
		trimmedPath, err := f.trimmedFilePath(srcPath)
		if err != nil {
			return nil, err
		}

		return f.fileManager.Open(trimmedPath)
	}

	return f.downloader.Get(ctx, srcPath)
}

// trimmedFilePath removes the file:// prefix from the path and returns the absolute path.
func (f *fileProvider) trimmedFilePath(path string) (string, error) {
	pth := strings.TrimPrefix(path, fileScheme)
	return f.pathModifier.AbsPath(pth)
}

// downloadFileToLocalPath downloads a remote file to a temporary directory
// and returns the local path to the downloaded file.
func (f *fileProvider) downloadFileToLocalPath(ctx context.Context, urlPath string) (string, error) {
	tmpDir, err := f.pathProvider.CreateTempDir("FileProvider")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	fileName, err := f.fileNameFromURL(urlPath)
	if err != nil {
		return "", fmt.Errorf("failed to extract filename from URL %s: %w", urlPath, err)
	}

	localPath := filepath.Join(tmpDir, fileName)
	if err := f.downloader.Download(ctx, localPath, urlPath); err != nil {
		return "", fmt.Errorf("failed to download file from %s: %w", urlPath, err)
	}

	return localPath, nil
}

// fileNameFromURL extracts the filename from a URL path.
func (f *fileProvider) fileNameFromURL(urlPath string) (string, error) {
	parsedURL, err := url.Parse(urlPath)
	if err != nil {
		return "", err
	}

	return filepath.Base(parsedURL.Path), nil
}
