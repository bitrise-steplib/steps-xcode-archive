package localcodesignasset

import (
	"github.com/bitrise-io/go-xcode/autocodesign"
	"github.com/bitrise-io/go-xcode/autocodesign/devportalclient/appstoreconnect"
)

func certificateSerials(certsByType map[appstoreconnect.CertificateType][]autocodesign.Certificate, distrType autocodesign.DistributionType) []string {
	certType := autocodesign.CertificateTypeByDistribution[distrType]
	certs := certsByType[certType]

	var serials []string
	for _, cert := range certs {
		serials = append(serials, cert.CertificateInfo.Serial)
	}

	return serials
}

func remove(slice []string, i int) []string {
	copy(slice[i:], slice[i+1:])
	return slice[:len(slice)-1]
}

func contains(array []string, element string) bool {
	for _, item := range array {
		if item == element {
			return true
		}
	}
	return false
}
