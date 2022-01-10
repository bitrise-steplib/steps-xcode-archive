package pathutil

import (
	"errors"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// PathProvider ...
type PathProvider interface {
	CreateTempDir(prefix string) (string, error)
}

type pathProvider struct{}

// NewPathProvider ...
func NewPathProvider() PathProvider {
	return pathProvider{}
}

// CreateTempDir creates a temp dir, and returns its path.
// If prefix is provided it'll be used as the tmp dir's name prefix.
// Normalized: it's guaranteed that the path won't end with '/'.
func (pathProvider) CreateTempDir(prefix string) (dir string, err error) {
	dir, err = ioutil.TempDir("", prefix)
	dir = strings.TrimSuffix(dir, "/")

	return
}

// PathChecker ...
type PathChecker interface {
	IsPathExists(pth string) (bool, error)
}

type pathChecker struct{}

// NewPathChecker ...
func NewPathChecker() PathChecker {
	return pathChecker{}
}

// IsPathExists ...
func (c pathChecker) IsPathExists(pth string) (bool, error) {
	_, isExists, err := c.genericIsPathExists(pth)
	return isExists, err
}

func (pathChecker) genericIsPathExists(pth string) (os.FileInfo, bool, error) {
	if pth == "" {
		return nil, false, errors.New("No path provided")
	}

	fileInf, err := os.Lstat(pth)
	if err == nil {
		return fileInf, true, nil
	}

	if os.IsNotExist(err) {
		return nil, false, nil
	}

	return fileInf, false, err
}

// PathModifier ...
type PathModifier interface {
	AbsPath(pth string) (string, error)
}

type pathModifier struct{}

// NewPathModifier ...
func NewPathModifier() PathModifier {
	return pathModifier{}
}

// AbsPath expands ENV vars and the ~ character then calls Go's Abs
func (p pathModifier) AbsPath(pth string) (string, error) {
	if pth == "" {
		return "", errors.New("No Path provided")
	}

	pth, err := p.expandTilde(pth)
	if err != nil {
		return "", err
	}

	return filepath.Abs(os.ExpandEnv(pth))
}

func (pathModifier) expandTilde(pth string) (string, error) {
	if pth == "" {
		return "", errors.New("No Path provided")
	}

	if strings.HasPrefix(pth, "~") {
		pth = strings.TrimPrefix(pth, "~")

		if len(pth) == 0 || strings.HasPrefix(pth, "/") {
			return os.ExpandEnv("$HOME" + pth), nil
		}

		splitPth := strings.Split(pth, "/")
		username := splitPth[0]

		usr, err := user.Lookup(username)
		if err != nil {
			return "", err
		}

		pathInUsrHome := strings.Join(splitPth[1:], "/")

		return filepath.Join(usr.HomeDir, pathInUsrHome), nil
	}

	return pth, nil
}
