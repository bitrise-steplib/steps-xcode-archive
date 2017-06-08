package xcarchive

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-tools/go-xcode/utility"
)

// FindEmbeddedMobileProvision ...
func FindEmbeddedMobileProvision(archivePth string) (string, error) {
	if exist, err := pathutil.IsDirExists(archivePth); err != nil {
		return "", fmt.Errorf("failed to check if archive exist, error: %s", err)
	} else if !exist {
		return "", fmt.Errorf("archive not exist at: %s", archivePth)
	}

	applicationsDirPth := filepath.Join(archivePth, "Products/Applications")
	apps, err := utility.ListEntries(applicationsDirPth, utility.ExtensionFilter(".app", true))
	if err != nil {
		return "", err
	}

	for _, app := range apps {
		embeddedProfiles, err := utility.ListEntries(app, utility.BaseFilter("embedded.mobileprovision", true))
		if err != nil {
			return "", err
		}
		if len(embeddedProfiles) > 0 {
			return embeddedProfiles[0], nil
		}
	}

	return "", fmt.Errorf("no embedded.mobileprovision found")
}

// FindDSYMs ...
func FindDSYMs(archivePth string) (string, []string, error) {
	if exist, err := pathutil.IsDirExists(archivePth); err != nil {
		return "", []string{}, fmt.Errorf("failed to check if archive exist, error: %s", err)
	} else if !exist {
		return "", []string{}, fmt.Errorf("archive not exist at: %s", archivePth)
	}

	dsymsDirPth := filepath.Join(archivePth, "dSYMs")
	dsyms, err := utility.ListEntries(dsymsDirPth, utility.ExtensionFilter(".dsym", true))
	if err != nil {
		return "", []string{}, err
	}

	appDSYM := ""
	frameworkDSYMs := []string{}
	for _, dsym := range dsyms {
		if strings.HasSuffix(dsym, ".app.dSYM") {
			appDSYM = dsym
		} else {
			frameworkDSYMs = append(frameworkDSYMs, dsym)
		}
	}
	if appDSYM == "" && len(frameworkDSYMs) == 0 {
		return "", []string{}, fmt.Errorf("no dsym found")
	}

	return appDSYM, frameworkDSYMs, nil
}

// FindApp ...
func FindApp(archivePth string) (string, error) {
	if exist, err := pathutil.IsDirExists(archivePth); err != nil {
		return "", fmt.Errorf("failed to check if archive exist, error: %s", err)
	} else if !exist {
		return "", fmt.Errorf("archive not exist at: %s", archivePth)
	}

	applicationsDirPth := filepath.Join(archivePth, "Products/Applications")
	apps, err := utility.ListEntries(applicationsDirPth, utility.ExtensionFilter(".app", true))
	if err != nil {
		return "", err
	}

	if len(apps) == 0 {
		return "", fmt.Errorf("no app found")
	}

	return apps[0], nil
}
