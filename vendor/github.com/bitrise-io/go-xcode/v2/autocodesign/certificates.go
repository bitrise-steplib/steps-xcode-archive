package autocodesign

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-xcode/certificateutil"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/devportalclient/appstoreconnect"
)

func selectCertificatesAndDistributionTypes(certificateSource DevPortalClient, certs []certificateutil.CertificateInfoModel, distribution DistributionType, signUITestTargets bool, verboseLog bool) (map[appstoreconnect.CertificateType][]Certificate, []DistributionType, error) {
	certType, ok := CertificateTypeByDistribution[distribution]
	if !ok {
		panic(fmt.Sprintf("no valid certificate provided for distribution type: %s", distribution))
	}

	distrTypes := []DistributionType{distribution}
	requiredCertTypes := map[appstoreconnect.CertificateType]bool{certType: true}
	if distribution != Development {
		distrTypes = append(distrTypes, Development)

		if signUITestTargets {
			log.Warnf("UITest target requires development code signing in addition to the specified %s code signing", distribution)
			requiredCertTypes[appstoreconnect.IOSDevelopment] = true
		} else {
			requiredCertTypes[appstoreconnect.IOSDevelopment] = false
		}
	}

	certsByType, err := getValidCertificates(certs, certificateSource, requiredCertTypes, verboseLog)
	if err != nil {
		if missingCertErr, ok := err.(missingCertificateError); ok {
			return nil, nil, &DetailedError{
				ErrorMessage:   "",
				Title:          fmt.Sprintf("No valid %s type certificates uploaded", missingCertErr.Type),
				Description:    fmt.Sprintf("Maybe you forgot to provide a(n) %s type certificate.", missingCertErr.Type),
				Recommendation: fmt.Sprintf("Upload a %s type certificate (.p12) on the Code Signing tab of the Workflow Editor.", missingCertErr.Type),
			}
		}
		return nil, nil, fmt.Errorf("failed to get valid certificates: %s", err)
	}

	if len(certsByType) == 1 && distribution != Development {
		// remove development distribution if there is no development certificate uploaded
		distrTypes = []DistributionType{distribution}
	}
	log.Printf("ensuring codesigning files for distribution types: %s", distrTypes)

	return certsByType, distrTypes, nil
}

func getValidCertificates(localCertificates []certificateutil.CertificateInfoModel, client DevPortalClient, requiredCertificateTypes map[appstoreconnect.CertificateType]bool, isDebugLog bool) (map[appstoreconnect.CertificateType][]Certificate, error) {
	typeToLocalCerts, err := GetValidLocalCertificates(localCertificates)
	if err != nil {
		return nil, err
	}

	log.Debugf("Certificates required for Development: %t; Distribution: %t", requiredCertificateTypes[appstoreconnect.IOSDevelopment], requiredCertificateTypes[appstoreconnect.IOSDistribution])

	for certificateType, required := range requiredCertificateTypes {
		if required && len(typeToLocalCerts[certificateType]) == 0 {
			return map[appstoreconnect.CertificateType][]Certificate{}, missingCertificateError{certificateType}
		}
	}

	// only for debugging
	if isDebugLog {
		if err := logAllAPICertificates(client); err != nil {
			log.Debugf("Failed to log all Developer Portal certificates: %s", err)
		}
	}

	validAPICertificates := map[appstoreconnect.CertificateType][]Certificate{}
	for certificateType, validLocalCertificates := range typeToLocalCerts {
		matchingCertificates, err := matchLocalToAPICertificates(client, validLocalCertificates)
		if err != nil {
			return nil, err
		}

		if len(matchingCertificates) > 0 {
			log.Debugf("Certificates type %s has matches on Developer Portal:", certificateType)
			for _, cert := range matchingCertificates {
				log.Debugf("- %s", cert.CertificateInfo)
			}
		}

		if requiredCertificateTypes[certificateType] && len(matchingCertificates) == 0 {
			return nil, fmt.Errorf("not found any of the following %s certificates on Developer Portal:\n%s", certificateType, certsToString(validLocalCertificates))
		}

		if len(matchingCertificates) > 0 {
			validAPICertificates[certificateType] = matchingCertificates
		}
	}

	return validAPICertificates, nil
}

// GetValidLocalCertificates returns validated and deduplicated local certificates
func GetValidLocalCertificates(certificates []certificateutil.CertificateInfoModel) (map[appstoreconnect.CertificateType][]certificateutil.CertificateInfoModel, error) {
	preFilteredCerts := certificateutil.FilterValidCertificateInfos(certificates)

	if len(preFilteredCerts.InvalidCertificates) != 0 {
		log.Warnf("Ignoring expired or not yet valid certificates: %s", preFilteredCerts.InvalidCertificates)
	}
	if len(preFilteredCerts.DuplicatedCertificates) != 0 {
		log.Warnf("Ignoring duplicated certificates with the same name: %s", preFilteredCerts.DuplicatedCertificates)
	}

	log.Debugf("Valid and deduplicated certificates:\n%s", certsToString(preFilteredCerts.ValidCertificates))

	localCertificates := map[appstoreconnect.CertificateType][]certificateutil.CertificateInfoModel{}
	for _, certType := range []appstoreconnect.CertificateType{appstoreconnect.IOSDevelopment, appstoreconnect.IOSDistribution} {
		localCertificates[certType] = filterCertificates(preFilteredCerts.ValidCertificates, certType)
	}

	log.Debugf("Valid and deduplicated certificates:\n%s", certsToString(preFilteredCerts.ValidCertificates))

	return localCertificates, nil
}

// matchLocalToAPICertificates ...
func matchLocalToAPICertificates(client DevPortalClient, localCertificates []certificateutil.CertificateInfoModel) ([]Certificate, error) {
	var matchingCertificates []Certificate

	for _, localCert := range localCertificates {
		cert, err := client.QueryCertificateBySerial(*localCert.Certificate.SerialNumber)
		if err != nil {
			log.Warnf("Certificate (%s) not found on Developer Portal: %s", localCert, err)
			continue
		}
		cert.CertificateInfo = localCert

		log.Debugf("Certificate (%s) found with ID: %s", localCert, cert.ID)

		matchingCertificates = append(matchingCertificates, cert)
	}

	return matchingCertificates, nil
}

// logAllAPICertificates ...
func logAllAPICertificates(client DevPortalClient) error {
	certificates, err := client.QueryAllIOSCertificates()
	if err != nil {
		return fmt.Errorf("failed to query certificates on Developer Portal: %s", err)
	}

	for certType, certs := range certificates {
		log.Debugf("Developer Portal %s certificates:", certType)
		for _, cert := range certs {
			log.Debugf("- %s", cert.CertificateInfo)
		}
	}

	return nil
}

// filterCertificates returns the certificates matching to the given common name, developer team ID, and distribution type.
func filterCertificates(certificates []certificateutil.CertificateInfoModel, certificateType appstoreconnect.CertificateType) []certificateutil.CertificateInfoModel {
	// filter by distribution type
	var filteredCertificates []certificateutil.CertificateInfoModel
	for _, certificate := range certificates {
		if certificateType == appstoreconnect.IOSDistribution && isDistributionCertificate(certificate) {
			filteredCertificates = append(filteredCertificates, certificate)
		} else if certificateType == appstoreconnect.IOSDevelopment && !isDistributionCertificate(certificate) {
			filteredCertificates = append(filteredCertificates, certificate)
		}
	}

	log.Debugf("Valid certificates with type %s:\n%s", certificateType, certsToString(filteredCertificates))

	if len(filteredCertificates) == 0 {
		return nil
	}

	log.Debugf("Valid certificates with type %s:\n%s", certificateType, certsToString(filteredCertificates))

	if len(filteredCertificates) == 0 {
		return nil
	}

	log.Debugf("Valid certificates with type %s\n%s ", certificateType, certsToString(filteredCertificates))

	return filteredCertificates
}

func isDistributionCertificate(cert certificateutil.CertificateInfoModel) bool {
	// Apple certificate types: https://help.apple.com/xcode/mac/current/#/dev80c6204ec)
	return strings.HasPrefix(strings.ToLower(cert.CommonName), strings.ToLower("iPhone Distribution")) ||
		strings.HasPrefix(strings.ToLower(cert.CommonName), strings.ToLower("Apple Distribution"))
}

func certsToString(certs []certificateutil.CertificateInfoModel) (s string) {
	for i, cert := range certs {
		s += "- "
		s += cert.String()
		if i < len(certs)-1 {
			s += "\n"
		}
	}
	return
}
