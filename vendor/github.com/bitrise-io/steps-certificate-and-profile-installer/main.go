package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/steps-certificate-and-profile-installer/certificateutil"
	"github.com/bitrise-io/steps-certificate-and-profile-installer/profileutil"
	"github.com/bitrise-tools/go-steputils/input"
	version "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
)

const (
	notValidParameterErrorMessage = "security: SecPolicySetValue: One or more parameters passed to a function were not valid."
)

// -----------------------
// --- Models
// -----------------------

// ConfigsModel ...
type ConfigsModel struct {
	CertificateURL         string
	CertificatePassphrase  string
	ProvisioningProfileURL string

	DefaultCertificateURL         string
	DefaultCertificatePassphrase  string
	DefaultProvisioningProfileURL string

	KeychainPath     string
	KeychainPassword string
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		CertificateURL:         os.Getenv("certificate_url"),
		CertificatePassphrase:  os.Getenv("certificate_passphrase"),
		ProvisioningProfileURL: os.Getenv("provisioning_profile_url"),

		DefaultCertificateURL:         os.Getenv("default_certificate_url"),
		DefaultCertificatePassphrase:  os.Getenv("default_certificate_passphrase"),
		DefaultProvisioningProfileURL: os.Getenv("default_provisioning_profile_url"),

		KeychainPath:     os.Getenv("keychain_path"),
		KeychainPassword: os.Getenv("keychain_password"),
	}
}

func secureInput(str string) string {
	if str == "" {
		return ""
	}

	secureStr := func(s string, show int) string {
		runeCount := utf8.RuneCountInString(s)
		if runeCount < 6 || show == 0 {
			return strings.Repeat("*", 3)
		}
		if show*4 > runeCount {
			show = 1
		}

		sec := fmt.Sprintf("%s%s%s", s[0:show], strings.Repeat("*", 3), s[len(s)-show:len(s)])
		return sec
	}

	prefix := ""
	cont := str
	sec := secureStr(cont, 0)

	if strings.HasPrefix(str, "file://") {
		prefix = "file://"
		cont = strings.TrimPrefix(str, prefix)
		sec = secureStr(cont, 3)
	} else if strings.HasPrefix(str, "http://www.") {
		prefix = "http://www."
		cont = strings.TrimPrefix(str, prefix)
		sec = secureStr(cont, 3)
	} else if strings.HasPrefix(str, "https://www.") {
		prefix = "https://www."
		cont = strings.TrimPrefix(str, prefix)
		sec = secureStr(cont, 3)
	} else if strings.HasPrefix(str, "http://") {
		prefix = "http://"
		cont = strings.TrimPrefix(str, prefix)
		sec = secureStr(cont, 3)
	} else if strings.HasPrefix(str, "https://") {
		prefix = "https://"
		cont = strings.TrimPrefix(str, prefix)
		sec = secureStr(cont, 3)
	}

	return prefix + sec
}

func (configs ConfigsModel) print() {
	fmt.Println()
	log.Infof("Configs:")
	log.Printf(" - CertificateURL: %s", secureInput(configs.CertificateURL))
	log.Printf(" - CertificatePassphrase: %s", secureInput(configs.CertificatePassphrase))
	log.Printf(" - ProvisioningProfileURL: %s", secureInput(configs.ProvisioningProfileURL))

	log.Printf(" - DefaultCertificateURL: %s", secureInput(configs.DefaultCertificateURL))
	log.Printf(" - DefaultCertificatePassphrase: %s", secureInput(configs.DefaultCertificatePassphrase))
	log.Printf(" - DefaultProvisioningProfileURL: %s", secureInput(configs.DefaultProvisioningProfileURL))

	log.Printf(" - KeychainPath: %s", configs.KeychainPath)
	log.Printf(" - KeychainPassword: %s", secureInput(configs.KeychainPassword))
}

func (configs ConfigsModel) validate() error {
	if err := input.ValidateIfNotEmpty(configs.KeychainPath); err != nil {
		return fmt.Errorf("issue with inpout KeychainPath: %s", err)
	}

	if err := input.ValidateIfNotEmpty(configs.KeychainPassword); err != nil {
		return fmt.Errorf("issue with inpout KeychainPassword: %s", err)
	}

	return nil
}

//--------------------
// Functions
//--------------------

func downloadFile(destionationPath, URL string) error {
	url, err := url.Parse(URL)
	if err != nil {
		return err
	}

	scheme := url.Scheme

	tmpDstFilePath := ""
	if scheme != "file" {
		tmpDir, err := pathutil.NormalizedOSTempDirPath("download")
		if err != nil {
			return err
		}

		tmpDst := path.Join(tmpDir, "tmp_file")
		tmpDstFile, err := os.Create(tmpDst)
		if err != nil {
			return err
		}
		defer func() {
			if err := tmpDstFile.Close(); err != nil {
				log.Errorf("Failed to close file (%s), error: %s", tmpDst, err)
			}
		}()

		success := false
		var response *http.Response
		for i := 0; i < 3 && !success; i++ {
			if i > 0 {
				fmt.Println("-> Retrying...")
				time.Sleep(3 * time.Second)
			}

			response, err = http.Get(URL)
			if err != nil {
				log.Errorf(err.Error())
			} else {
				success = true
			}

			if response != nil {
				defer func() {
					if err := response.Body.Close(); err != nil {
						log.Errorf("Failed to close response body, error: %s", err)
					}
				}()
			}
		}
		if !success {
			return err
		}

		_, err = io.Copy(tmpDstFile, response.Body)
		if err != nil {
			return err
		}

		tmpDstFilePath = tmpDstFile.Name()
	} else {
		tmpDstFilePath = strings.Replace(URL, scheme+"://", "", -1)
	}

	return command.CopyFile(tmpDstFilePath, destionationPath)
}

func strip(str string) string {
	str = strings.TrimSpace(str)
	return strings.Trim(str, "\"")
}

func splitAndStrip(str, sep string) []string {
	items := []string{}
	split := strings.Split(str, sep)
	for _, item := range split {
		item = strings.TrimSpace(item)
		item = strings.Trim(item, "\"")
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

func splitAndTrimSpace(str, sep string) []string {
	items := []string{}
	split := strings.Split(str, sep)
	for _, item := range split {
		item = strings.TrimSpace(item)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

func appendWithoutDuplicatesAndKeepOrder(items []string, item string) []string {
	result := []string{}
	resultMap := map[string]bool{}

	list := append(items, item)
	for _, i := range list {
		exist, _ := resultMap[i]
		if !exist {
			result = append(result, i)
			resultMap[i] = true
		}
	}

	return result
}

func printCertificate(certInfo certificateutil.CertificateInfoModel) {
	if certInfo.CommonName != "" && certInfo.TeamID != "" {
		log.Donef(certInfo.CommonName)
		log.Printf("serial: %s", certInfo.Serial)
		log.Printf("teamID: %s", certInfo.TeamID)
	} else {
		log.Donef(certInfo.RawSubject)
		log.Printf("serial: %s", certInfo.Serial)
	}

	if certInfo.EndDate.IsZero() {
		log.Printf(certInfo.RawEndDate)
	} else {
		log.Printf("endDate: %s", certInfo.EndDate)

		if certInfo.IsExpired() {
			log.Errorf("[X] certificate expired")
		}
	}
}

func printProfile(profileInfo profileutil.ProfileInfoModel, installedCertificates []certificateutil.CertificateInfoModel) {
	log.Donef("%s (%s)", profileInfo.Name, profileInfo.UUID)
	log.Printf("exportType: %s", string(profileInfo.ExportType))
	log.Printf("teamID: %s", profileInfo.TeamIdentifier)
	log.Printf("bundleID: %s", profileInfo.BundleIdentifier)
	log.Printf("expirationDate: %s", profileInfo.ExpirationDate)
	log.Printf("certificates:")

	for _, certInfo := range profileInfo.DeveloperCertificates {
		if certInfo.CommonName != "" && certInfo.TeamID != "" {
			log.Printf("- %s", certInfo.CommonName)
			log.Printf("  serial: %s", certInfo.Serial)
			log.Printf("  teamID: %s", certInfo.TeamID)
		} else {
			log.Printf("- %s", certInfo.RawSubject)
			log.Printf("  serial: %s", certInfo.Serial)
		}
	}

	if !profileInfo.HasInstalledCertificate(installedCertificates) {
		log.Errorf("[X] none of the profile's certificates are installed")
	}

	if profileInfo.IsXcodeManaged() {
		log.Warnf("[!] xcode managed profile")
	}

	if profileInfo.IsExpired() {
		log.Errorf("[X] profile expired")
	}
}

func commandError(printableCmd string, cmdOut string, cmdErr error) error {
	return errors.Wrapf(cmdErr, "%s failed, out: %s", printableCmd, cmdOut)
}

func failF(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func failE(err error) {
	log.Errorf(err.Error())
	os.Exit(1)
}

//--------------------
// Main
//--------------------

func main() {
	configs := createConfigsModelFromEnvs()
	configs.print()
	if err := configs.validate(); err != nil {
		failF("Issue with input: %s", err)
	}
	fmt.Println()

	// Collect Certificates
	certificateURLPassphraseMap := map[string]string{}

	if configs.CertificateURL != "" {
		certificateURLs := splitAndTrimSpace(configs.CertificateURL, "|")

		// Do not splitAndTrimSpace passphrases, since passphrase is may empty !!!
		certificatePassphrases := strings.Split(configs.CertificatePassphrase, "|")

		if len(certificateURLs) != len(certificatePassphrases) {
			failF("Certificate url count: (%d), not equals to Certificate Passphrase count: (%d)", len(certificateURLs), len(certificatePassphrases))
		}

		for i := 0; i < len(certificateURLs); i++ {
			certificateURL := certificateURLs[i]
			certificatePassphrase := certificatePassphrases[i]

			certificateURLPassphraseMap[certificateURL] = certificatePassphrase
		}
	}

	if configs.DefaultCertificateURL != "" {
		log.Printf("Default Certificate given")
		certificateURLPassphraseMap[configs.DefaultCertificateURL] = configs.DefaultCertificatePassphrase
	}

	certificateCount := len(certificateURLPassphraseMap)
	log.Printf("Provided Certificate count: %d", certificateCount)

	if certificateCount == 0 {
		log.Warnf("No Certificate provided")
	}

	// Collect Provisioning Profiles
	provisioningProfileURLs := splitAndTrimSpace(configs.ProvisioningProfileURL, "|")

	if configs.DefaultProvisioningProfileURL != "" {
		log.Printf("Default Provisioning Profile given")
		provisioningProfileURLs = append(provisioningProfileURLs, configs.DefaultProvisioningProfileURL)
	}

	profileCount := len(provisioningProfileURLs)
	log.Printf("Provided Provisioning Profile count: %d", profileCount)

	if profileCount == 0 {
		log.Warnf("No Provisioning Profile provided")
	}

	// Init
	homeDir := os.Getenv("HOME")
	provisioningProfileDir := path.Join(homeDir, "Library/MobileDevice/Provisioning Profiles")
	if exist, err := pathutil.IsPathExists(provisioningProfileDir); err != nil {
		failF("Failed to check path (%s), err: %s", provisioningProfileDir, err)
	} else if !exist {
		if err := os.MkdirAll(provisioningProfileDir, 0777); err != nil {
			failF("Failed to create path (%s), err: %s", provisioningProfileDir, err)
		}
	}

	tempDir, err := pathutil.NormalizedOSTempDirPath("bitrise-cert-tmp")
	if err != nil {
		failF("Failed to create tmp directory, err: %s", err)
	}

	if exist, err := pathutil.IsPathExists(configs.KeychainPath); err != nil {
		failF("Failed to check path (%s), err: %s", configs.KeychainPath, err)
	} else if !exist {
		fmt.Println()
		log.Warnf("Keychain (%s) does not exist", configs.KeychainPath)

		keychainPth := fmt.Sprintf("%s-db", configs.KeychainPath)

		log.Printf(" Checking (%s)", keychainPth)

		if exist, err := pathutil.IsPathExists(keychainPth); err != nil {
			failF("Failed to check path (%s), err: %s", keychainPth, err)
		} else if !exist {
			log.Infof("Creating keychain: %s", configs.KeychainPath)

			cmd := command.New("security", "-v", "create-keychain", "-p", configs.KeychainPassword, configs.KeychainPath)
			if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
				failE(commandError(cmd.PrintableCommandArgs(), out, err))
			}
		}
	} else {
		log.Printf("Keychain already exists, using it: %s", configs.KeychainPath)
	}

	//
	// Download certificate
	fmt.Println()
	log.Infof("Downloading & installing Certificate(s)")
	fmt.Println()

	certificatePassphraseMap := map[string]string{}
	idx := 0
	for certURL, pass := range certificateURLPassphraseMap {
		log.Printf("Downloading certificate: %d/%d", idx+1, certificateCount)

		certPath := path.Join(tempDir, fmt.Sprintf("Certificate-%d.p12", idx))
		if err := downloadFile(certPath, certURL); err != nil {
			failF("Download failed, err: %s", err)
		}
		certificatePassphraseMap[certPath] = pass

		idx++
	}

	//
	// Install certificate
	log.Printf("Installing downloaded certificates")
	fmt.Println()

	installedCertificates := []certificateutil.CertificateInfoModel{}

	for cert, pass := range certificatePassphraseMap {
		certInfos, err := certificateutil.CertificateInfosFromP12(cert, pass)
		if err != nil {
			failF("Failed to parse certificate, error: %s", err)
		}
		installedCertificates = append(installedCertificates, certInfos...)

		for _, certInfo := range certInfos {
			printCertificate(certInfo)
		}
		fmt.Println()

		// Import items into a keychain.
		cmd := command.New("security", "import", cert, "-k", configs.KeychainPath, "-P", pass, "-A")
		if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
			failE(commandError(cmd.PrintableCommandArgs(), out, err))
		}
	}

	// This is new behavior in Sierra, [openradar](https://openradar.appspot.com/28524119)
	// You need to use "security set-key-partition-list -S apple-tool:,apple: -k keychainPass keychainName" after importing the item and before attempting to use it via codesign.
	cmd := command.New("sw_vers", "-productVersion")
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		failE(commandError(cmd.PrintableCommandArgs(), out, err))
	}

	osVersion, err := version.NewVersion(out)
	if err != nil {
		failF("Failed to parse os version (%s), error: %s", out, err)
	}

	sierraVersionStr := "10.12.0"
	sierraVersion, err := version.NewVersion(sierraVersionStr)
	if err != nil {
		failF("Failed to parse os version (%s), error: %s", sierraVersionStr, err)
	}

	if !osVersion.LessThan(sierraVersion) {
		cmd := command.New("security", "set-key-partition-list", "-S", "apple-tool:,apple:", "-k", configs.KeychainPassword, configs.KeychainPath)
		if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
			failE(commandError(cmd.PrintableCommandArgs(), out, err))
		}
	}
	// ---

	// Set keychain settings: Lock keychain when the system sleeps, Lock keychain after timeout interval, Timeout in seconds
	cmd = command.New("security", "-v", "set-keychain-settings", "-lut", "72000", configs.KeychainPath)
	if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
		failE(commandError(cmd.PrintableCommandArgs(), out, err))
	}

	// List keychains
	cmd = command.New("security", "list-keychains")
	listKeychainsOut, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		failE(commandError(cmd.PrintableCommandArgs(), listKeychainsOut, err))
	}

	keychainList := splitAndStrip(listKeychainsOut, "\n")
	keychainList = appendWithoutDuplicatesAndKeepOrder(keychainList, configs.KeychainPath)

	// Set keychain search path
	args := append([]string{"-v", "list-keychains", "-s"}, keychainList...)
	cmd = command.New("security", args...)
	if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
		failE(commandError(cmd.PrintableCommandArgs(), out, err))
	}

	// Set the default keychain
	cmd = command.New("security", "-v", "default-keychain", "-s", configs.KeychainPath)
	if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
		failE(commandError(cmd.PrintableCommandArgs(), out, err))
	}

	// Unlock the specified keychain
	cmd = command.New("security", "-v", "unlock-keychain", "-p", configs.KeychainPassword, configs.KeychainPath)
	if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
		failE(commandError(cmd.PrintableCommandArgs(), out, err))
	}

	//
	// Install provisioning profiles
	fmt.Println()
	log.Infof("Downloading & installing Provisioning Profile(s)")

	for idx, profileURL := range provisioningProfileURLs {
		fmt.Println()
		log.Printf("Downloading provisioning profile: %d/%d", idx+1, profileCount)

		provisioningProfileExt := "provisionprofile"
		if !strings.Contains(profileURL, "."+provisioningProfileExt) {
			provisioningProfileExt = "mobileprovision"
		}

		profileTmpPth := path.Join(tempDir, fmt.Sprintf("profile-%d.%s", idx, provisioningProfileExt))
		if err := downloadFile(profileTmpPth, profileURL); err != nil {
			failF("Download failed, err: %s", err)
		}

		profile, err := profileutil.ProfileFromFile(profileTmpPth)
		if err != nil {
			failF("Failed to parse profile, error: %s", err)
		}

		profilePth := path.Join(provisioningProfileDir, profile.UUID+"."+provisioningProfileExt)

		log.Printf("Moving it to: %s", profilePth)

		if err := command.CopyFile(profileTmpPth, profilePth); err != nil {
			failF("Failed to copy profile from: %s to: %s", profileTmpPth, profilePth)
		}

		fmt.Println()
		printProfile(profile, installedCertificates)
	}
}
