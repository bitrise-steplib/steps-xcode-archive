package httputil

import (
	"net/http"
	"net/http/httputil"

	"github.com/bitrise-io/go-utils/log"
)

// PrintRequest ...
func PrintRequest(request *http.Request) error {
	if request == nil {
		return nil
	}

	dump, err := httputil.DumpRequest(request, true)
	if err != nil {
		return err
	}

	log.Debugf("%s", dump)

	return nil
}

// PrintResponse ...
func PrintResponse(response *http.Response) error {
	if response == nil {
		return nil
	}

	dump, err := httputil.DumpResponse(response, true)
	if err != nil {
		return err
	}
	log.Debugf("%s", dump)

	return nil
}

// IsUserFixable returns true if statusCode is a value
// that is deemed retryable, i.e. something that could
// be fixed by the user.
func IsUserFixable(statusCode int) bool {
	return statusCode == 400 || statusCode == 401
}
