package profileutil

import "time"

// TimeProvider ...
type TimeProvider interface {
	Now() time.Time
}

// DefaultTimeProvider ...
type DefaultTimeProvider struct{}

// Now ...
func (DefaultTimeProvider) Now() time.Time {
	return time.Now()
}
