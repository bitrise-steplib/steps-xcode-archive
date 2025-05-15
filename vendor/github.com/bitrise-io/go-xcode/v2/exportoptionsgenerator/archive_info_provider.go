package exportoptionsgenerator

import (
	"github.com/bitrise-io/go-xcode/xcarchive"
)

// ExportProduct ...
type ExportProduct string

const (
	// ExportProductApp ...
	ExportProductApp ExportProduct = "app"
	// ExportProductAppClip ...
	ExportProductAppClip ExportProduct = "app-clip"
)

// ReadArchiveExportInfo ...
func ReadArchiveExportInfo(archive xcarchive.IosArchive) (ArchiveInfo, error) {
	appClipBundleID := ""
	if archive.Application.ClipApplication != nil {
		appClipBundleID = archive.Application.ClipApplication.BundleIdentifier()
	}

	return ArchiveInfo{
		AppBundleID:            archive.Application.BundleIdentifier(),
		AppClipBundleID:        appClipBundleID,
		EntitlementsByBundleID: archive.BundleIDEntitlementsMap(),
	}, nil
}
