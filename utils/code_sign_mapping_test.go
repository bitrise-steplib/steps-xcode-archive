package utils

import (
	"testing"

	"github.com/bitrise-io/steps-certificate-and-profile-installer/certificateutil"
	"github.com/bitrise-io/steps-certificate-and-profile-installer/profileutil"
	"github.com/bitrise-tools/go-xcode/exportoptions"
	"github.com/stretchr/testify/require"
)

func TestIsCertificateInstalled(t *testing.T) {
	t.Log("isCertificateInstalled")
	{
		certNotInstalled1 := certificateutil.CertificateInfosModel{
			RawSubject:     "subject= /UID=23442233441/CN=iPhone Developer: User Name (679345FD33)/OU=1010101010/O=My Company/C=US",
			TeamID:         "1010101010",
			Name:           "My Company",
			CommonName:     "iPhone Developer: User Name (679345FD33)",
			IsDevelopement: true,
		}

		certNotInstalled2 := certificateutil.CertificateInfosModel{
			RawSubject:     "subject= /UID=SD53FS3A2N/CN=iPhone Developer: Other User (DBVR53G453)/OU=671115FD33/O=My Company/C=US",
			TeamID:         "671115FD33",
			Name:           "My Company",
			CommonName:     "iPhone Developer: Other User (DBVR53G453)",
			IsDevelopement: true,
		}

		certNotInstalled3 := certificateutil.CertificateInfosModel{
			RawSubject:     "subject= /UID=58HK6K3K693/CN=iPhone Developer: Another User (LGK9578DIXM)/OU=671115FD33/O=My Company/C=US",
			TeamID:         "671115FD33",
			Name:           "My Company",
			CommonName:     "iPhone Developer: Another User (LGK9578DIXM)",
			IsDevelopement: true,
		}

		certs, _ := getTestCertificatesAndProfiles()
		for _, cert := range certs {
			require.Equal(t, true, isCertificateInstalled(certs, cert))
		}

		require.Equal(t, false, isCertificateInstalled(certs, certNotInstalled1))
		require.Equal(t, false, isCertificateInstalled(certs, certNotInstalled2))
		require.Equal(t, false, isCertificateInstalled(certs, certNotInstalled3))

		require.NoError(t, nil)
	}
}

func TestResolveCodeSignMapping(t *testing.T) {
	t.Log("ResolveCodeSignMapping")
	{
		certs, profs := getTestCertificatesAndProfiles()

		profileGroups := ResolveCodeSignMapping([]string{"com.bitrise.testbundleid", "com.bitrise.testbundleid.notification", "com.bitrise.testbundleid.widget"}, exportoptions.MethodDevelopment, profs, certs)

		require.Equal(t, 2, len(profileGroups))
		require.Equal(t, true, (profileGroups[0].Profiles["com.bitrise.testbundleid"].TeamIdentifier == "72SA8V3WYL" || profileGroups[0].Profiles["com.bitrise.testbundleid"].TeamIdentifier == "671115FD33"))
		require.Equal(t, true, (profileGroups[1].Profiles["com.bitrise.testbundleid"].TeamIdentifier == "72SA8V3WYL" || profileGroups[1].Profiles["com.bitrise.testbundleid"].TeamIdentifier == "671115FD33"))
		require.Equal(t, true, (profileGroups[0].Profiles["com.bitrise.testbundleid"].TeamIdentifier != profileGroups[1].Profiles["com.bitrise.testbundleid"].TeamIdentifier))

		profileGroups = ResolveCodeSignMapping([]string{"com.bitrise.testbundleid"}, exportoptions.MethodDevelopment, profs, certs)

		require.Equal(t, 2, len(profileGroups))
		require.Equal(t, true, (profileGroups[0].Profiles["com.bitrise.testbundleid"].TeamIdentifier == "72SA8V3WYL" || profileGroups[0].Profiles["com.bitrise.testbundleid"].TeamIdentifier == "671115FD33"))
		require.Equal(t, true, (profileGroups[1].Profiles["com.bitrise.testbundleid"].TeamIdentifier == "72SA8V3WYL" || profileGroups[1].Profiles["com.bitrise.testbundleid"].TeamIdentifier == "671115FD33"))
		require.Equal(t, true, (profileGroups[0].Profiles["com.bitrise.testbundleid"].TeamIdentifier != profileGroups[1].Profiles["com.bitrise.testbundleid"].TeamIdentifier))

		profileGroups = ResolveCodeSignMapping([]string{"com.bitrise.testbundleid", "com.bitrise.testbundleid.notification", "com.bitrise.testbundleid.widget", "com.bitrise.testbundleid.onlywildcard"}, exportoptions.MethodDevelopment, profs, certs)

		require.Equal(t, 1, len(profileGroups))
		require.Equal(t, true, (profileGroups[0].Profiles["com.bitrise.testbundleid.onlywildcard"].TeamIdentifier == "72SA8V3WYL"))

		profileGroups = ResolveCodeSignMapping([]string{"com.bitrise.anything"}, exportoptions.MethodDevelopment, profs, certs)

		require.Equal(t, 1, len(profileGroups))
		require.Equal(t, true, (profileGroups[0].Profiles["com.bitrise.anything"].TeamIdentifier == "72SA8V3WYL"))

		profileGroups = ResolveCodeSignMapping([]string{"com.bitrise.anything", "com.bitrise.another"}, exportoptions.MethodDevelopment, profs, certs)

		require.Equal(t, 1, len(profileGroups))
		require.Equal(t, true, (profileGroups[0].Profiles["com.bitrise.anything"].TeamIdentifier == "72SA8V3WYL"))
		require.Equal(t, true, (profileGroups[0].Profiles["com.bitrise.another"].TeamIdentifier == "72SA8V3WYL"))
	}
}

func getTestCertificatesAndProfiles() ([]certificateutil.CertificateInfosModel, []profileutil.ProfileModel) {
	certs := []certificateutil.CertificateInfosModel{}

	cert1 := certificateutil.CertificateInfosModel{
		RawSubject:     "subject= /UID=23442233441/CN=iPhone Developer: User Name (679345FD33)/OU=671115FD33/O=My Company/C=US",
		TeamID:         "671115FD33",
		Name:           "My Company",
		CommonName:     "iPhone Developer: User Name (679345FD33)",
		IsDevelopement: true,
	}
	certs = append(certs, cert1)

	cert2 := certificateutil.CertificateInfosModel{
		RawSubject:     "subject= /UID=671115FD33/CN=iPhone Distribution: My Company (671115FD33)/OU=671115FD33/O=My Company/C=US",
		TeamID:         "671115FD33",
		CommonName:     "iPhone Distribution: My Company (671115FD33)",
		Name:           "My Company",
		IsDevelopement: false,
	}
	certs = append(certs, cert2)

	cert3 := certificateutil.CertificateInfosModel{
		RawSubject:     "subject= /UID=647N2UNN67/CN=iPhone Developer: Bitrise Bot (VV2J4SV8V4)/OU=72SA8V3WYL/O=BITFALL FEJLESZTO KORLATOLT FELELOSSEGU TARSASAG/C=US",
		TeamID:         "72SA8V3WYL",
		Name:           "BITFALL FEJLESZTO KORLATOLT FELELOSSEGU TARSASAG",
		CommonName:     "iPhone Developer: Bitrise Bot (VV2J4SV8V4)",
		IsDevelopement: true,
	}
	certs = append(certs, cert3)

	cert4 := certificateutil.CertificateInfosModel{
		RawSubject:     "subject= /UID=SD53FS3A2N/CN=iPhone Developer: Other User (DBVR53G453)/OU=671115FD33/O=My Company/C=US",
		TeamID:         "671115FD33",
		Name:           "My Company",
		CommonName:     "iPhone Developer: Other User (DBVR53G453)",
		IsDevelopement: true,
	}

	cert5 := certificateutil.CertificateInfosModel{
		RawSubject:     "subject= /UID=58HK6K3K693/CN=iPhone Developer: Another User (LGK9578DIXM)/OU=671115FD33/O=My Company/C=US",
		TeamID:         "671115FD33",
		Name:           "My Company",
		CommonName:     "iPhone Developer: Another User (LGK9578DIXM)",
		IsDevelopement: true,
	}

	profs := []profileutil.ProfileModel{}

	prof1 := profileutil.ProfileModel{
		TeamIdentifier:   "671115FD33",
		BundleIdentifier: "com.bitrise.testbundleid.notification",
		ExportType:       "development",
		Name:             "iOS Team Provisioning Profile: com.bitrise.testbundleid.notification",
		DeveloperCertificates: []certificateutil.CertificateInfosModel{
			cert1,
			cert4,
			cert5,
		},
	}
	profs = append(profs, prof1)

	prof2 := profileutil.ProfileModel{
		TeamIdentifier:   "671115FD33",
		BundleIdentifier: "com.bitrise.testbundleid.notification",
		ExportType:       "app-store",
		Name:             "XC iOS: com.bitrise.testbundleid.notification",
		DeveloperCertificates: []certificateutil.CertificateInfosModel{
			cert2,
		},
	}
	profs = append(profs, prof2)

	prof3 := profileutil.ProfileModel{
		TeamIdentifier:   "671115FD33",
		BundleIdentifier: "com.bitrise.testbundleid.widget",
		ExportType:       "app-store",
		Name:             "XC iOS: com.bitrise.testbundleid.widget",
		DeveloperCertificates: []certificateutil.CertificateInfosModel{
			cert2,
		},
	}
	profs = append(profs, prof3)

	prof4 := profileutil.ProfileModel{
		TeamIdentifier:   "671115FD33",
		BundleIdentifier: "com.bitrise.testbundleid",
		ExportType:       "development",
		Name:             "iOS Team Provisioning Profile: com.bitrise.testbundleid",
		DeveloperCertificates: []certificateutil.CertificateInfosModel{
			cert1,
			cert4,
			cert5,
		},
	}
	profs = append(profs, prof4)

	prof5 := profileutil.ProfileModel{
		TeamIdentifier:   "72SA8V3WYL",
		BundleIdentifier: "*",
		ExportType:       "development",
		Name:             "BitriseBot-Wildcard",
		DeveloperCertificates: []certificateutil.CertificateInfosModel{
			cert3,
		},
	}
	profs = append(profs, prof5)

	prof6 := profileutil.ProfileModel{
		TeamIdentifier:   "671115FD33",
		BundleIdentifier: "com.bitrise.testbundleid",
		ExportType:       "app-store",
		Name:             "XC iOS: com.bitrise.testbundleid",
		DeveloperCertificates: []certificateutil.CertificateInfosModel{
			cert2,
		},
	}
	profs = append(profs, prof6)

	prof7 := profileutil.ProfileModel{
		TeamIdentifier:   "671115FD33",
		BundleIdentifier: "com.bitrise.testbundleid.widget",
		ExportType:       "development",
		Name:             "iOS Team Provisioning Profile: com.bitrise.testbundleid.widget",
		DeveloperCertificates: []certificateutil.CertificateInfosModel{
			cert1,
			cert4,
			cert5,
		},
	}
	profs = append(profs, prof7)

	return certs, profs
}
