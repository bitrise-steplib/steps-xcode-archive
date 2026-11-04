package xcodeproj

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bitrise-io/go-xcode/v2/xcodeproject/xcodeproj/internal/xcproj"
)

const (
	assetCatalogExtension = ".xcassets"
	projectPathBase       = "<PROJECT>/"
)

var filePathUnescaper = strings.NewReplacer(`\\`, `\`, `\<`, `<`)

// AppIconSetPaths implements XcodeProj. It looks for asset catalogs in the targets' resources
// build phases and, on disk, in the targets' synchronized folders.
func (p *XCProj) AppIconSetPaths() (map[string][]string, error) {
	return p.appIconSetPaths(p.assetCatalogPaths)
}

func (p *XCProj) assetCatalogPaths(target Target) ([]string, error) {
	if !target.isNativeTarget() {
		return nil, nil
	}

	resources := xcproj.BuildPhaseReference{Target: target.Name, Kind: xcproj.BuildPhaseKindResources}
	projectDir := filepath.Dir(p.path)

	var catalogs, folders []string
	var err error
	walkFiles(p.files, projectDir, nil, projectDir, func(reference xcproj.Reference, pth string, pathErr error) {
		// Membership exceptions are not decoded, so a catalog an exception excludes is still found,
		// and one it adds to another target is missed.
		isFolder := reference.Kind == xcproj.ReferenceKindFolder && slices.Contains(reference.Targets, target.Name)
		isCatalog := reference.Kind == xcproj.ReferenceKindFile && filepath.Ext(reference.Path) == assetCatalogExtension &&
			slices.Contains(reference.BuildPhases, resources)

		switch {
		case (isFolder || isCatalog) && pathErr != nil:
			err = cmp.Or(err, pathErr)
		case isFolder:
			folders = append(folders, pth)
		case isCatalog:
			catalogs = append(catalogs, pth)
		}
	})
	if err != nil {
		return nil, err
	}

	for _, folder := range folders {
		folderCatalogs, err := p.findAssetCatalogs(folder)
		if err != nil {
			return nil, err
		}
		catalogs = append(catalogs, folderCatalogs...)
	}

	return catalogs, nil
}

// walkFiles hands the path error to visit instead of stopping, because a path needs to resolve
// only when it leads to one of the target's catalogs.
func walkFiles(references []xcproj.Reference, groupDir string, groupErr error, projectDir string, visit func(xcproj.Reference, string, error)) {
	for _, reference := range references {
		pth, err := resolveXCProjFilePath(reference.Path, groupDir, groupErr, projectDir)
		visit(reference, pth, err)
		if reference.Kind == xcproj.ReferenceKindGroup {
			walkFiles(reference.Children, pth, err, projectDir, visit)
		}
	}
}

// resolveXCProjFilePath fails for ~ and for a base that needs build settings, such as <PRODUCTS>/,
// <SDK>/ or <USER:SETTING>/.
func resolveXCProjFilePath(pth, groupDir string, groupErr error, projectDir string) (string, error) {
	if relativePath, ok := strings.CutPrefix(pth, projectPathBase); ok {
		return filepath.Join(projectDir, relativePath), nil
	}
	if strings.HasPrefix(pth, "<") || strings.HasPrefix(pth, "~") {
		return "", fmt.Errorf("failed to resolve path %s: unsupported path base", pth)
	}

	pth = filePathUnescaper.Replace(pth)
	if filepath.IsAbs(pth) {
		return filepath.Clean(pth), nil
	}
	if groupErr != nil {
		return "", groupErr
	}
	return filepath.Join(groupDir, pth), nil
}

// findAssetCatalogs uses Lstat so that a symbolic link cycle cannot loop. A missing dir has no
// catalogs: git does not keep an empty synchronized folder.
func (p *XCProj) findAssetCatalogs(dir string) ([]string, error) {
	names, err := p.fileManager.ReadDirEntryNames(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list %s: %w", dir, err)
	}

	var catalogs []string
	for _, name := range names {
		pth := filepath.Join(dir, name)
		info, err := p.fileManager.Lstat(pth)
		if err != nil {
			return nil, fmt.Errorf("failed to check %s: %w", pth, err)
		}
		if !info.IsDir() {
			continue
		}

		if filepath.Ext(name) == assetCatalogExtension {
			catalogs = append(catalogs, pth)
			continue
		}

		nestedCatalogs, err := p.findAssetCatalogs(pth)
		if err != nil {
			return nil, err
		}
		catalogs = append(catalogs, nestedCatalogs...)
	}

	return catalogs, nil
}
