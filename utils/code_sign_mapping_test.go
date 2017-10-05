package utils

import (
	"fmt"
	"testing"

	"github.com/bitrise-io/steps-certificate-and-profile-installer/certificateutil"
	"github.com/bitrise-io/steps-certificate-and-profile-installer/profileutil"
	"github.com/bitrise-tools/go-xcode/exportoptions"
	"github.com/stretchr/testify/require"
)

func TestIsCertificateInstalled(t *testing.T) {
	t.Log("isCertificateInstalled")
	{
		installedCerts := []certificateutil.CertificateInfosModel{}

		cert1 := certificateutil.CertificateInfosModel{
			RawSubject: "subject= /UID=23442233441/CN=iPhone Developer: User Name (679345FD33)/OU=671115FD33/O=My Company/C=US",
		}
		installedCerts = append(installedCerts, cert1)

		cert2 := certificateutil.CertificateInfosModel{
			RawSubject: "subject= /UID=671115FD33/CN=iPhone Distribution: My Company (671115FD33)/OU=671115FD33/O=My Company/C=US",
		}
		installedCerts = append(installedCerts, cert2)

		cert3 := certificateutil.CertificateInfosModel{
			RawSubject: "subject= /UID=647N2UNN67/CN=iPhone Developer: Bitrise Bot (VV2J4SV8V4)/OU=72SA8V3WYL/O=BITFALL FEJLESZTO KORLATOLT FELELOSSEGU TARSASAG/C=US",
		}
		installedCerts = append(installedCerts, cert3)

		certNotInstalled1 := certificateutil.CertificateInfosModel{
			RawSubject: "subject= /UID=23442233441/CN=iPhone Developer: User Name (679345FD33)/OU=1010101010/O=My Company/C=US",
		}

		certNotInstalled2 := certificateutil.CertificateInfosModel{
			RawSubject: "subject= /UID=SD53FS3A2N/CN=iPhone Developer: Other User (DBVR53G453)/OU=671115FD33/O=My Company/C=US",
		}

		certNotInstalled3 := certificateutil.CertificateInfosModel{
			RawSubject: "subject= /UID=58HK6K3K693/CN=iPhone Developer: Another User (LGK9578DIXM)/OU=671115FD33/O=My Company/C=US",
		}

		for _, cert := range installedCerts {
			require.Equal(t, true, isCertificateInstalled(installedCerts, cert))
		}

		require.Equal(t, false, isCertificateInstalled(installedCerts, certNotInstalled1))
		require.Equal(t, false, isCertificateInstalled(installedCerts, certNotInstalled2))
		require.Equal(t, false, isCertificateInstalled(installedCerts, certNotInstalled3))
	}
}

func TestResolveCodeSignGroupItems(t *testing.T) {
	t.Log("ResolveCodeSignGroupItems")
	{
		bundleID := []string{
			"com.tomi",
			"com.godrei",
		}

		method := exportoptions.MethodDevelopment

		cert1 := certificateutil.CertificateInfosModel{
			RawSubject: "subject= /UID=23442233441/CN=iPhone Developer: User Name1 (679345FD33)/OU=671115FD33/O=My Company/C=US",
		}
		cert2 := certificateutil.CertificateInfosModel{
			RawSubject: "subject= /UID=23442233441/CN=iPhone Developer: User Name2 (679345FD33)/OU=671115FD33/O=My Company/C=US",
		}
		certs := []certificateutil.CertificateInfosModel{cert1, cert2}

		profs := []profileutil.ProfileModel{
			profileutil.ProfileModel{
				BundleIdentifier: "com.tomi",
				ExportType:       "development",
				DeveloperCertificates: []certificateutil.CertificateInfosModel{
					cert1,
				},
			},
			profileutil.ProfileModel{
				BundleIdentifier: "*",
				ExportType:       "development",
				DeveloperCertificates: []certificateutil.CertificateInfosModel{
					cert1,
				},
			},
			profileutil.ProfileModel{
				BundleIdentifier: "*",
				ExportType:       "development",
				DeveloperCertificates: []certificateutil.CertificateInfosModel{
					cert2,
				},
			},
		}

		profileGroups := ResolveCodeSignGroupItems(bundleID, method, profs, certs)
		for _, group := range profileGroups {
			t.Logf("cert: %s", group.Certificate.RawSubject)
			t.Logf("Profiles: %s", group.BundleIDProfileMap)
			fmt.Println()
		}
	}
}
