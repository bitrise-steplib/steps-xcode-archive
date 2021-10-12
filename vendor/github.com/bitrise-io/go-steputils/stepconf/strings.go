package stepconf

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/bitrise-io/go-utils/colorstring"
)

// Print the name of the struct with Title case in blue color with followed by a newline,
// then print all fields formatted as '- field name: field value` separated by newline.
func Print(config interface{}) {
	fmt.Print(toString(config))
}

func valueString(v reflect.Value) string {
	if v.Kind() != reflect.Ptr {
		return fmt.Sprintf("%v", v.Interface())
	}

	if !v.IsNil() {
		return fmt.Sprintf("%v", v.Elem().Interface())
	}

	return ""
}

// returns the name of the struct with Title case in blue color followed by a newline,
// then print all fields formatted as '- field name: field value` separated by newline.
func toString(config interface{}) string {
	v := reflect.ValueOf(config)
	t := reflect.TypeOf(config)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	str := fmt.Sprint(colorstring.Bluef("%s:\n", strings.Title(t.Name())))
	for i := 0; i < t.NumField(); i++ {
		str += fmt.Sprintf("- %s: %s\n", t.Field(i).Name, valueString(v.Field(i)))
	}

	return str
}
