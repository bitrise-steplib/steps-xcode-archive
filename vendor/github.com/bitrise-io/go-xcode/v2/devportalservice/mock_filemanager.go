package devportalservice

import (
	"io"
	"os"
	"strings"

	"github.com/bitrise-io/go-utils/v2/fileutil"
)

type mockFileReader struct {
	contents string
}

func newMockFileReader(contents string) fileutil.FileManager {
	return &mockFileReader{
		contents: contents,
	}
}

// Open ...
func (r *mockFileReader) Open(path string) (*os.File, error) {
	panic("not implemented")
}

// OpenReaderIfExists ...
func (r *mockFileReader) OpenReaderIfExists(path string) (io.Reader, error) {
	return io.NopCloser(strings.NewReader(r.contents)), nil
}

// ReadDirEntryNames ...
func (r *mockFileReader) ReadDirEntryNames(path string) ([]string, error) {
	panic("not implemented")
}

// Remove ...
func (r *mockFileReader) Remove(path string) error {
	panic("not implemented")
}

// RemoveAll ...
func (r *mockFileReader) RemoveAll(path string) error {
	panic("not implemented")
}

// Write ...
func (r *mockFileReader) Write(path string, value string, perm os.FileMode) error {
	panic("not implemented")
}

// WriteBytes ...
func (r *mockFileReader) WriteBytes(path string, value []byte) error {
	panic("not implemented")
}

// FileSizeInBytes ...
func (r *mockFileReader) FileSizeInBytes(pth string) (int64, error) {
	panic("not implemented")
}
