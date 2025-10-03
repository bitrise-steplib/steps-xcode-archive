package logio

import (
	"bytes"
	"io"
	"os"
	"regexp"
)

// PipeWiring is a helper struct to define the setup and binding of tools and
// xcbuild with a filter and stdout. It is purely boilerplate reduction and it is the
// users responsibility to choose between this and manual hooking of the in/outputs.
// It also provides a convenient Close() method that only closes things that can/should be closed.
type PipeWiring struct {
	XcbuildRawout bytes.Buffer
	XcbuildStdout io.Writer
	XcbuildStderr io.Writer
	ToolStdin     io.ReadCloser
	ToolStdout    io.WriteCloser
	ToolStderr    io.WriteCloser

	toolPipeW *io.PipeWriter
	filter    *PrefixFilter
}

// CloseToolInput...
func (p *PipeWiring) CloseToolInput() error {
	return p.toolPipeW.Close()
}

// CloseFilter...
func (p *PipeWiring) CloseFilter() error {
	return p.filter.Close()
}

// SetupPipeWiring creates a new PipeWiring instance that contains the usual
// input/outputs that an xcodebuild command and a logging tool needs when we are also
// using a logging filter.
func SetupPipeWiring(filter *regexp.Regexp) *PipeWiring {
	// Create a buffer to store raw xcbuild output
	var rawXcbuild bytes.Buffer
	// Pipe filtered logs to tool
	toolPipeR, toolPipeW := io.Pipe()

	// Add a buffer before stdout
	bufferedStdout := NewSink(os.Stdout)
	// Add a buffer before tool input
	xcbuildLogs := NewSink(toolPipeW)
	// Create a filter for [Bitrise ...] prefixes
	bitrisePrefixFilter := NewPrefixFilter(
		filter,
		bufferedStdout,
		xcbuildLogs,
	)

	// Send raw xcbuild out to raw out and filter
	rawInputDuplication := io.MultiWriter(&rawXcbuild, bitrisePrefixFilter)

	return &PipeWiring{
		XcbuildRawout: rawXcbuild,
		XcbuildStdout: rawInputDuplication,
		XcbuildStderr: rawInputDuplication,
		ToolStdin:     toolPipeR,
		ToolStdout:    os.Stdout,
		ToolStderr:    os.Stderr,

		toolPipeW: toolPipeW,
		filter:    bitrisePrefixFilter,
	}
}
