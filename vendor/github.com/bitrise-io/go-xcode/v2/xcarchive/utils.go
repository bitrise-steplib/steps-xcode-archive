package xcarchive

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-xcode/plistutil"
)

func executableNameFromInfoPlist(infoPlist plistutil.PlistData) string {
	if name, ok := infoPlist.GetString("CFBundleExecutable"); ok {
		return name
	}
	return ""
}

func getEntitlements(cmdFactory command.Factory, basePath, executableRelativePath string) (plistutil.PlistData, error) {
	entitlements, err := entitlementsFromExecutable(cmdFactory, basePath, executableRelativePath)
	if err != nil {
		return plistutil.PlistData{}, err
	}

	if entitlements != nil {
		return *entitlements, nil
	}

	return plistutil.PlistData{}, nil
}

func entitlementsFromExecutable(cmdFactory command.Factory, basePath, executableRelativePath string) (*plistutil.PlistData, error) {
	fmt.Printf("Fetching entitlements from executable")

	cmd := cmdFactory.Create("codesign", []string{"--display", "--entitlements", ":-", filepath.Join(basePath, executableRelativePath)}, nil)
	entitlementsString, err := cmd.RunAndReturnTrimmedOutput()
	if err != nil {
		return nil, err
	}

	plist, err := plistutil.NewPlistDataFromContent(entitlementsString)
	if err != nil {
		return nil, err
	}

	return &plist, nil
}

func findDSYMs(archivePath string) ([]string, []string, error) {
	dsymsDirPth := filepath.Join(archivePath, "dSYMs")
	dsyms, err := listEntries(dsymsDirPth, extensionFilter(".dsym", true))
	if err != nil {
		return []string{}, []string{}, err
	}

	appDSYMs := []string{}
	frameworkDSYMs := []string{}
	for _, dsym := range dsyms {
		if strings.HasSuffix(dsym, ".app.dSYM") {
			appDSYMs = append(appDSYMs, dsym)
		} else {
			frameworkDSYMs = append(frameworkDSYMs, dsym)
		}
	}

	return appDSYMs, frameworkDSYMs, nil
}

func escapeGlobPath(path string) string {
	var escaped string
	for _, ch := range path {
		if ch == '[' || ch == ']' || ch == '-' || ch == '*' || ch == '?' || ch == '\\' {
			escaped += "\\"
		}
		escaped += string(ch)
	}
	return escaped
}

type filterFunc func(string) (bool, error)

func listEntries(dir string, filters ...filterFunc) ([]string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return []string{}, err
	}

	entries, err := os.ReadDir(absDir)
	if err != nil {
		return []string{}, err
	}

	var paths []string
	for _, entry := range entries {
		pth := filepath.Join(absDir, entry.Name())
		paths = append(paths, pth)
	}

	return filterPaths(paths, filters...)
}

func filterPaths(fileList []string, filters ...filterFunc) ([]string, error) {
	var filtered []string

	for _, pth := range fileList {
		allowed := true
		for _, filter := range filters {
			if allows, err := filter(pth); err != nil {
				return []string{}, err
			} else if !allows {
				allowed = false
				break
			}
		}
		if allowed {
			filtered = append(filtered, pth)
		}
	}

	return filtered, nil
}

func extensionFilter(ext string, allowed bool) filterFunc {
	return func(pth string) (bool, error) {
		e := filepath.Ext(pth)
		return allowed == strings.EqualFold(ext, e), nil
	}
}
