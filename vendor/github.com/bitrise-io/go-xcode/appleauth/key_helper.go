package appleauth

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"

	"github.com/bitrise-io/go-steputils/input"
	"github.com/bitrise-io/go-utils/filedownloader"
)

func fetchPrivateKey(privateKeyURL string) ([]byte, string, error) {
	fileURL, err := url.Parse(privateKeyURL)
	if err != nil {
		return nil, "", err
	}

	// Download or load local file
	filedownloader := filedownloader.New(http.DefaultClient)
	fileProvider := input.NewFileProvider(filedownloader)
	localFile, err := fileProvider.LocalPath(fileURL.String())
	if err != nil {
		return nil, "", err
	}
	key, err := ioutil.ReadFile(localFile)
	if err != nil {
		return nil, "", err
	}

	return key, getKeyID(fileURL), nil
}

func getKeyID(u *url.URL) string {
	var keyID = "Bitrise" // as default if no ID found in file name

	// get the ID of the key from the file
	if matches := regexp.MustCompile(`AuthKey_(.+)\.p8`).FindStringSubmatch(filepath.Base(u.Path)); len(matches) == 2 {
		keyID = matches[1]
	}

	return keyID
}
