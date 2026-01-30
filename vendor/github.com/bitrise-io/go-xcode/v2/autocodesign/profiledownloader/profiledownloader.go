package profiledownloader

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/filedownloader"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-io/go-xcode/profileutil"
	"github.com/bitrise-io/go-xcode/v2/autocodesign"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/localcodesignasset"
)

type downloader struct {
	urls         []string
	logger       log.Logger
	fileProvider stepconf.FileProvider
}

// New returns an implementation that can download from remote, local file paths
func New(profileURLs []string, logger log.Logger) autocodesign.ProfileProvider {
	fileDownloader := filedownloader.NewDownloader(logger)
	fileProvider := stepconf.NewFileProvider(fileDownloader, fileutil.NewFileManager(), pathutil.NewPathProvider(), pathutil.NewPathModifier())

	return downloader{
		urls:         profileURLs,
		logger:       logger,
		fileProvider: fileProvider,
	}
}

// IsAvailable returns true if there are available remote profiles to download
func (d downloader) IsAvailable() bool {
	return len(d.urls) != 0
}

// GetProfiles downloads remote profiles and returns their contents
func (d downloader) GetProfiles() ([]autocodesign.LocalProfile, error) {
	var profiles []autocodesign.LocalProfile

	for _, url := range d.urls {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		contentReader, err := d.fileProvider.Contents(ctx, url)
		if err != nil {
			return nil, err
		}
		defer func() {
			if err := contentReader.Close(); err != nil {
				d.logger.Warnf("Failed to close profile reader: %s", err)
			}
		}()

		content, err := io.ReadAll(contentReader)
		if err != nil {
			return nil, err
		}
		if len(content) == 0 {
			return nil, fmt.Errorf("profile (%s) is empty", url)
		}

		parsedProfile, err := profileutil.ProvisioningProfileFromContent(content)
		if err != nil {
			return nil, fmt.Errorf("invalid pkcs7 file format: %w", err)
		}

		profileInfo, err := profileutil.NewProvisioningProfileInfo(*parsedProfile)
		if err != nil {
			return nil, fmt.Errorf("unknown provisioning profile format: %w", err)
		}

		profiles = append(profiles, autocodesign.LocalProfile{
			Profile: localcodesignasset.NewProfile(profileInfo, content),
			Info:    profileInfo,
		})
	}

	return profiles, nil
}
