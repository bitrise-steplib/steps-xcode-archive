package stepconf

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/bitrise-io/go-utils/v2/log/colorstring"
)

// Print the name of the struct with Title case in blue color with followed by a newline,
// then print all fields formatted as `- field name: field value` separated by newline.
func Print(config interface{}) {
	fmt.Print(toString(config))
}

func valueString(v reflect.Value) string {
	if v.Kind() != reflect.Ptr {
		if v.Kind() == reflect.String && v.Len() == 0 {
			return "<unset>"
		}
		return fmt.Sprintf("%v", v.Interface())
	}

	if !v.IsNil() {
		return fmt.Sprintf("%v", v.Elem().Interface())
	}

	return ""
}

// returns the name of the struct with Title case in blue color followed by a newline,
// then print all fields formatted as `- field name: field value` separated by newline.
func toString(config interface{}) string {
	v := reflect.ValueOf(config)
	t := reflect.TypeOf(config)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	configName := strings.Title(t.Name()) //nolint:staticcheck
	// It's not worth pulling the heavy /x/text lib for this simple case, string.Title() can handle the struct name
	str := fmt.Sprint(colorstring.Bluef("%s:\n", configName))
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		var key, _ = parseTag(field.Tag.Get("env"))
		if key == "" {
			key = field.Name
		}
		str += fmt.Sprintf("- %s: %s\n", key, valueString(v.Field(i)))
	}

	return str
}
