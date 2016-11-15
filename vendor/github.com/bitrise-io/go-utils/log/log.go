package log

import (
	"fmt"

	"github.com/bitrise-io/go-utils/colorstring"
)

// Error ...
func Error(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	fmt.Println(colorstring.Red(message))
}

// Warn ...
func Warn(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	fmt.Println(colorstring.Yellow(message))
}

// Info ...
func Info(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	fmt.Println(colorstring.Blue(message))
}

// Detail ...
func Detail(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	fmt.Println(message)
}

// Done ...
func Done(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	fmt.Println(colorstring.Green(message))
}
