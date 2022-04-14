package time

import (
	"fmt"
	"strings"
	"time"
)

// Time ...
type Time time.Time

// UnmarshalJSON ...
func (t *Time) UnmarshalJSON(b []byte) error {
	timeStr := strings.Trim(string(b), `"`)
	var errors []error

	for _, timeFormat := range timeFormats() {
		parsed, err := time.Parse(timeFormat, timeStr)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		*t = Time(parsed)
		return nil
	}

	return fmt.Errorf("%s", errors)
}

func timeFormats() []string {
	formats := []string{time.RFC3339}
	formats = append(formats, appleKeyAuthTimeFormats()...)
	formats = append(formats, appleIDAuthTimeFormats()...)

	return formats
}

func appleKeyAuthTimeFormats() []string {
	// Apple is using an ISO 8601 time format (https://en.wikipedia.org/wiki/ISO_8601). In this format the offset from
	// the UTC time can have the following equivalent and interchangeable formats:
	// * [+/-]07:00
	// * [+/-]0700
	// * [+/-]07
	// (* also if there is no UTC offset then [+0000, +00:00, +00] are the same as adding a Z after the seconds)
	//
	// Go has built in support for ISO 8601 but only for the zero offset UTC and the [+/-]07:00 format under time.RFC3339.
	// We still need to check for the other two.
	return []string{
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05.000-07",
	}
}

func appleIDAuthTimeFormats() []string {
	// Spaceship returns this time format when setting SPACESHIP_AVOID_XCODE_API=true. This is needed because Apple's
	// API started to return an error for the old spaceship implementation.
	return []string{
		"2006-01-02 15:04:05 UTC",
	}
}
