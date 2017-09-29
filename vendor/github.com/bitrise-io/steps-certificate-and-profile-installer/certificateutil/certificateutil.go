package certificateutil

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

// CertificateInfosModel ...
type CertificateInfosModel struct {
	UserID         string
	CommonName     string
	TeamID         string
	Name           string
	Local          string
	EndDate        time.Time
	RawSubject     string
	RawEndDate     string
	IsDevelopement bool
}

func convertP12ToPem(p12Pth, password string) (string, error) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("__pem__")
	if err != nil {
		return "", err
	}

	pemPth := filepath.Join(tmpDir, "certificate.pem")
	if out, err := command.New("openssl", "pkcs12", "-in", p12Pth, "-out", pemPth, "-nodes", "-passin", "pass:"+password).RunAndReturnTrimmedCombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to convert .p12 certificate to .pem file, out: %s, error: %s", out, err)
	}

	return pemPth, nil
}

// CertificateInfosFromP12 ...
func CertificateInfosFromP12(p12Pth, password string) ([]CertificateInfosModel, error) {
	pemPth, err := convertP12ToPem(p12Pth, password)
	if err != nil {
		return []CertificateInfosModel{}, err
	}

	content, err := fileutil.ReadBytesFromFile(pemPth)
	if err != nil {
		return []CertificateInfosModel{}, err
	}

	return CertificateInfosFromPemContent(content)
}

// CertificateInfosFromPemContent ...
func CertificateInfosFromPemContent(pemContent []byte) ([]CertificateInfosModel, error) {
	certInfoModels := []CertificateInfosModel{}

	pems := []string{}

	for _, pem := range strings.Split(string(pemContent), "Bag Attributes") {
		if strings.Contains(pem, "subject=") {
			pems = append(pems, fmt.Sprintf("Bag Attributes\n%s", pem))
		}
	}

	for _, pem := range pems {
		cmd := command.New("openssl", "x509", "-noout", "-enddate", "-subject")
		cmd.SetStdin(bytes.NewReader([]byte(pem)))
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		if err != nil {
			return []CertificateInfosModel{}, fmt.Errorf("failed to read certificate infos, out: %s, error: %s", out, err)
		}

		certInfoModel, err := parsePemOutput(out)
		if err != nil {
			return []CertificateInfosModel{}, fmt.Errorf("failed to parse pem output, out: %s, error: %s", out, err)
		}

		certInfoModels = append(certInfoModels, certInfoModel)
	}

	return certInfoModels, nil
}

// CertificateInfosFromDerContent ...
func CertificateInfosFromDerContent(pemContent []byte) (CertificateInfosModel, error) {
	cmd := command.New("openssl", "x509", "-noout", "-enddate", "-subject", "-inform", "DER")
	cmd.SetStdin(bytes.NewReader(pemContent))
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return CertificateInfosModel{}, fmt.Errorf("failed to read certificate infos, out: %s, error: %s", out, err)
	}

	return parsePemOutput(out)
}

func parsePemOutput(out string) (CertificateInfosModel, error) {
	lines := strings.Split(out, "\n")
	if len(lines) < 2 {
		return CertificateInfosModel{}, fmt.Errorf("failed to parse certificate infos")
	}

	certificateInfos := CertificateInfosModel{}

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
		} else {
			log.Warnf("Failed to parse certificate endDate, error: %s", err)
		}
	} else {
		log.Warnf("Failed to find pattern in %s, matches: %d", endDateLine, len(matches))
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
	} else {
		log.Warnf("Failed to find pattern in %s, matches: %d", subjectLine, len(matches))
	}

	return certificateInfos, nil
}

func (certInfo CertificateInfosModel) String() string {
	certInfoString := ""

	if certInfo.CommonName != "" && certInfo.TeamID != "" {
		certInfoString += fmt.Sprintf("- TeamID: %s\n", certInfo.TeamID)
	} else {
		certInfoString += fmt.Sprintf("- RawSubject: %s\n", certInfo.RawSubject)
	}

	if !certInfo.EndDate.IsZero() {
		certInfoString += fmt.Sprintf("- EndDate: %s\n", certInfo.EndDate)
	} else {
		certInfoString += fmt.Sprintf("- RawEndDate: %s\n", certInfo.RawEndDate)
	}
	certInfoString += fmt.Sprintf("- IsDevelopement: %t", certInfo.IsDevelopement)

	return certInfoString
}
