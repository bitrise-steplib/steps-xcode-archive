package spaceship

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-xcode/devportalservice"
	"github.com/bitrise-io/go-xcode/v2/autocodesign/devportalclient/appstoreconnect"
)

// DeviceClient ...
type DeviceClient struct {
	client *Client
}

// NewDeviceClient ...
func NewDeviceClient(client *Client) *DeviceClient {
	return &DeviceClient{client: client}
}

// DeviceInfo ...
type DeviceInfo struct {
	ID       string                           `json:"id"`
	UDID     string                           `json:"udid"`
	Name     string                           `json:"name"`
	Model    string                           `json:"model"`
	Status   appstoreconnect.Status           `json:"status"`
	Platform appstoreconnect.BundleIDPlatform `json:"platform"`
	Class    appstoreconnect.DeviceClass      `json:"class"`
}

func newDevice(d DeviceInfo) appstoreconnect.Device {
	return appstoreconnect.Device{
		ID:   d.ID,
		Type: d.Model,
		Attributes: appstoreconnect.DeviceAttributes{
			DeviceClass: d.Class,
			Model:       d.Model,
			Name:        d.Name,
			Platform:    d.Platform,
			Status:      d.Status,
			UDID:        d.UDID,
		},
	}
}

// ListDevices ...
func (d *DeviceClient) ListDevices(udid string, platform appstoreconnect.DevicePlatform) ([]appstoreconnect.Device, error) {
	log.Debugf("Fetching devices")

	output, err := d.client.runSpaceshipCommand("list_devices")
	if err != nil {
		return nil, err
	}

	var deviceResponse struct {
		Data []DeviceInfo `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &deviceResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	var devices []appstoreconnect.Device
	for _, d := range deviceResponse.Data {
		devices = append(devices, newDevice(d))
	}

	var filteredDevices []appstoreconnect.Device
	for _, d := range devices {
		if udid != "" && d.Attributes.UDID != udid {
			log.Debugf("Device filtered out, UDID required: %s actual: %s", udid, d.Attributes.UDID)
			continue
		}
		if d.Attributes.Platform != appstoreconnect.BundleIDPlatform(platform) {
			log.Debugf("Device filtered out, platform required: %s actual: %s", appstoreconnect.BundleIDPlatform(platform), d.Attributes.Platform)
			continue
		}

		filteredDevices = append(filteredDevices, d)
	}

	return filteredDevices, nil
}

// RegisterDevice ...
func (d *DeviceClient) RegisterDevice(testDevice devportalservice.TestDevice) (*appstoreconnect.Device, error) {
	log.Debugf("Registering device")

	output, err := d.client.runSpaceshipCommand("register_device",
		"--udid", testDevice.DeviceID,
		"--name", testDevice.Title,
	)
	if err != nil {
		return nil, err
	}

	var deviceResponse struct {
		Data struct {
			Device   *DeviceInfo `json:"device"`
			Warnings []string    `json:"warnings"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(output), &deviceResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if deviceResponse.Data.Device == nil {
		if len(deviceResponse.Data.Warnings) != 0 {
			return nil, appstoreconnect.DeviceRegistrationError{
				Reason: strings.Join(deviceResponse.Data.Warnings, "\n"),
			}
		}

		return nil, errors.New("unexpected device registration failure")
	}

	device := newDevice(*deviceResponse.Data.Device)

	return &device, nil
}
