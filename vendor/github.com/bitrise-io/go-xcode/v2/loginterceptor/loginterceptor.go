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
//
// Note: Callers are responsible for closing intercepted and target writers that implement io.Closer
type PrefixInterceptor struct {
	prefixRegexp  *regexp.Regexp
	targetCh      chan string
	interceptedCh chan string
	logger        log.Logger

	// internal pipe and goroutine to scan and route
	internalReader *io.PipeReader
	internalWriter *io.PipeWriter

	// closing
	closeOnce sync.Once
	closeErr  error

	// TargetDelivered receives a single value when the target goroutine
	// finishes processing all messages. Callers should consume from this
	// channel to avoid goroutine leaks, or use Close() and ignore it.
	TargetDelivered <-chan struct{}
	// InterceptedDelivered receives a single value when the intercepted goroutine
	// finishes processing all messages. Callers should consume from this
	// channel to avoid goroutine leaks, or use Close() and ignore it.
	InterceptedDelivered <-chan struct{}
}

// NewPrefixInterceptor returns an io.WriteCloser. Writes are based on line prefix.
//
// Note: Callers are responsible for closing intercepted and target writers that implement io.Closer
func NewPrefixInterceptor(prefixRegexp *regexp.Regexp, intercepted, target io.Writer, logger log.Logger) *PrefixInterceptor {
	pipeReader, pipeWriter := io.Pipe()

	targetCh := make(chan string, 10000)
	targetDoneCh := make(chan struct{}, 1)
	interceptedCh := make(chan string, 10000)
	interceptedDoneCh := make(chan struct{}, 1)

	go sendingTo(targetCh, targetDoneCh, target, logger)
	go sendingTo(interceptedCh, interceptedDoneCh, intercepted, logger)

	interceptor := &PrefixInterceptor{
		prefixRegexp:         prefixRegexp,
		targetCh:             targetCh,
		interceptedCh:        interceptedCh,
		logger:               logger,
		internalReader:       pipeReader,
		internalWriter:       pipeWriter,
		TargetDelivered:      targetDoneCh,
		InterceptedDelivered: interceptedDoneCh,
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
	if err := i.internalReader.Close(); err != nil {
		i.logger.Errorf("internal reader: %v", err)
	}
	close(i.targetCh)
	close(i.interceptedCh)
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
		logLine := line + "\n"

		select {
		case i.targetCh <- logLine:
		default:
			i.logger.Debugf("target channel full, dropping message")
		}

		if i.prefixRegexp.MatchString(line) {
			select {
			case i.interceptedCh <- logLine:
			default:
				i.logger.Debugf("intercepted channel full, dropping message")
			}
		}
	}

	// handle any scanner error
	if err := scanner.Err(); err != nil {
		i.logger.Errorf("router scanner error: %v\n", err)
	}
}

func sendingTo(
	srcCh <-chan string,
	done chan<- struct{},
	writer io.Writer,
	logger log.Logger,
) {
	for msg := range srcCh {
		if _, err := io.WriteString(writer, msg); err != nil {
			logger.Errorf(" writer error: %v", err)
		}
	}

	done <- struct{}{}
	close(done)
}
