// Code generated by mockery 2.9.4. DO NOT EDIT.

package autocodesign

import (
	appstoreconnect "github.com/bitrise-io/go-xcode/autocodesign/devportalclient/appstoreconnect"
	mock "github.com/stretchr/testify/mock"
)

// MockProfile is an autogenerated mock type for the Profile type
type MockProfile struct {
	mock.Mock
}

// Attributes provides a mock function with given fields:
func (_m *MockProfile) Attributes() appstoreconnect.ProfileAttributes {
	ret := _m.Called()

	var r0 appstoreconnect.ProfileAttributes
	if rf, ok := ret.Get(0).(func() appstoreconnect.ProfileAttributes); ok {
		r0 = rf()
	} else {
		r0, ok = ret.Get(0).(appstoreconnect.ProfileAttributes)
		if !ok {
		}
	}

	return r0
}

// BundleID provides a mock function with given fields:
func (_m *MockProfile) BundleID() (appstoreconnect.BundleID, error) {
	ret := _m.Called()

	var r0 appstoreconnect.BundleID
	if rf, ok := ret.Get(0).(func() appstoreconnect.BundleID); ok {
		r0 = rf()
	} else {
		r0, ok = ret.Get(0).(appstoreconnect.BundleID)
		if !ok {
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CertificateIDs provides a mock function with given fields:
func (_m *MockProfile) CertificateIDs() ([]string, error) {
	ret := _m.Called()

	var r0 []string
	if rf, ok := ret.Get(0).(func() []string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0, ok = ret.Get(0).([]string)
			if !ok {
			}
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeviceIDs provides a mock function with given fields:
func (_m *MockProfile) DeviceIDs() ([]string, error) {
	ret := _m.Called()

	var r0 []string
	if rf, ok := ret.Get(0).(func() []string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0, ok = ret.Get(0).([]string)
			if !ok {
			}
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Entitlements provides a mock function with given fields:
func (_m *MockProfile) Entitlements() (Entitlements, error) {
	ret := _m.Called()

	var r0 Entitlements
	if rf, ok := ret.Get(0).(func() Entitlements); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0, ok = ret.Get(0).(Entitlements)
			if !ok {
			}
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ID provides a mock function with given fields:
func (_m *MockProfile) ID() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0, ok = ret.Get(0).(string)
		if !ok {
		}
	}

	return r0
}
