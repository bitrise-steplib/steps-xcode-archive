package serialized

import (
	"errors"
	"fmt"
)

// ErrKeyNotFound is for callers that report a missing key as an error: they wrap it, and their
// callers tell a missing key from a failed read with errors.Is. The getters themselves return
// (value, ok) and never produce it.
var ErrKeyNotFound = errors.New("key not found")

// WrongTypeError reports a key whose value is not of the type the caller asked
// for. Value is the stored value, so the file it came from can be corrected.
type WrongTypeError struct {
	Key      string
	Value    any
	Expected string
}

func (e WrongTypeError) Error() string {
	return fmt.Sprintf("%s is not a %s: %#v (%T)", e.Key, e.Expected, e.Value, e.Value)
}
