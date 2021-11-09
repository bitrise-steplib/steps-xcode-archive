package appstoreconnect

import (
	"strings"
	"time"
)

// Time ...
type Time time.Time

// UnmarshalJSON ...
func (t *Time) UnmarshalJSON(b []byte) error {
	timeStr := strings.Trim(string(b), `"`)
	parsed, err := time.Parse("2006-01-02T15:04:05.000-0700", timeStr)
	if err != nil {
		return err
	}
	*t = Time(parsed)
	return nil
}
