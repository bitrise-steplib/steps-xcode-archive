package exportoptiongenerator

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/steps-certificate-and-profile-installer/certificateutil"
	"github.com/bitrise-io/steps-certificate-and-profile-installer/profileutil"
	"github.com/bitrise-io/steps-xcode-archive/utils"
	"github.com/bitrise-tools/go-xcode/exportoptions"
	"github.com/bitrise-tools/go-xcode/xcodeproj"
	glob "github.com/ryanuber/go-glob"
)

// ByBundleIDLength ...
type ByBundleIDLength []profileutil.ProfileModel

// Len ..
func (s ByBundleIDLength) Len() int {
	return len(s)
}

// Swap ...
func (s ByBundleIDLength) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less ...
func (s ByBundleIDLength) Less(i, j int) bool {
	return len(s[i].BundleIdentifier) > len(s[j].BundleIdentifier)
}

// ByLength ...
type ByLength []string

// Len ..
func (s ByLength) Len() int {
	return len(s)
}

// Swap ...
func (s ByLength) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less ...
func (s ByLength) Less(i, j int) bool {
	return len(s[i]) > len(s[j])
}

// ExportOptionConfig ...
type ExportOptionConfig struct {
	Method                exportoptions.Method
	CodesignInfoMap       map[string]xcodeproj.CodeSignInfo
	Profiles              []profileutil.ProfileModel
	InstalledCertificates []certificateutil.CertificateInfosModel
}

// New ...
func New(method exportoptions.Method, codeSignInfoMap map[string]xcodeproj.CodeSignInfo) (ExportOptionConfig, error) {
	exportOptionConfig := ExportOptionConfig{
		Method:          method,
		CodesignInfoMap: codeSignInfoMap,
	}

	installedCerts, err := GetAllInstalledCertificates()
	if err != nil {
		return ExportOptionConfig{}, err
	}
	exportOptionConfig.InstalledCertificates = installedCerts

	exportOptionConfig.Profiles = []profileutil.ProfileModel{}

	if err := utils.WalkIOSProvProfilesPth(func(pth string) bool {
		profile, err := profileutil.ProfileFromFile(pth)
		if err != nil {
			log.Errorf("Failed to walk provisioning profiles, error: %s", err)
			os.Exit(1)
		}

		exportOptionConfig.Profiles = append(exportOptionConfig.Profiles, profile)
		return false
	}); err != nil {
		return ExportOptionConfig{}, err
	}

	sort.Sort(ByBundleIDLength(exportOptionConfig.Profiles))

	return exportOptionConfig, nil
}

// GenerateBundleIDProfileMap ...
func (exportOptionConfig ExportOptionConfig) GenerateBundleIDProfileMap() (certificateutil.CertificateInfosModel, map[string]profileutil.ProfileModel) {
	bundleIDTeamIDmap := map[string]xcodeproj.CodeSignInfo{}
	for _, val := range exportOptionConfig.CodesignInfoMap {
		bundleIDTeamIDmap[val.BundleIdentifier] = val
	}

	filtered := map[string]profileutil.ProfileModel{}

	groupedProfiles := map[string][]profileutil.ProfileModel{}

	for _, profile := range exportOptionConfig.Profiles {
		for _, embeddedCert := range profile.DeveloperCertificates {
			if embeddedCert.RawSubject == "" {
				continue
			}
			if _, ok := groupedProfiles[embeddedCert.RawSubject]; !ok {
				groupedProfiles[embeddedCert.RawSubject] = []profileutil.ProfileModel{}
			}
			groupedProfiles[embeddedCert.RawSubject] = append(groupedProfiles[embeddedCert.RawSubject], profile)
		}
	}

	for certSubject, profiles := range groupedProfiles {
		certSubjectFound := false
		for _, profile := range profiles {
			foundProfiles := map[string]profileutil.ProfileModel{}
			skipMatching := false
			for bundleIDToCheck, codesignInfo := range bundleIDTeamIDmap {
				if codesignInfo.ProvisioningProfileSpecifier == profile.Name {
					foundProfiles[bundleIDToCheck] = profile
					skipMatching = true
					continue
				}
			}
			if !skipMatching {
				for bundleIDToCheck, codesignInfo := range bundleIDTeamIDmap {
					if codesignInfo.ProvisioningProfile == profile.UUID {
						foundProfiles[bundleIDToCheck] = profile
						skipMatching = true
						continue
					}
				}
			}
			if !skipMatching {
				for bundleIDToCheck, codesignInfo := range bundleIDTeamIDmap {
					if glob.Glob(profile.BundleIdentifier, bundleIDToCheck) && exportOptionConfig.Method == profile.ExportType && profile.TeamIdentifier == codesignInfo.DevelopmentTeam {
						foundProfiles[bundleIDToCheck] = profile
						continue
					}
				}
			}
			if len(foundProfiles) >= len(bundleIDTeamIDmap) {
				certSubjectFound = true
				filtered = foundProfiles
				break
			}
		}
		if certSubjectFound {
			for _, cert := range exportOptionConfig.InstalledCertificates {
				if cert.RawSubject == certSubject {
					return cert, filtered
				}
			}
			break
		}
	}

	return certificateutil.CertificateInfosModel{}, nil
}

// GetAllInstalledCertificates ...
func GetAllInstalledCertificates() ([]certificateutil.CertificateInfosModel, error) {
	getCertificatesCmd := command.New("security", "find-certificate", "-a", "-p")
	certsPemOutput, err := getCertificatesCmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return nil, err
	}

	parsedCerts := []certificateutil.CertificateInfosModel{}

	for _, certPemContent := range strings.Split(certsPemOutput, "-----BEGIN CERTIFICATE-----") {
		if certPemContent == "" {
			continue
		}
		cmd := command.New("openssl", "x509", "-noout", "-enddate", "-subject")
		cmd.SetStdin(bytes.NewReader([]byte(fmt.Sprintf("-----BEGIN CERTIFICATE-----%s", certPemContent))))
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		if err != nil {
			return []certificateutil.CertificateInfosModel{}, fmt.Errorf("failed to read certificate infos, out: %s, error: %s", out, err)
		}
		parsedCert := parsePemOutput(out)
		if err != nil {
			return nil, err
		}
		if parsedCert.RawSubject != "" && parsedCert.RawEndDate != "" {
			parsedCerts = append(parsedCerts, parsedCert)
		}
	}

	return parsedCerts, nil
}

func parsePemOutput(out string) certificateutil.CertificateInfosModel {
	lines := strings.Split(out, "\n")
	if len(lines) < 2 {
		return certificateutil.CertificateInfosModel{}
	}

	certificateInfos := certificateutil.CertificateInfosModel{}

	// notAfter=Aug 15 14:15:19 2018 GMT
	endDateLine := strings.TrimSpace(lines[0])
	certificateInfos.RawEndDate = endDateLine
	endDatePattern := `notAfter=(?P<date>.*)`
	endDateRe := regexp.MustCompile(endDatePattern)
	if matches := endDateRe.FindStringSubmatch(endDateLine); len(matches) == 2 {
		endDateStr := matches[1]
		endDate, err := time.Parse("Jan 2 15:04:05 2006 MST", endDateStr)
		if err == nil {
			certificateInfos.EndDate = endDate
		}
	}

	// subject= /UID=5KN/CN=iPhone Developer: Bitrise Bot (T36)/OU=339/O=Bitrise Bot/C=US
	subjectLine := strings.TrimSpace(lines[1])
	certificateInfos.RawSubject = subjectLine
	certificateInfos.IsDevelopement = (strings.Contains(subjectLine, "Developer:") || strings.Contains(subjectLine, "Development:"))
	subjectPattern := `subject= /UID=(?P<userID>.*)/CN=(?P<commonName>.*)/OU=(?P<teamID>.*)/O=(?P<name>.*)/C=(?P<local>.*)`
	subjectRe := regexp.MustCompile(subjectPattern)
	if matches := subjectRe.FindStringSubmatch(subjectLine); len(matches) == 6 {
		userID := matches[1]
		commonName := matches[2]
		teamID := matches[3]
		name := matches[4]
		local := matches[5]

		certificateInfos.UserID = userID
		certificateInfos.CommonName = commonName
		certificateInfos.TeamID = teamID
		certificateInfos.Name = name
		certificateInfos.Local = local
		certificateInfos.IsDevelopement = (strings.Contains(commonName, "Developer:") || strings.Contains(commonName, "Development:"))
	}

	return certificateInfos
}
