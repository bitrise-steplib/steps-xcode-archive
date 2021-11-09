package autocodesign

import (
	"fmt"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-io/go-xcode/autocodesign/devportalclient/appstoreconnect"
	"github.com/bitrise-io/go-xcode/profileutil"
	"github.com/bitrise-io/go-xcode/xcodeproject/serialized"
)

func appIDName(bundleID string) string {
	prefix := ""
	if strings.HasSuffix(bundleID, ".*") {
		prefix = "Wildcard "
	}
	r := strings.NewReplacer(".", " ", "_", " ", "-", " ", "*", " ")
	return prefix + "Bitrise " + r.Replace(bundleID)
}

func ensureProfiles(profileClient DevPortalClient, distrTypes []DistributionType,
	certsByType map[appstoreconnect.CertificateType][]Certificate, app AppLayout,
	devPortalDeviceIDs []string, minProfileDaysValid int) (map[DistributionType]AppCodesignAssets, error) {
	// Ensure Profiles
	codesignAssetsByDistributionType := map[DistributionType]AppCodesignAssets{}

	bundleIDByBundleIDIdentifer := map[string]*appstoreconnect.BundleID{}

	containersByBundleID := map[string][]string{}

	profileManager := profileManager{
		client:                      profileClient,
		bundleIDByBundleIDIdentifer: bundleIDByBundleIDIdentifer,
		containersByBundleID:        containersByBundleID,
	}

	for _, distrType := range distrTypes {
		fmt.Println()
		log.Infof("Checking %s provisioning profiles", distrType)
		certType := CertificateTypeByDistribution[distrType]
		certs := certsByType[certType]

		if len(certs) == 0 {
			return nil, fmt.Errorf("no valid certificate provided for distribution type: %s", distrType)
		} else if len(certs) > 1 {
			log.Warnf("Multiple certificates provided for distribution type: %s", distrType)
			for _, c := range certs {
				log.Warnf("- %s", c.CertificateInfo.CommonName)
			}
			log.Warnf("Using: %s", certs[0].CertificateInfo.CommonName)
		}
		log.Debugf("Using certificate for distribution type %s (certificate type %s): %s", distrType, certType, certs[0])

		codesignAssets := AppCodesignAssets{
			ArchivableTargetProfilesByBundleID: map[string]Profile{},
			UITestTargetProfilesByBundleID:     map[string]Profile{},
			Certificate:                        certs[0].CertificateInfo,
		}

		var certIDs []string
		for _, cert := range certs {
			certIDs = append(certIDs, cert.ID)
		}

		platformProfileTypes, ok := PlatformToProfileTypeByDistribution[app.Platform]
		if !ok {
			return nil, fmt.Errorf("no profiles for platform: %s", app.Platform)
		}

		profileType := platformProfileTypes[distrType]

		for bundleIDIdentifier, entitlements := range app.EntitlementsByArchivableTargetBundleID {
			var profileDeviceIDs []string
			if distributionTypeRequiresDeviceList([]DistributionType{distrType}) {
				profileDeviceIDs = devPortalDeviceIDs
			}

			profile, err := profileManager.ensureProfile(profileType, bundleIDIdentifier, entitlements, certIDs, profileDeviceIDs, minProfileDaysValid)
			if err != nil {
				return nil, err
			}
			codesignAssets.ArchivableTargetProfilesByBundleID[bundleIDIdentifier] = *profile
		}

		if len(app.UITestTargetBundleIDs) > 0 && distrType == Development {
			// Capabilities are not supported for UITest targets.
			// Xcode managed signing uses Wildcard Provisioning Profiles for UITest target signing.
			for _, bundleIDIdentifier := range app.UITestTargetBundleIDs {
				wildcardBundleID, err := createWildcardBundleID(bundleIDIdentifier)
				if err != nil {
					return nil, fmt.Errorf("could not create wildcard bundle id: %s", err)
				}

				// Capabilities are not supported for UITest targets.
				profile, err := profileManager.ensureProfile(profileType, wildcardBundleID, nil, certIDs, devPortalDeviceIDs, minProfileDaysValid)
				if err != nil {
					return nil, err
				}
				codesignAssets.UITestTargetProfilesByBundleID[bundleIDIdentifier] = *profile
			}
		}

		codesignAssetsByDistributionType[distrType] = codesignAssets
	}

	if len(profileManager.containersByBundleID) > 0 {
		iCloudContainers := ""
		for bundleID, containers := range containersByBundleID {
			iCloudContainers = fmt.Sprintf("%s, containers:\n", bundleID)
			for _, container := range containers {
				iCloudContainers += fmt.Sprintf("- %s\n", container)
			}
			iCloudContainers += "\n"
		}

		return nil, &DetailedError{
			ErrorMessage:   "",
			Title:          "Unable to automatically assign iCloud containers to the following app IDs:",
			Description:    iCloudContainers,
			Recommendation: "You have to manually add the listed containers to your app ID at: https://developer.apple.com/account/resources/identifiers/list.",
		}
	}

	return codesignAssetsByDistributionType, nil
}

type profileManager struct {
	client                      DevPortalClient
	bundleIDByBundleIDIdentifer map[string]*appstoreconnect.BundleID
	containersByBundleID        map[string][]string
}

func (m profileManager) ensureBundleID(bundleIDIdentifier string, entitlements Entitlements) (*appstoreconnect.BundleID, error) {
	fmt.Println()
	log.Infof("  Searching for app ID for bundle ID: %s", bundleIDIdentifier)

	bundleID, ok := m.bundleIDByBundleIDIdentifer[bundleIDIdentifier]
	if !ok {
		var err error
		bundleID, err = m.client.FindBundleID(bundleIDIdentifier)
		if err != nil {
			return nil, fmt.Errorf("failed to find bundle ID: %s", err)
		}
	}

	if bundleID == nil && isAppClip(entitlements) {
		return nil, ErrAppClipAppID{}
	}

	if bundleID != nil {
		log.Printf("  app ID found: %s", bundleID.Attributes.Name)

		m.bundleIDByBundleIDIdentifer[bundleIDIdentifier] = bundleID

		// Check if BundleID is sync with the project
		err := m.client.CheckBundleIDEntitlements(*bundleID, entitlements)
		if err != nil {
			if mErr, ok := err.(NonmatchingProfileError); ok {
				if isAppClip(entitlements) && hasSignInWithAppleEntitlement(entitlements) {
					return nil, ErrAppClipAppIDWithAppleSigning{}
				}

				log.Warnf("  app ID capabilities invalid: %s", mErr.Reason)
				log.Warnf("  app ID capabilities are not in sync with the project capabilities, synchronizing...")
				if err := m.client.SyncBundleID(*bundleID, entitlements); err != nil {
					return nil, fmt.Errorf("failed to update bundle ID capabilities: %s", err)
				}

				return bundleID, nil
			}

			return nil, fmt.Errorf("failed to validate bundle ID: %s", err)
		}

		log.Printf("  app ID capabilities are in sync with the project capabilities")

		return bundleID, nil
	}

	// Create BundleID
	log.Warnf("  app ID not found, generating...")

	bundleID, err := m.client.CreateBundleID(bundleIDIdentifier, appIDName(bundleIDIdentifier))
	if err != nil {
		return nil, fmt.Errorf("failed to create bundle ID: %s", err)
	}

	containers, err := entitlements.ICloudContainers()
	if err != nil {
		return nil, fmt.Errorf("failed to get list of iCloud containers: %s", err)
	}

	if len(containers) > 0 {
		m.containersByBundleID[bundleIDIdentifier] = containers
		log.Errorf("  app ID created but couldn't add iCloud containers: %v", containers)
	}

	if err := m.client.SyncBundleID(*bundleID, entitlements); err != nil {
		return nil, fmt.Errorf("failed to update bundle ID capabilities: %s", err)
	}

	m.bundleIDByBundleIDIdentifer[bundleIDIdentifier] = bundleID

	return bundleID, nil
}

func (m profileManager) ensureProfile(profileType appstoreconnect.ProfileType, bundleIDIdentifier string, entitlements Entitlements, certIDs, deviceIDs []string, minProfileDaysValid int) (*Profile, error) {
	fmt.Println()
	log.Infof("  Checking bundle id: %s", bundleIDIdentifier)
	log.Printf("  capabilities: %s", entitlements)

	// Search for Bitrise managed Profile
	name := profileName(profileType, bundleIDIdentifier)
	profile, err := m.client.FindProfile(name, profileType)
	if err != nil {
		return nil, fmt.Errorf("failed to find profile: %s", err)
	}

	if profile == nil {
		log.Warnf("  profile does not exist, generating...")
	} else {
		log.Printf("  Bitrise managed profile found: %s ID: %s UUID: %s Expiry: %s", profile.Attributes().Name, profile.ID(), profile.Attributes().UUID, time.Time(profile.Attributes().ExpirationDate))

		if profile.Attributes().ProfileState == appstoreconnect.Active {
			// Check if Bitrise managed Profile is sync with the project
			err := checkProfile(m.client, profile, entitlements, deviceIDs, certIDs, minProfileDaysValid)
			if err != nil {
				if mErr, ok := err.(NonmatchingProfileError); ok {
					log.Warnf("  the profile is not in sync with the project requirements (%s), regenerating ...", mErr.Reason)
				} else {
					return nil, fmt.Errorf("failed to check if profile is valid: %s", err)
				}
			} else { // Profile matches
				log.Donef("  profile is in sync with the project requirements")
				return &profile, nil
			}
		}

		if profile.Attributes().ProfileState == appstoreconnect.Invalid {
			// If the profile's bundle id gets modified, the profile turns in Invalid state.
			log.Warnf("  the profile state is invalid, regenerating ...")
		}

		if err := m.client.DeleteProfile(profile.ID()); err != nil {
			return nil, fmt.Errorf("failed to delete profile: %s", err)
		}
	}

	// Search for BundleID
	bundleID, err := m.ensureBundleID(bundleIDIdentifier, entitlements)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure application identifier for %s: %w", bundleIDIdentifier, err)
	}

	// Create Bitrise managed Profile
	fmt.Println()
	log.Infof("  Creating profile for bundle id: %s", bundleID.Attributes.Name)

	profile, err = m.client.CreateProfile(name, profileType, *bundleID, certIDs, deviceIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile: %s", err)
	}

	log.Donef("  profile created: %s", profile.Attributes().Name)
	return &profile, nil
}

func isAppClip(entitlements Entitlements) bool {
	for key := range entitlements {
		if key == appstoreconnect.ParentApplicationIdentifierEntitlementKey {
			return true
		}
	}
	return false
}

func hasSignInWithAppleEntitlement(entitlements Entitlements) bool {
	for key := range entitlements {
		if key == appstoreconnect.SignInWithAppleEntitlementKey {
			return true
		}
	}
	return false
}

func distributionTypeRequiresDeviceList(distrTypes []DistributionType) bool {
	for _, distrType := range distrTypes {
		if distrType == Development || distrType == AdHoc {
			return true
		}
	}
	return false
}

func createWildcardBundleID(bundleID string) (string, error) {
	idx := strings.LastIndex(bundleID, ".")
	if idx == -1 {
		return "", fmt.Errorf("invalid bundle id (%s): does not contain *", bundleID)
	}

	return bundleID[:idx] + ".*", nil
}

// profileName generates profile name with layout: Bitrise <platform> <distribution type> - (<bundle id>)
func profileName(profileType appstoreconnect.ProfileType, bundleID string) string {
	platform, ok := ProfileTypeToPlatform[profileType]
	if !ok {
		panic(fmt.Sprintf("unknown profile type: %s", profileType))
	}

	distribution, ok := ProfileTypeToDistribution[profileType]
	if !ok {
		panic(fmt.Sprintf("unknown profile type: %s", profileType))
	}

	prefix := ""
	if strings.HasSuffix(bundleID, ".*") {
		// `*` char is not allowed in Profile name.
		bundleID = strings.TrimSuffix(bundleID, ".*")
		prefix = "Wildcard "
	}

	return fmt.Sprintf("%sBitrise %s %s - (%s)", prefix, platform, distribution, bundleID)
}

func checkProfileEntitlements(client DevPortalClient, prof Profile, appEntitlements Entitlements) error {
	profileEnts, err := prof.Entitlements()
	if err != nil {
		return err
	}

	missingContainers, err := findMissingContainers(appEntitlements, profileEnts)
	if err != nil {
		return fmt.Errorf("failed to check missing containers: %s", err)
	}
	if len(missingContainers) > 0 {
		return NonmatchingProfileError{
			Reason: fmt.Sprintf("project uses containers that are missing from the provisioning profile: %v", missingContainers),
		}
	}

	bundleID, err := prof.BundleID()
	if err != nil {
		return err
	}

	return client.CheckBundleIDEntitlements(bundleID, appEntitlements)
}

// ParseRawProfileEntitlements ...
func ParseRawProfileEntitlements(profileContents []byte) (Entitlements, error) {
	pkcs, err := profileutil.ProvisioningProfileFromContent(profileContents)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pkcs7 from profile content: %s", err)
	}

	profile, err := profileutil.NewProvisioningProfileInfo(*pkcs)
	if err != nil {
		return nil, fmt.Errorf("failed to parse profile info from pkcs7 content: %s", err)
	}
	return Entitlements(profile.Entitlements), nil
}

func findMissingContainers(projectEnts, profileEnts Entitlements) ([]string, error) {
	projContainerIDs, err := serialized.Object(projectEnts).StringSlice("com.apple.developer.icloud-container-identifiers")
	if err != nil {
		if serialized.IsKeyNotFoundError(err) {
			return nil, nil // project has no container
		}
		return nil, err
	}

	// project has containers, so the profile should have at least the same

	profContainerIDs, err := serialized.Object(profileEnts).StringSlice("com.apple.developer.icloud-container-identifiers")
	if err != nil {
		if serialized.IsKeyNotFoundError(err) {
			return projContainerIDs, nil
		}
		return nil, err
	}

	// project and profile also has containers, check if profile contains the containers the project need

	var missing []string
	for _, projContainerID := range projContainerIDs {
		var found bool
		for _, profContainerID := range profContainerIDs {
			if projContainerID == profContainerID {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, projContainerID)
		}
	}

	return missing, nil
}

func checkProfileCertificates(profileCertificateIDs []string, certificateIDs []string) error {
	for _, id := range certificateIDs {
		if !sliceutil.IsStringInSlice(id, profileCertificateIDs) {
			return NonmatchingProfileError{
				Reason: fmt.Sprintf("certificate with ID (%s) not included in the profile", id),
			}
		}
	}
	return nil
}

func checkProfileDevices(profileDeviceIDs []string, deviceIDs []string) error {
	for _, id := range deviceIDs {
		if !sliceutil.IsStringInSlice(id, profileDeviceIDs) {
			return NonmatchingProfileError{
				Reason: fmt.Sprintf("device with ID (%s) not included in the profile", id),
			}
		}
	}

	return nil
}

func isProfileExpired(prof Profile, minProfileDaysValid int) bool {
	relativeExpiryTime := time.Now()
	if minProfileDaysValid > 0 {
		relativeExpiryTime = relativeExpiryTime.Add(time.Duration(minProfileDaysValid) * 24 * time.Hour)
	}
	return time.Time(prof.Attributes().ExpirationDate).Before(relativeExpiryTime)
}

func checkProfile(client DevPortalClient, prof Profile, entitlements Entitlements, deviceIDs, certificateIDs []string, minProfileDaysValid int) error {
	if isProfileExpired(prof, minProfileDaysValid) {
		return NonmatchingProfileError{
			Reason: fmt.Sprintf("profile expired, or will expire in less then %d day(s)", minProfileDaysValid),
		}
	}

	if err := checkProfileEntitlements(client, prof, entitlements); err != nil {
		return err
	}

	profileCertificateIDs, err := prof.CertificateIDs()
	if err != nil {
		return err
	}
	if err := checkProfileCertificates(profileCertificateIDs, certificateIDs); err != nil {
		return err
	}

	profileDeviceIDs, err := prof.DeviceIDs()
	if err != nil {
		return err
	}
	return checkProfileDevices(profileDeviceIDs, deviceIDs)
}
