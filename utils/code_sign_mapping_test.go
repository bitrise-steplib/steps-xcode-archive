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
	t.Log("certificate installed")
	{
		certificate := certificateutil.CertificateInfosModel{
			RawSubject: "subject= /UID=23442233441/CN=iPhone Developer: INSTALLED (679345FD33)/OU=671115FD33/O=My Company/C=US",
		}
		installedCertificates := []certificateutil.CertificateInfosModel{certificate}

		require.Equal(t, true, isCertificateInstalled(installedCertificates, certificate))
	}

	t.Log("certificate NOT installed")
	{
		installedCertificates := []certificateutil.CertificateInfosModel{certificateutil.CertificateInfosModel{
			RawSubject: "subject= /UID=23442233441/CN=iPhone Developer: INSTALLED (679345FD33)/OU=671115FD33/O=My Company/C=US",
		}}
		certificate := certificateutil.CertificateInfosModel{
			RawSubject: "subject= /UID=23442233441/CN=iPhone Developer: NOT INSTALLED (679345FD33)/OU=671115FD33/O=My Company/C=US",
		}

		require.Equal(t, false, isCertificateInstalled(installedCertificates, certificate))
	}

	t.Log("certificate NOT installed - no installed certificates")
	{
		installedCertificates := []certificateutil.CertificateInfosModel{}
		certificate := certificateutil.CertificateInfosModel{
			RawSubject: "subject= /UID=23442233441/CN=iPhone Developer: NOT INSTALLED (679345FD33)/OU=671115FD33/O=My Company/C=US",
		}

		require.Equal(t, false, isCertificateInstalled(installedCertificates, certificate))
	}
}

func TestCreateCertificateProfilesMapping(t *testing.T) {
	t.Log("1 certificate - 1 profile map")
	{
		certificate := certificateutil.CertificateInfosModel{
			RawSubject: "subject= /UID=23442233441/CN=iPhone Developer: User Name1 (679345FD33)/OU=671115FD33/O=My Company/C=US",
		}

		profile1 := profileutil.ProfileModel{
			DeveloperCertificates: []certificateutil.CertificateInfosModel{
				certificate,
			},
		}

		certificates := []certificateutil.CertificateInfosModel{certificate}
		profiles := []profileutil.ProfileModel{profile1}

		mapping := createCertificateProfilesMapping(profiles, certificates)
		expected := map[string][]profileutil.ProfileModel{
			"subject= /UID=23442233441/CN=iPhone Developer: User Name1 (679345FD33)/OU=671115FD33/O=My Company/C=US": profiles,
		}
		require.Equal(t, expected, mapping)
	}

	t.Log("1 certificate - 1 profile map")
	{
		certificate := certificateutil.CertificateInfosModel{
			RawSubject: "subject= /UID=23442233441/CN=iPhone Developer: User Name1 (679345FD33)/OU=671115FD33/O=My Company/C=US",
		}

		profile := profileutil.ProfileModel{
			DeveloperCertificates: []certificateutil.CertificateInfosModel{
				certificate,
			},
		}

		certificates := []certificateutil.CertificateInfosModel{certificate}
		profiles := []profileutil.ProfileModel{profile}

		mapping := createCertificateProfilesMapping(profiles, certificates)
		expected := map[string][]profileutil.ProfileModel{
			"subject= /UID=23442233441/CN=iPhone Developer: User Name1 (679345FD33)/OU=671115FD33/O=My Company/C=US": profiles,
		}
		require.Equal(t, expected, mapping)
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
