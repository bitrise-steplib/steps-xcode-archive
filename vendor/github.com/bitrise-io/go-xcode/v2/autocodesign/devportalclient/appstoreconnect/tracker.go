package appstoreconnect

import (
	"time"

	"github.com/bitrise-io/go-utils/v2/analytics"
	"github.com/bitrise-io/go-utils/v2/env"
)

// Tracker defines the interface for tracking App Store Connect API usage and errors.
type Tracker interface {
	// TrackAPIRequest tracks one HTTP request+response. This is called for each individual attempt in case of automatic retries.
	TrackAPIRequest(method, host, endpoint string, statusCode int, duration time.Duration, isRetry bool)

	// TrackAPIError tracks a failed API request with error details
	TrackAPIError(method, host, endpoint string, statusCode int, errorMessage string)

	// TrackAuthError tracks authentication-specific errors
	TrackAuthError(errorMessage string)
}

// NoOpAnalyticsTracker is a dummy implementation used in tests.
type NoOpAnalyticsTracker struct{}

// TrackAPIRequest ...
func (n NoOpAnalyticsTracker) TrackAPIRequest(method, host, endpoint string, statusCode int, duration time.Duration, isRetry bool) {
}

// TrackAPIError ...
func (n NoOpAnalyticsTracker) TrackAPIError(method, host, endpoint string, statusCode int, errorMessage string) {
}

// TrackAuthError ...
func (n NoOpAnalyticsTracker) TrackAuthError(errorMessage string) {}

// DefaultTracker is the main implementation of Tracker
type DefaultTracker struct {
	tracker analytics.Tracker
	envRepo env.Repository
}

// NewDefaultTracker ...
func NewDefaultTracker(tracker analytics.Tracker, envRepo env.Repository) *DefaultTracker {
	return &DefaultTracker{
		tracker: tracker,
		envRepo: envRepo,
	}
}

// TrackAPIRequest ...
func (d *DefaultTracker) TrackAPIRequest(method, host, endpoint string, statusCode int, duration time.Duration, isRetry bool) {
	d.tracker.Enqueue("step_appstoreconnect_request", analytics.Properties{
		"build_slug":        d.envRepo.Get("BITRISE_BUILD_SLUG"),
		"step_execution_id": d.envRepo.Get("BITRISE_STEP_EXECUTION_ID"),
		"http_method":       method,
		"host":              host, // Regular, enterprise, or any future third option
		"endpoint":          endpoint,
		"status_code":       statusCode,
		"duration_ms":       duration.Truncate(time.Millisecond).Milliseconds(),
		"is_retry":          isRetry,
	})
}

// TrackAPIError ...
func (d *DefaultTracker) TrackAPIError(method, host, endpoint string, statusCode int, errorMessage string) {
	d.tracker.Enqueue("step_appstoreconnect_error", analytics.Properties{
		"build_slug":        d.envRepo.Get("BITRISE_BUILD_SLUG"),
		"step_execution_id": d.envRepo.Get("BITRISE_STEP_EXECUTION_ID"),
		"http_method":       method,
		"host":              host, // Regular, enterprise, or any future third option
		"endpoint":          endpoint,
		"status_code":       statusCode,
		"error_message":     errorMessage,
	})
}

// TrackAuthError ...
func (d *DefaultTracker) TrackAuthError(errorMessage string) {
	d.tracker.Enqueue("step_appstoreconnect_auth_error", analytics.Properties{
		"build_slug":        d.envRepo.Get("BITRISE_BUILD_SLUG"),
		"step_execution_id": d.envRepo.Get("BITRISE_STEP_EXECUTION_ID"),
		"error_message":     errorMessage,
	})
}
