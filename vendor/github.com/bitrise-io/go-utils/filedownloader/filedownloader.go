package filedownloader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/bitrise-io/go-utils/log"
)

// HTTPClient ...
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// FileDownloader ...
type FileDownloader struct {
	client  HTTPClient
	context context.Context
}

// New ...
func New(client HTTPClient) FileDownloader {
	return FileDownloader{
		client: client,
	}
}

// NewWithContext ...
func NewWithContext(context context.Context, client HTTPClient) FileDownloader {
	return FileDownloader{
		client:  client,
		context: context,
	}
}

// GetWithFallback downloads a file from a given source. Provided destination should be a file that does not exist.
// You can specify fallback sources which will be used in order if downloading fails from either source.
func (downloader FileDownloader) GetWithFallback(destination, source string, fallbackSources ...string) error {
	sources := append([]string{source}, fallbackSources...)
	for _, source := range sources {
		err := downloader.Get(destination, source)
		if err != nil {
			log.Warnf("Could not download file (%s): %s", source, err)
		} else {
			return nil
		}
	}

	return fmt.Errorf("None of the sources returned 200 OK status")
}

// Get downloads a file from a given source. Provided destination should be a file that does not exist.
func (downloader FileDownloader) Get(destination, source string) error {
	f, err := os.Create(destination)
	if err != nil {
		return err
	}

	defer func() {
		if err := f.Close(); err != nil {
			log.Errorf("Failed to close file, error: %s", err)
		}
	}()

	return download(downloader.context, downloader.client, source, f)
}

// GetRemoteContents fetches a remote URL contents
func (downloader FileDownloader) GetRemoteContents(URL string) ([]byte, error) {
	var buffer bytes.Buffer
	if err := download(downloader.context, downloader.client, URL, &buffer); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// ReadLocalFile returns a local file contents
func (downloader FileDownloader) ReadLocalFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func download(context context.Context, client HTTPClient, source string, destination io.Writer) error {
	req, err := http.NewRequest(http.MethodGet, source, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %s", err)
	}

	if context != nil {
		req = req.WithContext(context)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		if resp.Body != nil {
			if err := resp.Body.Close(); err != nil {
				log.Errorf("Failed to close body, error: %s", err)
			}
		}
	}()

	if resp.StatusCode != http.StatusOK {
		responseBytes, err := httputil.DumpResponse(resp, true)
		if err == nil {
			return fmt.Errorf("unable to download file from: %s. Status code: %d. Response: %s", source, resp.StatusCode, string(responseBytes))
		}
		return fmt.Errorf("unable to download file from: %s. Status code: %d", source, resp.StatusCode)
	}

	if _, err = io.Copy(destination, resp.Body); err != nil {
		return err
	}

	return nil
}
