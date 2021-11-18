package autocodesign

import (
	"errors"
	"fmt"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-xcode/autocodesign/devportalclient/appstoreconnect"
	"github.com/bitrise-io/go-xcode/devportalservice"
)

func EnsureTestDevices(deviceClient DevPortalClient, testDevices []devportalservice.TestDevice, platform Platform) ([]string, error) {
	var devPortalDeviceIDs []string

	log.Infof("Fetching Apple Developer Portal devices")
	// IOS device platform includes: APPLE_WATCH, IPAD, IPHONE, IPOD and APPLE_TV device classes.
	devPortalDevices, err := deviceClient.ListDevices("", appstoreconnect.IOSDevice)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch devices: %s", err)
	}

	log.Printf("%d devices are registered on the Apple Developer Portal", len(devPortalDevices))
	for _, devPortalDevice := range devPortalDevices {
		log.Debugf("- %s, %s, UDID (%s), ID (%s)", devPortalDevice.Attributes.Name, devPortalDevice.Attributes.DeviceClass, devPortalDevice.Attributes.UDID, devPortalDevice.ID)
	}

	if len(testDevices) != 0 {
		fmt.Println()
		log.Infof("Checking if %d Bitrise test device(s) are registered on Developer Portal", len(testDevices))
		for _, d := range testDevices {
			log.Debugf("- %s, %s, UDID (%s), added at %s", d.Title, d.DeviceType, d.DeviceID, d.UpdatedAt)
		}

		newDevPortalDevices, err := registerMissingTestDevices(deviceClient, testDevices, devPortalDevices)
		if err != nil {
			return nil, fmt.Errorf("failed to register Bitrise Test device on Apple Developer Portal: %s", err)
		}
		devPortalDevices = append(devPortalDevices, newDevPortalDevices...)
	}

	devPortalDevices = filterDevPortalDevices(devPortalDevices, platform)

	for _, devPortalDevice := range devPortalDevices {
		devPortalDeviceIDs = append(devPortalDeviceIDs, devPortalDevice.ID)
	}

	return devPortalDeviceIDs, nil
}

func registerMissingTestDevices(client DevPortalClient, testDevices []devportalservice.TestDevice, devPortalDevices []appstoreconnect.Device) ([]appstoreconnect.Device, error) {
	if client == nil {
		return []appstoreconnect.Device{}, fmt.Errorf("the App Store Connect API client not provided")
	}

	var newDevPortalDevices []appstoreconnect.Device

	for _, testDevice := range testDevices {
		log.Printf("checking if the device (%s) is registered", testDevice.DeviceID)

		devPortalDevice := findDevPortalDevice(testDevice, devPortalDevices)
		if devPortalDevice != nil {
			log.Printf("device already registered")
			continue
		}

		log.Printf("registering device")
		newDevPortalDevice, err := client.RegisterDevice(testDevice)
		if err != nil {
			var registrationError appstoreconnect.DeviceRegistrationError
			if errors.As(err, &registrationError) {
				log.Warnf("Failed to register device (can be caused by invalid UDID or trying to register a Mac device): %s", registrationError.Reason)
				return nil, nil
			}

			return nil, err
		}

		if newDevPortalDevice != nil {
			newDevPortalDevices = append(newDevPortalDevices, *newDevPortalDevice)
		}
	}

	return newDevPortalDevices, nil
}

func findDevPortalDevice(testDevice devportalservice.TestDevice, devPortalDevices []appstoreconnect.Device) *appstoreconnect.Device {
	for _, devPortalDevice := range devPortalDevices {
		if devportalservice.IsEqualUDID(devPortalDevice.Attributes.UDID, testDevice.DeviceID) {
			return &devPortalDevice
		}
	}
	return nil
}

func filterDevPortalDevices(devPortalDevices []appstoreconnect.Device, platform Platform) []appstoreconnect.Device {
	var filteredDevices []appstoreconnect.Device

	for _, devPortalDevice := range devPortalDevices {
		deviceClass := devPortalDevice.Attributes.DeviceClass

		switch platform {
		case IOS:
			isIosOrWatchosDevice := deviceClass == appstoreconnect.AppleWatch ||
				deviceClass == appstoreconnect.Ipad ||
				deviceClass == appstoreconnect.Iphone ||
				deviceClass == appstoreconnect.Ipod

			if isIosOrWatchosDevice {
				filteredDevices = append(filteredDevices, devPortalDevice)
			}
		case TVOS:
			if deviceClass == appstoreconnect.AppleTV {
				filteredDevices = append(filteredDevices, devPortalDevice)
			}
		}
	}

	return filteredDevices
}
