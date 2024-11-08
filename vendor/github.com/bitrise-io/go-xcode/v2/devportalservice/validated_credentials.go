package devportalservice

// Credentials contains only one of Apple ID (the session cookies already checked) or APIKey auth info
type Credentials struct {
	AppleID *AppleID
	APIKey  *APIKeyConnection
}

// AppleID contains Apple ID auth info
//
// Without 2FA:
//
//	Required: username, password
//
// With 2FA:
//
//	  Required: username, password, appSpecificPassword
//				   session (Only for Fastlane, set as FASTLANE_SESSION)
//
// As Fastlane spaceship uses:
//   - iTMSTransporter: it requires Username + Password (or App-specific password with 2FA)
//   - TunesAPI: it requires Username + Password (+ 2FA session with 2FA)
type AppleID struct {
	Username, Password           string
	Session, AppSpecificPassword string
}
