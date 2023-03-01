package profiledownloader

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bitrise-io/go-steputils/input"
	"github.com/bitrise-io/go-utils/filedownloader"
	"github.com/bitrise-io/go-xcode/profileutil"
	"github.com/bitrise-io/go-xcode/v2/autocodesign"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/localcodesignasset"
)

type downloader struct {
	urls   []string
	client *http.Client
}

// New returns an implementation that can download from remote, local file paths
func New(profileURLs []string, client *http.Client) autocodesign.ProfileProvider {
	return downloader{
		urls:   profileURLs,
		client: client,
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

		downloader := filedownloader.NewWithContext(ctx, d.client)
		fileProvider := input.NewFileProvider(downloader)

		content, err := fileProvider.Contents(url)
		if err != nil {
			return nil, err
		} else if content == nil {
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
