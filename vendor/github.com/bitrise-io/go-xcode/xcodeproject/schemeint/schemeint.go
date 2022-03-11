package schemeint

import (
	"github.com/bitrise-io/go-xcode/xcodeproject/xcodeproj"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcscheme"
	"github.com/bitrise-io/go-xcode/xcodeproject/xcworkspace"
	"github.com/bitrise-io/go-utils/v2/log"
)

var logger = log.NewLogger()

// HasScheme represents a struct that implements Scheme.
type HasScheme interface {
	Scheme(string) (*xcscheme.Scheme, string, error)
}

// Scheme returns the project or workspace scheme by name.
func Scheme(pth string, name string) (*xcscheme.Scheme, string, error) {
	var p HasScheme
	var err error

	logger.Infof("[mattrob] Scheme - 1")

	if xcodeproj.IsXcodeProj(pth) {
		logger.Infof("[mattrob] Scheme - xcodeproj.Open(pth)")
		p, err = xcodeproj.Open(pth)
	} else {
		logger.Infof("[mattrob] Scheme - xcworkspace.Open(pth)")
		p, err = xcworkspace.Open(pth)
	}

	logger.Infof("[mattrob] Scheme - 2")

	if err != nil {
		return nil, "", err
	}
	return p.Scheme(name)
}
