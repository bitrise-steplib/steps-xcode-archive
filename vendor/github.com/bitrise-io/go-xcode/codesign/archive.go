package codesign

import (
	"github.com/bitrise-io/go-xcode/autocodesign"
	"github.com/bitrise-io/go-xcode/xcarchive"
)

// Archive ...
type Archive struct {
	archive xcarchive.IosArchive
}

// NewArchive ...
func NewArchive(archive xcarchive.IosArchive) Archive {
	return Archive{
		archive: archive,
	}
}

// IsSigningManagedAutomatically ...
func (a Archive) IsSigningManagedAutomatically() (bool, error) {
	return a.archive.IsXcodeManaged(), nil
}

// Platform ...
func (a Archive) Platform() (autocodesign.Platform, error) {
	return a.archive.Platform()
}

// GetAppLayout ...
func (a Archive) GetAppLayout(uiTestTargets bool) (autocodesign.AppLayout, error) {
	params, err := a.archive.ReadCodesignParameters()
	if err != nil {
		return autocodesign.AppLayout{}, err
	}
	return *params, nil
}
