package appstoreconnectclient

import (
	"fmt"
	"net/http"

	"github.com/bitrise-io/go-xcode/autocodesign/devportalclient/appstoreconnect"
	"github.com/bitrise-io/go-xcode/devportalservice"
)

// DeviceClient ...
type DeviceClient struct {
	client *appstoreconnect.Client
}

// NewDeviceClient ...
func NewDeviceClient(client *appstoreconnect.Client) *DeviceClient {
	return &DeviceClient{client: client}
}

// ListDevices returns the registered devices on the Apple Developer portal
func (d *DeviceClient) ListDevices(udid string, platform appstoreconnect.DevicePlatform) ([]appstoreconnect.Device, error) {
	var nextPageURL string
	var devices []appstoreconnect.Device
	for {
		response, err := d.client.Provisioning.ListDevices(&appstoreconnect.ListDevicesOptions{
			PagingOptions: appstoreconnect.PagingOptions{
				Limit: 20,
				Next:  nextPageURL,
			},
			FilterUDID:     udid,
			FilterPlatform: platform,
			FilterStatus:   appstoreconnect.Enabled,
		})
		if err != nil {
			return nil, err
		}

		devices = append(devices, response.Data...)

		nextPageURL = response.Links.Next
		if nextPageURL == "" {
			return devices, nil
		}
	}
}

// RegisterDevice ...
func (d *DeviceClient) RegisterDevice(testDevice devportalservice.TestDevice) (*appstoreconnect.Device, error) {
	// The API seems to recognize existing devices even with different casing and '-' separator removed.
	// The Developer Portal UI does not let adding devices with unexpected casing or separators removed.
	// Did not fully validate the ability to add devices with changed casing (or '-' removed) via the API, so passing the UDID through unchanged.
	req := appstoreconnect.DeviceCreateRequest{
		Data: appstoreconnect.DeviceCreateRequestData{
			Attributes: appstoreconnect.DeviceCreateRequestDataAttributes{
				Name:     "Bitrise test device",
				Platform: appstoreconnect.IOS,
				UDID:     testDevice.DeviceID,
			},
			Type: "devices",
		},
	}

	registeredDevice, err := d.client.Provisioning.RegisterNewDevice(req)
	if err != nil {
		rerr, ok := err.(*appstoreconnect.ErrorResponse)
		if ok && rerr.Response != nil && rerr.Response.StatusCode == http.StatusConflict {
			return nil, appstoreconnect.DeviceRegistrationError{
				Reason: fmt.Sprintf("%v", err),
			}
		}

		return nil, err
	}

	return &registeredDevice.Data, nil
}
