package utils

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/steps-certificate-and-profile-installer/certificateutil"
)

// InstalledCertificates ...
func InstalledCertificates() ([]certificateutil.CertificateInfosModel, error) {
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
