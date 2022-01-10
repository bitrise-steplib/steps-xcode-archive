package appstoreconnect

import (
	"net/http"
	"strings"
)

// DevicesEndpoint ...
const DevicesEndpoint = "devices"

// ListDevicesOptions ...
type ListDevicesOptions struct {
	PagingOptions
	FilterUDID     string         `url:"filter[udid],omitempty"`
	FilterPlatform DevicePlatform `url:"filter[platform],omitempty"`
	FilterStatus   Status         `url:"filter[status],omitempty"`
}

// DeviceClass ...
type DeviceClass string

// DeviceClasses ...
const (
	AppleWatch DeviceClass = "APPLE_WATCH"
	Ipad       DeviceClass = "IPAD"
	Iphone     DeviceClass = "IPHONE"
	Ipod       DeviceClass = "IPOD"
	AppleTV    DeviceClass = "APPLE_TV"
	Mac        DeviceClass = "MAC"
)

// DevicePlatform ...
type DevicePlatform string

// DevicePlatforms ...
const (
	IOSDevice   DevicePlatform = "IOS"
	MacOSDevice DevicePlatform = "MAC_OS"
)

// Status ...
type Status string

// Statuses ...
const (
	Enabled  Status = "ENABLED"
	Disabled Status = "DISABLED"
)

// DeviceAttributes ...
type DeviceAttributes struct {
	DeviceClass DeviceClass      `json:"deviceClass"`
	Model       string           `json:"model"`
	Name        string           `json:"name"`
	Platform    BundleIDPlatform `json:"platform"`
	Status      Status           `json:"status"`
	UDID        string           `json:"udid"`
	AddedDate   string           `json:"addedDate"`
}

// Device ...
type Device struct {
	Type       string           `json:"type"`
	ID         string           `json:"id"`
	Attributes DeviceAttributes `json:"attributes"`
}

// DevicesResponse ...
type DevicesResponse struct {
	Data  []Device           `json:"data"`
	Links PagedDocumentLinks `json:"links,omitempty"`
}

// DeviceResponse ...
type DeviceResponse struct {
	Data  Device             `json:"data"`
	Links PagedDocumentLinks `json:"links,omitempty"`
}

// ListDevices ...
func (s ProvisioningService) ListDevices(opt *ListDevicesOptions) (*DevicesResponse, error) {
	if err := opt.UpdateCursor(); err != nil {
		return nil, err
	}

	u, err := addOptions(DevicesEndpoint, opt)
	if err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	r := &DevicesResponse{}
	if _, err := s.client.Do(req, r); err != nil {
		return nil, err
	}

	return r, nil
}

// DeviceCreateRequestDataAttributes ...
type DeviceCreateRequestDataAttributes struct {
	Name     string           `json:"name"`
	Platform BundleIDPlatform `json:"platform"`
	UDID     string           `json:"udid"`
}

// DeviceCreateRequestData ...
type DeviceCreateRequestData struct {
	Attributes DeviceCreateRequestDataAttributes `json:"attributes"`
	Type       string                            `json:"type"`
}

// DeviceCreateRequest ...
type DeviceCreateRequest struct {
	Data DeviceCreateRequestData `json:"data"`
}

// RegisterNewDevice ...
func (s ProvisioningService) RegisterNewDevice(body DeviceCreateRequest) (*DeviceResponse, error) {
	req, err := s.client.NewRequest(http.MethodPost, DevicesEndpoint, body)
	if err != nil {
		return nil, err
	}

	r := &DeviceResponse{}
	if _, err := s.client.Do(req, r); err != nil {
		return nil, err
	}

	return r, nil
}

// Devices ...
func (s ProvisioningService) Devices(relationshipLink string, opt *PagingOptions) (*DevicesResponse, error) {
	if err := opt.UpdateCursor(); err != nil {
		return nil, err
	}

	u, err := addOptions(relationshipLink, opt)
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimPrefix(u, baseURL+apiVersion)
	req, err := s.client.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	r := &DevicesResponse{}
	if _, err := s.client.Do(req, r); err != nil {
		return nil, err
	}

	return r, nil
}
