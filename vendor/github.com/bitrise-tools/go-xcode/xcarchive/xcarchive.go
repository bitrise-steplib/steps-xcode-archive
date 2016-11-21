package xcarchive

import (
	"fmt"
	"path/filepath"
	"strings"
)

// EmbeddedMobileProvisionPth ...
func EmbeddedMobileProvisionPth(archivePth string) (string, error) {
	applicationPth := filepath.Join(archivePth, "/Products/Applications")
	mobileProvisionPthPattern := filepath.Join(applicationPth, "*.app/embedded.mobileprovision")
	mobileProvisionPths, err := filepath.Glob(mobileProvisionPthPattern)
	if err != nil {
		return "", fmt.Errorf("failed to find embedded.mobileprovision with pattern: %s, error: %s", mobileProvisionPthPattern, err)
	}
	if len(mobileProvisionPths) == 0 {
		return "", fmt.Errorf("no embedded.mobileprovision with pattern: %s", mobileProvisionPthPattern)
	}
	return mobileProvisionPths[0], nil
}

// FindDSYMs ...
func FindDSYMs(archivePth string) (string, []string, error) {
	pattern := filepath.Join(archivePth, "dSYMs", "*.dSYM")
	dsyms, err := filepath.Glob(pattern)
	if err != nil {
		return "", []string{}, fmt.Errorf("failed to find dSYM with pattern: %s, error: %s", pattern, err)
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
	return appDSYM, frameworkDSYMs, nil
}

// FindApp ...
func FindApp(archivePth string) (string, error) {
	pattern := filepath.Join(archivePth, "Products/Applications", "*.app")
	apps, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to find .app directory with pattern: %s, error: %s", pattern, err)
	}

	if len(apps) == 0 {
		return "", fmt.Errorf("no app found with pattern (%s)", pattern)
	}

	return apps[0], nil
}
