// Package keychain contains methods to manage and install certificates to the MacOS keychain.
package keychain

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-xcode/certificateutil"
	"github.com/hashicorp/go-version"
)

// Keychain describes a macOS Keychain
type Keychain struct {
	path     string
	password stepconf.Secret

	factory command.Factory
}

// New ...
func New(pth string, pass stepconf.Secret, factory command.Factory) (*Keychain, error) {
	if exist, err := pathutil.IsPathExists(pth); err != nil {
		return nil, err
	} else if exist {
		return &Keychain{
			path:     pth,
			password: stepconf.Secret(pass),
			factory:  factory,
		}, nil
	}

	p := pth + "-db"
	if exist, err := pathutil.IsPathExists(p); err != nil {
		return nil, err
	} else if exist {
		return &Keychain{
			path:     p,
			password: pass,
			factory:  factory,
		}, nil
	}

	return createKeychain(pth, pass, factory)
}

// InstallCertificate ...
func (k Keychain) InstallCertificate(cert certificateutil.CertificateInfoModel, pass stepconf.Secret) error {
	b, err := cert.EncodeToP12("bitrise")
	if err != nil {
		return err
	}

	tmpDir, err := pathutil.NormalizedOSTempDirPath("keychain")
	if err != nil {
		return err
	}
	pth := filepath.Join(tmpDir, "Certificate.p12")
	if err := fileutil.WriteBytesToFile(pth, b); err != nil {
		return err
	}

	if err := k.importCertificate(pth, "bitrise"); err != nil {
		return err
	}

	if needed, err := k.isKeyPartitionListNeeded(); err != nil {
		return err
	} else if needed {
		if err := k.setKeyPartitionList(); err != nil {
			return err
		}
	}

	if err := k.setLockSettings(); err != nil {
		return err
	}

	if err := k.addToSearchPath(); err != nil {
		return err
	}

	if err := k.setAsDefault(); err != nil {
		return err
	}

	return k.unlock()
}

func runSecurityCmd(factory command.Factory, args ...interface{}) error {
	var printableArgs []string
	var cmdArgs []string
	for _, arg := range args {
		v, ok := arg.(stepconf.Secret)
		if ok {
			printableArgs = append(printableArgs, v.String())
			cmdArgs = append(cmdArgs, string(v))
		} else if v, ok := arg.(string); ok {
			printableArgs = append(printableArgs, v)
			cmdArgs = append(cmdArgs, v)
		} else if v, ok := arg.([]string); ok {
			printableArgs = append(printableArgs, v...)
			cmdArgs = append(cmdArgs, v...)
		} else {
			return fmt.Errorf("unknown arg provided: %T, string, []string, and stepconf.Secret are acceptable", arg)
		}
	}

	out, err := factory.Create("security", cmdArgs, nil).RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		if errorutil.IsExitStatusError(err) {
			return fmt.Errorf("%s failed: %s", strings.Join(append([]string{"security"}, printableArgs...), " "), out)
		}
		return fmt.Errorf("%s failed: %s", strings.Join(append([]string{"security"}, printableArgs...), " "), err)
	}
	return nil
}

// listKeychains returns the paths of available keychains
func (k Keychain) listKeychains() ([]string, error) {
	cmd := k.factory.Create("security", []string{"list-keychain"}, nil)
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		if errorutil.IsExitStatusError(err) {
			return nil, fmt.Errorf("%s failed: %s", cmd.PrintableCommandArgs(), out)
		}
		return nil, fmt.Errorf("%s failed: %s", cmd.PrintableCommandArgs(), err)
	}

	var keychains []string
	for _, path := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(path)
		trimmed = strings.Trim(trimmed, `"`)
		keychains = append(keychains, trimmed)
	}

	return keychains, nil
}

// createKeychain creates a new keychain file at
// path, protected by password. Returns an error
// if the keychain could not be created, otherwise
// a Keychain object representing the created
// keychain is returned.
func createKeychain(path string, password stepconf.Secret, factory command.Factory) (*Keychain, error) {
	err := runSecurityCmd(factory, "-v", "create-keychain", "-p", password, path)
	if err != nil {
		return nil, err
	}

	return &Keychain{
		path:     path,
		password: password,
		factory:  factory,
	}, nil
}

// importCertificate adds the certificate at path, protected by
// passphrase to the k keychain.
func (k Keychain) importCertificate(path string, passphrase stepconf.Secret) error {
	return runSecurityCmd(k.factory, "import", path, "-k", k.path, "-P", passphrase, "-A")
}

// setKeyPartitionList sets the partition list
// for the keychain to allow access for tools.
func (k Keychain) setKeyPartitionList() error {
	return runSecurityCmd(k.factory, "set-key-partition-list", "-S", "apple-tool:,apple:", "-k", k.password, k.path)
}

// setLockSettings sets keychain autolocking.
func (k Keychain) setLockSettings() error {
	return runSecurityCmd(k.factory, "-v", "set-keychain-settings", "-lut", "72000", k.path)
}

// addToSearchPath registers the keychain
// in the systemwide search path
func (k Keychain) addToSearchPath() error {
	keychains, err := k.listKeychains()
	if err != nil {
		return fmt.Errorf("get keychain list: %s", err)
	}

	return runSecurityCmd(k.factory, "-v", "list-keychains", "-s", keychains)
}

// setAsDefault sets the keychain as the
// default keychain for the system.
func (k Keychain) setAsDefault() error {
	return runSecurityCmd(k.factory, "-v", "default-keychain", "-s", k.path)
}

// unlock unlocks the keychain
func (k Keychain) unlock() error {
	return runSecurityCmd(k.factory, "-v", "unlock-keychain", "-p", k.password, k.path)
}

// isKeyPartitionListNeeded determines whether
// key partition lists are used by the system.
func (k Keychain) isKeyPartitionListNeeded() (bool, error) {
	cmd := k.factory.Create("sw_vers", []string{"-productVersion"}, nil)
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		if errorutil.IsExitStatusError(err) {
			return false, fmt.Errorf("%s failed: %s", cmd.PrintableCommandArgs(), out)
		}
		return false, fmt.Errorf("%s failed: %s", cmd.PrintableCommandArgs(), err)
	}

	const versionSierra = "10.12.0"
	sierra, err := version.NewVersion(versionSierra)
	if err != nil {
		return false, fmt.Errorf("invalid version (%s): %s", versionSierra, err)
	}

	current, err := version.NewVersion(out)
	if err != nil {
		return false, fmt.Errorf("invalid version (%s): %s", current, err)
	}
	if current.GreaterThanOrEqual(sierra) {
		return true, nil
	}

	return false, nil
}
