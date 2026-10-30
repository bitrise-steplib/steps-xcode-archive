package xcodeproj

import (
	"fmt"
	"path"

	"github.com/bitrise-io/go-xcode/v2/xcodeproject/serialized"
)

const (
	resourcesBuildPhaseISA = "PBXResourcesBuildPhase"
	buildFileISA           = "PBXBuildFile"
	fileReferenceISA       = "PBXFileReference"
)

type resourcesBuildPhase struct {
	ID    string
	files []string
}

type buildFile struct {
	fileRef string
}

type fileReference struct {
	id   string
	path string
}

type sourceTree int

const (
	unsupportedParent sourceTree = iota
	groupParent
	absoluteParentPath
	undefinedParent
)

type projectEntry struct {
	id           string
	pathRelation sourceTree
	path         string
}

func isResourcesBuildPhase(raw serialized.Object) bool {
	isa, ok := raw.String("isa")
	return ok && isa == resourcesBuildPhaseISA
}

func parseResourcesBuildPhase(id string, objects serialized.Object) (resourcesBuildPhase, error) {
	raw, ok := objects.Object(id)
	if !ok {
		return resourcesBuildPhase{}, fmt.Errorf("object %s not found", id)
	}

	if !isResourcesBuildPhase(raw) {
		return resourcesBuildPhase{}, fmt.Errorf("not a %s element", resourcesBuildPhaseISA)
	}

	files, ok := raw.StringSlice("files")
	if !ok {
		return resourcesBuildPhase{}, fmt.Errorf("%s %s has no files", resourcesBuildPhaseISA, id)
	}

	return resourcesBuildPhase{ID: id, files: files}, nil
}

func parseBuildFile(id string, objects serialized.Object) (buildFile, error) {
	raw, ok := objects.Object(id)
	if !ok {
		return buildFile{}, fmt.Errorf("object %s not found", id)
	}
	if isa, ok := raw.String("isa"); !ok || isa != buildFileISA {
		return buildFile{}, fmt.Errorf("not a %s element", buildFileISA)
	}

	fileRef, ok := raw.String("fileRef")
	if !ok {
		return buildFile{}, fmt.Errorf("%s %s has no fileRef", buildFileISA, id)
	}

	return buildFile{fileRef: fileRef}, nil
}

func isFileReference(raw serialized.Object) (bool, error) {
	isa, ok := raw.String("isa")
	if !ok {
		return false, fmt.Errorf("object has no isa")
	}
	return isa == fileReferenceISA, nil
}

func parseFileReference(id string, objects serialized.Object) (fileReference, error) {
	raw, ok := objects.Object(id)
	if !ok {
		return fileReference{}, fmt.Errorf("object %s not found", id)
	}

	isReference, err := isFileReference(raw)
	if err != nil {
		return fileReference{}, err
	} else if !isReference {
		return fileReference{}, fmt.Errorf("not a %s element", fileReferenceISA)
	}

	pth, ok := raw.String("path")
	if !ok {
		return fileReference{}, fmt.Errorf("%s %s has no path", fileReferenceISA, id)
	}

	return fileReference{id: id, path: pth}, nil
}

// resolveObjectAbsolutePath resolves an object's path by walking the project's group tree from the
// main group down to the object.
func resolveObjectAbsolutePath(targetID, projectID, projectPath string, objects serialized.Object) (string, error) {
	if _, ok := objects.Object(targetID); !ok {
		return "", fmt.Errorf("object %s not found", targetID)
	}
	project, ok := objects.Object(projectID)
	if !ok {
		return "", fmt.Errorf("project %s not found", projectID)
	}

	projectDirPath, ok := project.String("projectDirPath")
	if !ok {
		return "", fmt.Errorf("key projectDirPath not found, project: %s", project)
	}
	projectRoot, ok := project.String("projectRoot")
	if !ok {
		return "", fmt.Errorf("key projectRoot not found, project: %s", project)
	}
	mainGroup, ok := project.String("mainGroup")
	if !ok {
		return "", fmt.Errorf("key mainGroup not found, project: %s", project)
	}

	pathInProjectTree, err := findInProjectTree(targetID, mainGroup, objects, map[string]bool{})
	if err != nil {
		return "", fmt.Errorf("failed to find target ID in project, error: %w", err)
	}
	pathInProjectTree = append(pathInProjectTree, projectEntry{
		path:         path.Join(projectPath, "..", projectDirPath, projectRoot),
		pathRelation: absoluteParentPath,
	})

	return resolveFilePath(pathInProjectTree)
}

// findInProjectTree returns the entries from target up to currentID, or nil if target is not below
// currentID.
func findInProjectTree(target, currentID string, objects serialized.Object, visited map[string]bool) ([]projectEntry, error) {
	if visited[currentID] {
		return nil, fmt.Errorf("circular reference in project, id: %s", currentID)
	}
	visited[currentID] = true

	entry, ok := objects.Object(currentID)
	if !ok {
		return nil, fmt.Errorf("object not found, id: %s", currentID)
	}

	entryPath, _ := entry.String("path")

	sourceTreeRaw, ok := entry.String("sourceTree")
	if !ok {
		return nil, fmt.Errorf("object %s has no sourceTree", currentID)
	}
	var pathRelation sourceTree
	switch sourceTreeRaw {
	case "<group>":
		pathRelation = groupParent
	case "<absolute>":
		pathRelation = absoluteParentPath
	case "":
		pathRelation = undefinedParent
	default:
		pathRelation = unsupportedParent
	}

	node := projectEntry{id: currentID, path: entryPath, pathRelation: pathRelation}

	if currentID == target {
		return []projectEntry{node}, nil
	}

	childIDs, ok := entry.StringSlice("children")
	if !ok {
		return nil, nil
	}
	for _, childID := range childIDs {
		pathInProjectTree, err := findInProjectTree(target, childID, objects, visited)
		if err != nil {
			return nil, err
		} else if pathInProjectTree != nil {
			return append(pathInProjectTree, node), nil
		}
	}
	return nil, nil
}

func resolveFilePath(nodes []projectEntry) (string, error) {
	var partialPath string
	for _, entry := range nodes {
		switch entry.pathRelation {
		case groupParent:
			partialPath = path.Join(entry.path, partialPath)
		case absoluteParentPath:
			return path.Join(entry.path, partialPath), nil
		case undefinedParent:
		case unsupportedParent:
			return "", fmt.Errorf("failed to resolve path, unsupported path relation")
		}
	}
	return partialPath, nil
}
