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
	XcbuildRawout *bytes.Buffer
	XcbuildStdout io.Writer
	XcbuildStderr io.Writer
	ToolStdin     io.ReadCloser
	ToolStdout    io.WriteCloser
	ToolStderr    io.WriteCloser

	toolPipeW      *io.PipeWriter
	bufferedStdout *Sink
	toolInSink     *Sink
	filter         *PrefixFilter
}

// Close ...
func (p *PipeWiring) Close() error {
	err := p.filter.Close()
	<-p.filter.Done()
	_ = p.toolInSink.Close()
	_ = p.toolPipeW.Close()
	_ = p.bufferedStdout.Close()

	return err
}

// SetupPipeWiring creates a new PipeWiring instance that contains the usual
// input/outputs that an xcodebuild command and a logging tool needs when we are also
// using a logging filter.
func SetupPipeWiring(filter *regexp.Regexp) *PipeWiring {
	// Create a buffer to store raw xcbuild output
	rawXcbuild := bytes.NewBuffer(nil)
	// Pipe filtered logs to tool
	toolPipeR, toolPipeW := io.Pipe()

	// Add a buffer before stdout
	bufferedStdout := NewSink(os.Stdout)
	// Add a buffer before tool input
	toolInSink := NewSink(toolPipeW)
	xcbuildLogs := io.MultiWriter(rawXcbuild, toolInSink)
	// Create a filter for [Bitrise ...] prefixes
	bitrisePrefixFilter := NewPrefixFilter(
		filter,
		bufferedStdout,
		xcbuildLogs,
	)

	return &PipeWiring{
		XcbuildRawout: rawXcbuild,
		XcbuildStdout: bitrisePrefixFilter,
		XcbuildStderr: bitrisePrefixFilter,
		ToolStdin:     toolPipeR,
		ToolStdout:    os.Stdout,
		ToolStderr:    os.Stderr,

		toolPipeW:      toolPipeW,
		bufferedStdout: bufferedStdout,
		toolInSink:     toolInSink,
		filter:         bitrisePrefixFilter,
	}
}
