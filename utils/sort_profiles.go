package utils

import "github.com/bitrise-io/steps-certificate-and-profile-installer/profileutil"

// ByBundleIDLength ...
type ByBundleIDLength []profileutil.ProfileModel

// Len ..
func (s ByBundleIDLength) Len() int {
	return len(s)
}

// Swap ...
func (s ByBundleIDLength) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less ...
func (s ByBundleIDLength) Less(i, j int) bool {
	return len(s[i].BundleIdentifier) > len(s[j].BundleIdentifier)
}
