package logio

import (
	"io"
	"time"

	"github.com/globocom/go-buffer/v2"
)

// Sink is an io.WriteCloser that uses a bufio.Writer to wrap the downstream and
// default buffer sizes and the regular flushing of the buffer for convenience.
type Sink struct {
	io.WriteCloser
	bufferedWriter bufferedWriter
	err            chan error
}

// NewSink creates a new Sink instance
func NewSink(downstream io.Writer) *Sink {
	errors := make(chan error, 10)

	return &Sink{
		bufferedWriter: buffer.New(
			// Flush after five writes
			buffer.WithSize(5),
			// Flushed every second if not full
			buffer.WithFlushInterval(time.Second),
			// Flush writes to downstream
			buffer.WithFlusher(buffer.FlusherFunc(func(items []interface{}) {
				for _, item := range items {
					_, err := downstream.Write(item.([]byte))

					select {
					case errors <- err:
					default:
					}
				}
			})),
		),
		err: errors,
	}
}

// Errors is a receive only channel where the sink can communicate
// errors happened on sending, should the user be interested in them
func (s *Sink) Errors() <-chan error {
	return s.err
}

// Write conformance
func (s *Sink) Write(p []byte) (int, error) {
	return len(p), s.bufferedWriter.Push(p)
}

// Close conformance
func (s *Sink) Close() error {
	return s.bufferedWriter.Close()
}

type bufferedWriter interface {
	Push(item any) error
	Close() error
}
