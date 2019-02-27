package xcscheme

import (
	"path/filepath"
	"regexp"
)

// FindSchemesIn ...
func FindSchemesIn(root string) (schemes []Scheme, err error) {
	//
	// Add the shared schemes to the list
	sharedPths, err := pathsByPattern(regexp.QuoteMeta(root), "xcshareddata", "xcschemes", "*.xcscheme")
	if err != nil {
		return nil, err
	}

	//
	// Add the non-shared user schemes to the list
	userPths, err := pathsByPattern(regexp.QuoteMeta(root), "xcuserdata", "*.xcuserdatad", "xcschemes", "*.xcscheme")
	if err != nil {
		return nil, err
	}

	for _, pth := range append(sharedPths, userPths...) {
		scheme, err := Open(pth)
		if err != nil {
			return nil, err
		}
		schemes = append(schemes, scheme)
	}
	return
}

func pathsByPattern(paths ...string) ([]string, error) {
	pattern := filepath.Join(paths...)
	return filepath.Glob(pattern)
}
