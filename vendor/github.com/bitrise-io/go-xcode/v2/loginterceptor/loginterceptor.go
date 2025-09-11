package loginterceptor

import (
	"bufio"
	"io"
	"regexp"
	"sync"

	"github.com/bitrise-io/go-utils/v2/log"
)

// PrefixInterceptor intercept writes: if a line begins with prefix, it will be written to
// both writers. Partial writes without newline are buffered until a newline.
type PrefixInterceptor struct {
	prefixRegexp *regexp.Regexp
	intercepted  *NonBlockingWriter
	target       *NonBlockingWriter
	logger       log.Logger

	// internal pipe and goroutine to scan and route
	internalReader *io.PipeReader
	internalWriter *io.PipeWriter

	// close once
	closeOnce sync.Once
	closeErr  error
}

// NewPrefixInterceptor returns an io.WriteCloser. Writes are based on line prefix.
func NewPrefixInterceptor(prefixRegexp *regexp.Regexp, intercepted, target io.Writer, logger log.Logger) *PrefixInterceptor {
	pipeReader, pipeWriter := io.Pipe()
	interceptor := &PrefixInterceptor{
		prefixRegexp:   prefixRegexp,
		intercepted:    NewNonBlockingWriter(intercepted, logger),
		target:         NewNonBlockingWriter(target, logger),
		logger:         logger,
		internalReader: pipeReader,
		internalWriter: pipeWriter,
	}
	go interceptor.run()
	return interceptor
}

// Write implements io.Writer. It writes into an internal pipe which the interceptor goroutine consumes.
func (i *PrefixInterceptor) Write(p []byte) (int, error) {
	return i.internalWriter.Write(p)
}

// Close stops the interceptor and closes the pipe.
func (i *PrefixInterceptor) Close() error {
	i.closeOnce.Do(func() {
		i.closeErr = i.internalWriter.Close()
	})
	return i.closeErr
}

func (i *PrefixInterceptor) closeAfterRun() {
	if err := i.intercepted.Close(); err != nil {
		i.logger.Errorf("intercepted writer: %v", err)
	}
	if err := i.target.Close(); err != nil {
		i.logger.Errorf("target writer: %v", err)
	}
	if err := i.internalReader.Close(); err != nil {
		i.logger.Errorf("internal reader: %v", err)
	}
}

// run reads lines (and partial final chunk) and writes them.
func (i *PrefixInterceptor) run() {
	defer i.closeAfterRun()

	// Use a scanner but with a large buffer to handle long lines.
	scanner := bufio.NewScanner(i.internalReader)
	const maxTokenSize = 10 * 1024 * 1024
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, maxTokenSize)

	for scanner.Scan() {
		line := scanner.Text() // note: newline removed
		// re-append newline to preserve same output format
		outLine := line + "\n"

		// Write to intercepted channel if matching regexp
		if i.prefixRegexp.MatchString(line) {
			if _, err := io.WriteString(i.intercepted, outLine); err != nil {
				i.logger.Errorf("intercept writer error: %v", err)
			}
		}
		// Always write to target channel
		if _, err := io.WriteString(i.target, outLine); err != nil {
			i.logger.Errorf("writer error: %v", err)
		}
	}

	// handle any scanner error
	if err := scanner.Err(); err != nil {
		i.logger.Errorf("router scanner error: %v\n", err)
	}
}

// NonBlockingWriter is an io.Writer that writes to a wrapped io.Writer in a non-blocking way.
type NonBlockingWriter struct {
	channel chan []byte
	wrapped io.Writer
	logger  log.Logger
}

// NewNonBlockingWriter creates a new NonBlockingWriter.
func NewNonBlockingWriter(w io.Writer, logger log.Logger) *NonBlockingWriter {
	writer := &NonBlockingWriter{
		channel: make(chan []byte, 10000), // buffered channel to avoid blocking
		wrapped: w,
		logger:  logger,
	}
	go writer.Run()
	return writer
}

// Write implements io.Writer. It writes into an internal pipe which the interceptor goroutine consumes.
func (i *NonBlockingWriter) Write(p []byte) (int, error) {
	select {
	case i.channel <- p:
		return len(p), nil
	default:
		i.logger.Debugf("buffer full, dropping log")
		return 0, nil
	}
}

// Close stops the interceptor and closes the pipe.
func (i *NonBlockingWriter) Close() error {
	close(i.channel)
	return nil
}

// Run consumes the channel and writes to the wrapped writer.
func (i *NonBlockingWriter) Run() {
	for msg := range i.channel {
		if _, err := i.wrapped.Write(msg); err != nil {
			i.logger.Errorf("NonBlockingWriter: wrapped writer error: %v", err)
		}
	}

	if closer, ok := i.wrapped.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			i.logger.Errorf("NonBlockingWriter: closing wrapped writer: %v", err)
		}
	}
}
