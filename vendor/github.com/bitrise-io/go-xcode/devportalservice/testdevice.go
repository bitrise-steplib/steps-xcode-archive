package devportalservice

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/pathutil"
)

// TestDevice ...
type TestDevice struct {
	ID     int `json:"id"`
	UserID int `json:"user_id"`
	// DeviceID is the Apple device UDID
	DeviceID   string    `json:"device_identifier"`
	Title      string    `json:"title"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	DeviceType string    `json:"device_type"`
}

// ParseTestDevicesFromFile ...
func ParseTestDevicesFromFile(path string, currentTime time.Time) ([]TestDevice, error) {
	absPath, err := pathutil.AbsPath(path)
	if err != nil {
		return []TestDevice{}, err
	}

	bytes, err := os.ReadFile(absPath)
	if err != nil {
		return []TestDevice{}, err
	}

	fileContent := strings.TrimSpace(string(bytes))
	identifiers := strings.Split(fileContent, ",")

	var testDevices []TestDevice
	for i, identifier := range identifiers {
		testDevices = append(testDevices, TestDevice{
			DeviceID:   identifier,
			Title:      fmt.Sprintf("Device %d", i+1),
			CreatedAt:  currentTime,
			UpdatedAt:  currentTime,
			DeviceType: "unknown",
		})
	}

	return testDevices, nil
}
