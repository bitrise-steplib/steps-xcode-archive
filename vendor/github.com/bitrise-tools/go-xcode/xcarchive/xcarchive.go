package xcarchive

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-tools/go-xcode/plistutil"
	"github.com/bitrise-tools/go-xcode/profileutil"
	"github.com/bitrise-tools/go-xcode/utility"
)

// Application ...
type Application struct {
	Path                string
	InfoPlist           plistutil.PlistData
	Entitlements        plistutil.PlistData
	ProvisioningProfile profileutil.ProvisioningProfileInfoModel
	Plugins             []Application
}

// BundleIdentifier ...
func (app Application) BundleIdentifier() string {
	bundleID, _ := app.InfoPlist.GetString("CFBundleIdentifier")
	return bundleID
}

// NewApplication ...
func NewApplication(appOrAppexPth string) (Application, error) {
	application := Application{
		Path: appOrAppexPth,
	}

	{
		infoPlistPth := filepath.Join(appOrAppexPth, "Info.plist")
		exist, err := pathutil.IsPathExists(infoPlistPth)
		if err != nil {
			return Application{}, err
		} else if !exist {
			return Application{}, fmt.Errorf("Info.plist does not exist at: %s", infoPlistPth)
		}
		infoPlist, err := plistutil.NewPlistDataFromFile(infoPlistPth)
		if err != nil {
			return Application{}, err
		}
		application.InfoPlist = infoPlist
	}

	{
		provisioningProfilePth := filepath.Join(appOrAppexPth, "embedded.mobileprovision")
		exist, err := pathutil.IsPathExists(provisioningProfilePth)
		if err != nil {
			return Application{}, err
		} else if !exist {
			return Application{}, fmt.Errorf("embedded.mobileprovision does not exist at: %s", provisioningProfilePth)
		}
		profile, err := profileutil.NewProvisioningProfileInfoFromFile(provisioningProfilePth)
		if err != nil {
			return Application{}, err
		}
		application.ProvisioningProfile = profile
	}

	{
		entitlementsPth := filepath.Join(appOrAppexPth, "archived-expanded-entitlements.xcent")
		exist, err := pathutil.IsPathExists(entitlementsPth)
		if err != nil {
			return Application{}, err
		} else if exist {
			entitlements, err := plistutil.NewPlistDataFromFile(entitlementsPth)
			if err != nil {
				return Application{}, err
			}

			application.Entitlements = entitlements
		}
	}

	return application, nil
}

// Applications ...
type Applications struct {
	MainApplication  Application
	WatchApplication *Application
}

// NewApplications ...
func NewApplications(applicationsDir string) (Applications, error) {
	mainApplication := Application{}
	mainApplicationPth := ""
	{
		pattern := filepath.Join(applicationsDir, "*.app")
		pths, err := filepath.Glob(pattern)
		if err != nil {
			return Applications{}, err
		}

		if len(pths) == 0 {
			return Applications{}, fmt.Errorf("Failed to find main application using pattern: %s", pattern)
		} else if len(pths) > 1 {
			log.Warnf("Multiple main applications found")
			for _, pth := range pths {
				log.Warnf("- %s", pth)
			}

			mainApplicationPth = pths[0]
			log.Warnf("Using first: %s", mainApplicationPth)
		} else {
			mainApplicationPth = pths[0]
		}

		mainApplication, err = NewApplication(mainApplicationPth)
		if err != nil {
			return Applications{}, err
		}
	}

	plugins := []Application{}
	{
		pattern := filepath.Join(mainApplicationPth, "PlugIns/*.appex")
		pths, err := filepath.Glob(pattern)
		if err != nil {
			return Applications{}, err
		}
		for _, pth := range pths {
			plugin, err := NewApplication(pth)
			if err != nil {
				return Applications{}, err
			}

			plugins = append(plugins, plugin)
		}
		mainApplication.Plugins = plugins
	}

	var watchApplicationPtr *Application
	watchApplicationPth := ""
	{
		pattern := filepath.Join(mainApplicationPth, "Watch/*.app")
		pths, err := filepath.Glob(pattern)
		if err != nil {
			return Applications{}, err
		}

		if len(pths) > 1 {
			log.Warnf("Multiple watch applications found")
			for _, pth := range pths {
				log.Warnf("- %s", pth)
			}

			watchApplicationPth = pths[0]
			log.Warnf("Using first: %s", watchApplicationPth)
		} else if len(pths) == 1 {
			watchApplicationPth = pths[0]
		}

		if watchApplicationPth != "" {
			watchApplication, err := NewApplication(watchApplicationPth)
			if err != nil {
				return Applications{}, err
			}

			watchApplicationPtr = &watchApplication
		}
	}

	watchPlugins := []Application{}
	{
		if watchApplicationPth != "" {
			pattern := filepath.Join(watchApplicationPth, "PlugIns/*.appex")
			pths, err := filepath.Glob(pattern)
			if err != nil {
				return Applications{}, err
			}
			for _, pth := range pths {
				plugin, err := NewApplication(pth)
				if err != nil {
					return Applications{}, err
				}

				watchPlugins = append(watchPlugins, plugin)
			}
			(*watchApplicationPtr).Plugins = watchPlugins
		}
	}

	return Applications{
		MainApplication:  mainApplication,
		WatchApplication: watchApplicationPtr,
	}, nil
}

// XCArchive ...
type XCArchive struct {
	Path         string
	Applications Applications
	InfoPlist    plistutil.PlistData
}

// IsXcodeManaged ...
func (archive XCArchive) IsXcodeManaged() bool {
	return archive.Applications.MainApplication.ProvisioningProfile.IsXcodeManaged()
}

// SigningIdentity ...
func (archive XCArchive) SigningIdentity() string {
	properties, found := archive.InfoPlist.GetMapStringInterface("ApplicationProperties")
	if found {
		identity, _ := properties.GetString("SigningIdentity")
		return identity
	}
	return ""
}

// BundleIDEntitlementsMap ...
func (archive XCArchive) BundleIDEntitlementsMap() map[string]plistutil.PlistData {
	bundleIDEntitlementsMap := map[string]plistutil.PlistData{}

	bundleID := archive.Applications.MainApplication.BundleIdentifier()
	bundleIDEntitlementsMap[bundleID] = archive.Applications.MainApplication.Entitlements

	for _, plugin := range archive.Applications.MainApplication.Plugins {
		bundleID := plugin.BundleIdentifier()
		bundleIDEntitlementsMap[bundleID] = plugin.Entitlements
	}

	if archive.Applications.WatchApplication != nil {
		watchApplication := *archive.Applications.WatchApplication

		bundleID := watchApplication.BundleIdentifier()
		bundleIDEntitlementsMap[bundleID] = watchApplication.Entitlements

		for _, plugin := range watchApplication.Plugins {
			bundleID := plugin.BundleIdentifier()
			bundleIDEntitlementsMap[bundleID] = plugin.Entitlements
		}
	}

	return bundleIDEntitlementsMap
}

// BundleIDProfileInfoMap ...
func (archive XCArchive) BundleIDProfileInfoMap() map[string]profileutil.ProvisioningProfileInfoModel {
	bundleIDProfileMap := map[string]profileutil.ProvisioningProfileInfoModel{}

	bundleID := archive.Applications.MainApplication.BundleIdentifier()
	bundleIDProfileMap[bundleID] = archive.Applications.MainApplication.ProvisioningProfile

	for _, plugin := range archive.Applications.MainApplication.Plugins {
		bundleID := plugin.BundleIdentifier()
		bundleIDProfileMap[bundleID] = plugin.ProvisioningProfile
	}

	if archive.Applications.WatchApplication != nil {
		watchApplication := *archive.Applications.WatchApplication

		bundleID := watchApplication.BundleIdentifier()
		bundleIDProfileMap[bundleID] = watchApplication.ProvisioningProfile

		for _, plugin := range watchApplication.Plugins {
			bundleID := plugin.BundleIdentifier()
			bundleIDProfileMap[bundleID] = plugin.ProvisioningProfile
		}
	}

	return bundleIDProfileMap
}

// FindDSYMs ...
func (archive XCArchive) FindDSYMs() (string, []string, error) {
	dsymsDirPth := filepath.Join(archive.Path, "dSYMs")
	dsyms, err := utility.ListEntries(dsymsDirPth, utility.ExtensionFilter(".dsym", true))
	if err != nil {
		return "", []string{}, err
	}

	appDSYM := ""
	frameworkDSYMs := []string{}
	for _, dsym := range dsyms {
		if strings.HasSuffix(dsym, ".app.dSYM") {
			appDSYM = dsym
		} else {
			frameworkDSYMs = append(frameworkDSYMs, dsym)
		}
	}
	if appDSYM == "" && len(frameworkDSYMs) == 0 {
		return "", []string{}, fmt.Errorf("no dsym found")
	}

	return appDSYM, frameworkDSYMs, nil
}

// NewXCArchive ...
func NewXCArchive(xcarchivePth string) (XCArchive, error) {
	applications := Applications{}
	{
		applicationsDir := filepath.Join(xcarchivePth, "Products/Applications")
		exist, err := pathutil.IsDirExists(applicationsDir)
		if err != nil {
			return XCArchive{}, err
		} else if !exist {
			return XCArchive{}, fmt.Errorf("Applications dir does not exist at: %s", applicationsDir)
		}

		applications, err = NewApplications(applicationsDir)
		if err != nil {
			return XCArchive{}, err
		}

	}

	infoPlist := plistutil.PlistData{}
	{
		infoPlistPth := filepath.Join(xcarchivePth, "Info.plist")
		exist, err := pathutil.IsPathExists(infoPlistPth)
		if err != nil {
			return XCArchive{}, err
		} else if !exist {
			return XCArchive{}, fmt.Errorf("Info.plist does not exist at: %s", infoPlistPth)
		}
		infoPlist, err = plistutil.NewPlistDataFromFile(infoPlistPth)
		if err != nil {
			return XCArchive{}, err
		}
	}

	return XCArchive{
		Path:         xcarchivePth,
		Applications: applications,
		InfoPlist:    infoPlist,
	}, nil
}
