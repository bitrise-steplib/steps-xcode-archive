package logio

import (
	"bytes"
	"errors"
	"fmt"
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

	closer func() error
}

// Close closes the PipeWiring instances that needs to be closing as part of this instance.
//
// In reality it can only close the filter and the tool input as everything else is
// managed by a command or the os.
func (i *PipeWiring) Close() error {
	return i.closer()
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
		closer: func() error {
			// XcbuildRawout - no need to close
			// XcbuildStdout - Multiwriter, meaning we need to close the subwriters
			// XcbuildStderr - Multiwriter, meaning we need to close the subwriters
			// ToolStdout - We are not closing stdout
			// ToolSterr - We are not closing stderr

			var errStr string

			if err := bitrisePrefixFilter.Close(); err != nil {
				errStr += fmt.Sprintf("failed to close log filter, error: %s", err.Error())
			}
			if err := toolPipeW.Close(); err != nil {
				if len(errStr) > 0 {
					errStr += ", "
				}
				errStr += fmt.Sprintf("failed to close xcodebuild-xcpretty pipe, error: %s", err.Error())
			}

			if len(errStr) > 0 {
				return errors.New(errStr)
			}

			return nil
		},
	}
}
