package logio

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"sync"
)

// PrefixFilter intercept writes: when the message has a prefix that matches a
// regexp it writes into the `Matching` sink, otherwise to the `Filtered` sink.
//
// Note: Callers are responsible for closing `Matching` and `Filtered` Sinks
type PrefixFilter struct {
	prefixRegexp *regexp.Regexp

	// internal buffered middleman between xcbuild and scan
	filterInput bufio.ReadWriter
	pipeW       *io.PipeWriter

	Matching *Sink
	Filtered io.Writer

	// closing
	closeOnce sync.Once

	done         chan struct{}
	messageLost  chan error
	scannerError chan error
}

// Done returns a channel on which the user can observe when the last messages are
// written to the outputs. The channel has a buffer of one to prevent early unreceived
// messages or late subscriptions to the channel.
func (p *PrefixFilter) Done() <-chan struct{} { return p.done }

// MessageLost returns a channel on which the user can observe if there were
// messages lost. The channel has a buffer of one to prevent early unreceived
// messages or late subscriptions to the channel.
func (p *PrefixFilter) MessageLost() <-chan error { return p.messageLost }

// ScannerError returns a channel on which the user can observe if there were
// any scanner errors. The channel has a buffer of one to prevent early unreceived
// messages or late subscriptions to the channel.
func (p *PrefixFilter) ScannerError() <-chan error { return p.scannerError }

// NewPrefixFilter returns a new PrefixFilter. Writes are based on line prefix.
//
// Note: Callers are responsible for closing intercepted and target writers that implement io.Closer
func NewPrefixFilter(prefixRegexp *regexp.Regexp, matching *Sink, filtered io.Writer) *PrefixFilter {
	// This is the backing field of the bufio.ReadWriter
	pipeR, pipeW := io.Pipe()
	messageLost := make(chan error, 1)
	done := make(chan struct{}, 1)
	scannerError := make(chan error, 1)

	filter := &PrefixFilter{
		prefixRegexp: prefixRegexp,
		filterInput:  *bufio.NewReadWriter(bufio.NewReader(pipeR), bufio.NewWriter(pipeW)),
		pipeW:        pipeW,
		closeOnce:    sync.Once{},
		messageLost:  messageLost,
		done:         done,
		scannerError: scannerError,

		Matching: matching,
		Filtered: filtered,
	}
	go filter.run()
	return filter
}

// Write implements io.Writer. It writes into an internal pipe which the interceptor goroutine consumes.
func (p *PrefixFilter) Write(data []byte) (int, error) {
	return p.filterInput.Write(data)
}

// Close stops the interceptor and closes the pipe.
func (p *PrefixFilter) Close() error {
	var errString string
	p.closeOnce.Do(func() {
		// Flush and close scanner input
		if err := p.filterInput.Flush(); err != nil {
			errString += fmt.Sprintf("failed to flush xcbuildoutput (%v)", err.Error())
		}
		if err := p.pipeW.Close(); err != nil {
			if len(errString) > 0 {
				errString += ", "
			}
			errString += fmt.Sprintf("failed to close scanner input (%v)", err.Error())
		}
	})

	if len(errString) > 0 {
		return fmt.Errorf("failed to close prefixFilter: %s", errString)
	}

	return nil
}

// run reads lines (and partial final chunk) and writes them.
func (p *PrefixFilter) run() {
	defer func() {
		// Signal done and close signaling channels
		p.done <- struct{}{}
		close(p.done)
		close(p.messageLost)
		close(p.scannerError)
	}()

	// Use a scanner but with a large buffer to handle long lines.
	scanner := bufio.NewScanner(p.filterInput)
	const maxTokenSize = 10 * 1024 * 1024
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, maxTokenSize)

	for scanner.Scan() {
		line := scanner.Text() // note: newline removed
		// re-append newline to preserve same output format
		logLine := line + "\n"

		if p.prefixRegexp.MatchString(line) {
			if _, err := p.Matching.Write([]byte(logLine)); err != nil {
				p.messageLost <- fmt.Errorf("intercepting message: %w", err)
			}
		} else {
			if _, err := p.Filtered.Write([]byte(logLine)); err != nil {
				p.messageLost <- fmt.Errorf("intercepting message: %w", err)
			}
		}
	}

	// handle any scanner error
	if err := scanner.Err(); err != nil {
		p.scannerError <- err
	}
}
