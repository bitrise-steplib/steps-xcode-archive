package autocodesign

import (
	"fmt"

	"github.com/bitrise-io/go-xcode/v2/autocodesign/devportalclient/appstoreconnect"
)

// DetailedError ...
type DetailedError struct {
	ErrorMessage   string
	Title          string
	Description    string
	Recommendation string
}

func (e DetailedError) Error() string {
	message := ""
	if e.ErrorMessage != "" {
		message += e.ErrorMessage + "\n"
	}
	message += "\n"
	if e.Title != "" {
		message += e.Title + "\n"
	}
	if e.Description != "" {
		message += e.Description + "\n"
	}
	if e.Recommendation != "" {
		message += "\n"
		message += e.Recommendation + "\n"
	}

	return message
}

// missingCertificateError ...
type missingCertificateError struct {
	Type appstoreconnect.CertificateType
}

func (e missingCertificateError) Error() string {
	return fmt.Sprintf("no valid %s type certificates uploaded\n ", e.Type)
}

// NonmatchingProfileError is returned when a profile/bundle ID does not match project requirements
// It is not a fatal error, as the profile can be regenerated
type NonmatchingProfileError struct {
	Reason string
}

func (e NonmatchingProfileError) Error() string {
	return fmt.Sprintf("provisioning profile does not match requirements: %s", e.Reason)
}

// ProfilesInconsistentError is returned when a profile is deleted by an other actor
type ProfilesInconsistentError struct {
	wrapErr error
}

// NewProfilesInconsistentError ...
func NewProfilesInconsistentError(wrapErr error) ProfilesInconsistentError {
	return ProfilesInconsistentError{
		wrapErr: wrapErr,
	}
}

func (e ProfilesInconsistentError) Error() string {
	return fmt.Sprintf("provisioning profiles were concurrently changed on Developer Portal, %s", e.wrapErr)
}

func (e ProfilesInconsistentError) Unwrap() error {
	return e.wrapErr
}

// ErrAppClipAppID ...
type ErrAppClipAppID struct {
}

// Error ...
func (ErrAppClipAppID) Error() string {
	return "can't create Application Identifier for App Clip target"
}

// ErrAppClipAppIDWithAppleSigning ...
type ErrAppClipAppIDWithAppleSigning struct {
}

// Error ...
func (ErrAppClipAppIDWithAppleSigning) Error() string {
	return "can't manage Application Identifier for App Clip target with 'Sign In With Apple' capability"
}
