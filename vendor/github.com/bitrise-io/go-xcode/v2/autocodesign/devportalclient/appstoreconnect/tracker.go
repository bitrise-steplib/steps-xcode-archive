package appstoreconnect

import (
	"fmt"
	"time"

	"github.com/bitrise-io/go-utils/v2/analytics"
	"github.com/bitrise-io/go-utils/v2/env"
)

// Tracker defines the interface for tracking App Store Connect API usage and errors.
type Tracker interface {
	// TrackAPIRequest tracks one completed API request (even if it failed)
	TrackAPIRequest(method, host, endpoint string, statusCode int, duration time.Duration)

	// TrackAPIError tracks a failed API request with error details
	TrackAPIError(method, host, endpoint string, statusCode int, errorMessage string)

	// TrackAuthError tracks authentication-specific errors
	TrackAuthError(errorMessage string)
}

// NoOpAnalyticsTracker is a dummy implementation used in tests.
type NoOpAnalyticsTracker struct{}

// TrackAPIRequest tracks API requests (no-op implementation).
func (n NoOpAnalyticsTracker) TrackAPIRequest(method, host, endpoint string, statusCode int, duration time.Duration) {
}
// TrackAPIError tracks API errors (no-op implementation).
func (n NoOpAnalyticsTracker) TrackAPIError(method, host, endpoint string, statusCode int, errorMessage string) {
}
// TrackAuthError tracks authentication errors (no-op implementation).
func (n NoOpAnalyticsTracker) TrackAuthError(errorMessage string) {}

// StdoutTracker logs all analytics events to stdout for debugging and testing purposes.
type StdoutTracker struct{}

// TrackAPIRequest logs API requests to stdout.
func (s StdoutTracker) TrackAPIRequest(method, host, endpoint string, statusCode int, duration time.Duration) {
	fmt.Printf("[ANALYTICS] API Request: %s %s%s -> %d (%v)\n", method, host, endpoint, statusCode, duration)
}

// TrackAPIError logs API errors to stdout.
func (s StdoutTracker) TrackAPIError(method, host, endpoint string, statusCode int, errorMessage string) {
	fmt.Printf("[ANALYTICS] API Error: %s %s%s -> %d: %s\n", method, host, endpoint, statusCode, errorMessage)
}

// TrackAuthError logs authentication errors to stdout.
func (s StdoutTracker) TrackAuthError(errorMessage string) {
	fmt.Printf("[ANALYTICS] Auth Error: %s\n", errorMessage)
}

// DefaultTracker is the default implementation of Tracker that sends analytics events.
type DefaultTracker struct {
	tracker analytics.Tracker
	envRepo env.Repository
}

// NewDefaultTracker creates a new DefaultTracker instance.
func NewDefaultTracker(tracker analytics.Tracker, envRepo env.Repository) *DefaultTracker {
	return &DefaultTracker{
		tracker: tracker,
		envRepo: envRepo,
	}
}
// TrackAPIRequest tracks a completed API request.
func (d *DefaultTracker) TrackAPIRequest(method, host, endpoint string, statusCode int, duration time.Duration) {
	d.tracker.Enqueue("step_appstoreconnect_request", analytics.Properties{
		"build_slug":  d.envRepo.Get("BITRISE_BUILD_SLUG"),
		"http_method": method,
		"host":        host, // Regular, enterprise, or any future third option
		"endpoint":    endpoint,
		"status_code": statusCode,
		"duration_ms": duration.Truncate(time.Millisecond).Milliseconds(),
	})
}
// TrackAPIError tracks a failed API request with error details.
func (d *DefaultTracker) TrackAPIError(method, host, endpoint string, statusCode int, errorMessage string) {
	d.tracker.Enqueue("step_appstoreconnect_error", analytics.Properties{
		"build_slug":    d.envRepo.Get("BITRISE_BUILD_SLUG"),
		"http_method":   method,
		"host":          host, // Regular, enterprise, or any future third option
		"endpoint":      endpoint,
		"status_code":   statusCode,
		"error_message": errorMessage,
	})
}
// TrackAuthError tracks authentication-specific errors.
func (d *DefaultTracker) TrackAuthError(errorMessage string) {
	d.tracker.Enqueue("step_appstoreconnect_auth_error", analytics.Properties{
		"build_slug":    d.envRepo.Get("BITRISE_BUILD_SLUG"),
		"error_message": errorMessage,
	})
}
