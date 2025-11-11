package xcarchive

import (
	"os"

	"github.com/bitrise-io/go-xcode/v2/plistutil"
)

func newPlistDataFromFile(plistPth string) (plistutil.PlistData, error) {
	content, err := os.ReadFile(plistPth)
	if err != nil {
		return plistutil.PlistData{}, err
	}
	return plistutil.NewPlistDataFromContent(string(content))
}
