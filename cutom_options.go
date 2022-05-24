package main

import "github.com/bitrise-io/go-utils/sliceutil"

func xcodebuildCustomOptions(platform string, customOptions []string) []string {
	destination := "generic/platform=" + platform
	destinationOptions := []string{"-destination", destination}

	var options []string
	if len(customOptions) != 0 {
		if !sliceutil.IsStringInSlice("-destination", customOptions) {
			options = append(options, destinationOptions...)
		}

		options = append(options, customOptions...)
	} else {
		options = append(options, destinationOptions...)
	}

	return options
}
