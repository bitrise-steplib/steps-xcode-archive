// Package serialized provides typed, map-like accessors over the loosely-typed
// maps produced when decoding Xcode's plist and pbxproj object graphs. A
// serialized.Object wraps a map[string]any and lets callers read nested values
// by key using the comma-ok idiom: each accessor returns the value and a bool
// that reports whether a usable value of the requested type was present.
//
// A false result means either the key was absent or the stored value was not
// of the requested type. Callers that need to tell those two cases apart can
// check Has first.
package serialized

// Object is a decoded plist/pbxproj node: a set of string keys mapped to
// arbitrary values.
type Object map[string]any

// Get returns the value stored at key in o as type T via a direct type
// assertion. ok is false — and value is the zero value of T — if key is absent
// or the stored value's concrete type is not exactly T.
//
// Because it asserts rather than converts, T must match the concrete type the
// decoder produced. For plist/JSON input that is a scalar: string, bool, int64
// or float64. Get does NOT handle the two composite cases that require
// conversion, so use the dedicated accessors for those:
//   - nested dictionaries decode as map[string]any, not Object — use Object.
//   - arrays decode as []any, not []string — use StringSlice.
func Get[T any](o Object, key string) (value T, ok bool) {
	raw, exists := o[key]
	if !exists {
		return value, false
	}
	value, ok = raw.(T)
	return value, ok
}

// Keys returns the object's keys in unspecified order.
func (o Object) Keys() []string {
	keys := make([]string, 0, len(o))
	for key := range o {
		keys = append(keys, key)
	}
	return keys
}

// Has reports whether key is present, regardless of the stored value's type.
func (o Object) Has(key string) bool {
	_, ok := o[key]
	return ok
}

// Value returns the raw value stored at key. ok mirrors Go map semantics: it is
// false only when key is absent.
func (o Object) Value(key string) (value any, ok bool) {
	value, ok = o[key]
	return value, ok
}

// Bool returns the bool stored at key. ok is false if key is absent or the
// value is not a bool.
func (o Object) Bool(key string) (bool, bool) {
	return Get[bool](o, key)
}

// String returns the string stored at key. ok is false if key is absent or the
// value is not a string.
func (o Object) String(key string) (string, bool) {
	return Get[string](o, key)
}

// Int64 returns the int64 stored at key. ok is false if key is absent or the
// value is not an int64.
func (o Object) Int64(key string) (int64, bool) {
	return Get[int64](o, key)
}

// StringSlice returns the []string stored at key. The value must be a []any
// whose every element is a string. ok is false if key is absent, the value is
// not a []any, or any element is not a string.
func (o Object) StringSlice(key string) ([]string, bool) {
	casted, ok := Get[[]any](o, key)
	if !ok {
		return nil, false
	}

	slice := make([]string, 0, len(casted))
	for _, v := range casted {
		item, ok := v.(string)
		if !ok {
			return nil, false
		}
		slice = append(slice, item)
	}

	return slice, true
}

// Object returns the nested Object stored at key. The value must be a
// map[string]any. ok is false if key is absent or the value is not a map.
func (o Object) Object(key string) (Object, bool) {
	casted, ok := Get[map[string]any](o, key)
	if !ok {
		return nil, false
	}
	return casted, true
}
