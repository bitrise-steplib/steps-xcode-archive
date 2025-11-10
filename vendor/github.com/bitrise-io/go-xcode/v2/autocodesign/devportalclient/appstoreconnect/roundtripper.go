package appstoreconnect

import (
	"context"
	"net/http"
	"time"
)

// trackingRoundTripper wraps an http.RoundTripper and tracks metrics for each HTTP attempt,
// including retries. It measures per-attempt duration without retry wait times included,
// allowing accurate tracking even when requests are retried due to rate limits or other errors.
type trackingRoundTripper struct {
	wrapped http.RoundTripper
	tracker Tracker
}

// isRetryContextKey is used to store whether a request is a retry attempt in the request context.
type isRetryContextKey struct{}

func newTrackingRoundTripper(wrapped http.RoundTripper, tracker Tracker) *trackingRoundTripper {
	if wrapped == nil {
		wrapped = http.DefaultTransport
	}
	return &trackingRoundTripper{
		wrapped: wrapped,
		tracker: tracker,
	}
}

// markAsRetry stores a flag in the request context indicating this is a retry attempt.
// It returns a new request with the updated context. This approach avoids shared state
// between concurrent requests (e.g., multiple POSTs to the same endpoint with different bodies).
func (t *trackingRoundTripper) markAsRetry(req *http.Request) *http.Request {
	ctx := context.WithValue(req.Context(), isRetryContextKey{}, true)
	return req.WithContext(ctx)
}

// RoundTrip executes an HTTP request and tracks its duration and retry status.
// Each HTTP attempt (including retries) generates a separate metric event, allowing
// accurate alerting based on individual response times rather than aggregate times
// that include retry backoff delays.
func (t *trackingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Check if this request was marked as a retry by RequestLogHook
	isRetry := req.Context().Value(isRetryContextKey{}) != nil

	startTime := time.Now()
	resp, err := t.wrapped.RoundTrip(req)
	duration := time.Since(startTime)

	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}

	// Track this attempt with its actual duration (no retry waits included)
	t.tracker.TrackAPIRequest(req.Method, req.URL.Host, req.URL.Path, statusCode, duration, isRetry)

	return resp, err
}
