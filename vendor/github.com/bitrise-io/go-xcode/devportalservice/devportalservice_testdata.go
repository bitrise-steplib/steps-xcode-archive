package devportalservice

import (
	"time"
)

func toTime(str string) *time.Time {
	if str == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		panic(err)
	}
	return &t
}

const testDevicesResponseBody = `{
   "test_devices":[
      {
         "id":24,
         "user_id":4,
         "device_identifier":"asdf12345ad9b298cb9a9f28555c49573d8bc322",
         "title":"iPhone 6",
         "created_at":"2015-03-13T16:16:13.665Z",
         "updated_at":"2015-03-13T16:16:13.665Z",
         "device_type":"ios"
      },
      {
         "id":28,
         "user_id":4,
         "device_identifier":"asdf12341e73b76df6e99d0d713133c3e078418f",
         "title":"iPad mini 2 (Wi-Fi)",
         "created_at":"2015-03-19T13:25:43.487Z",
         "updated_at":"2015-03-19T13:25:43.487Z",
         "device_type":"ios"
	  }
	]
}
`

var testDevices = []TestDevice{
	{
		ID:         24,
		UserID:     4,
		DeviceID:   "asdf12345ad9b298cb9a9f28555c49573d8bc322",
		Title:      "iPhone 6",
		CreatedAt:  *toTime("2015-03-13T16:16:13.665Z"),
		UpdatedAt:  *toTime("2015-03-13T16:16:13.665Z"),
		DeviceType: "ios",
	},
	{
		ID:         28,
		UserID:     4,
		DeviceID:   "asdf12341e73b76df6e99d0d713133c3e078418f",
		Title:      "iPad mini 2 (Wi-Fi)",
		CreatedAt:  *toTime("2015-03-19T13:25:43.487Z"),
		UpdatedAt:  *toTime("2015-03-19T13:25:43.487Z"),
		DeviceType: "ios",
	},
}

var testConnectionOnlyDevices = AppleDeveloperConnection{
	AppleIDConnection: nil,
	APIKeyConnection:  nil,
	TestDevices:       testDevices,
}

const testAppleIDConnectionResponseBody = `{
    "apple_id": "example@example.io",
    "password": "highSecurityPassword",
    "connection_expiry_date": "2019-04-06T12:04:59.000Z",
    "session_cookies": {
        "https://idmsa.apple.com": [
            {
                "name": "DES58b0eba556d80ed2b98707e15ffafd344",
                "path": "/",
                "value": "HSARMTKNSRVTWFlaFrGQTmfmFBwJuiX/aaaaaaaaa+A7FbJa4V8MmWijnJknnX06ME0KrI9V8vFg==SRVT",
                "domain": "idmsa.apple.com",
                "secure": true,
                "expires": "2019-04-06T12:04:59Z",
                "max_age": 2592000,
                "httponly": true
            },
            {
                "name": "myacinfo",
                "path": "/",
                "value": "DAWTKNV26a0a6db3ae43acd203d0d03e8bc45000cd4bdc668e90953f22ca3b36eaab0e18634660a10cf28cc65d8ddf633c017de09477dfb18c8a3d6961f96cbbf064be616e80cee62d3d7f39a485bf826377c5b5dbbfc4a97dcdb462052db73a3a1d9b4a325d5bdd496190b3088878cecce17e4d6db9230e0575cfbe7a8754d1de0c937080ef84569b6e4a75237c2ec01cf07db060a11d92e7220707dd00a2a565ee9e06074d8efa6a1b7f83db3e1b2acdafb5fc0708443e77e6d71e168ae2a83b848122264b2da5cadfd9e451f9fe3f6eebc71904d4bc36acc528cc2a844d4f2eb527649a69523756ec9955457f704c28a3b6b9f97d6df900bd60044d5bc50408260f096954f03c53c16ac40a796dc439b859f882a50390b1c7517a9f4479fb1ce9ba2db241d6b8f2eb127c46ef96e0ccccccccc",
                "domain": "apple.com",
                "secure": true,
                "httponly": true
            }
        ]
    }
}`

const testFastlaneSession = `---
- !ruby/object:HTTP::Cookie
  name: DES58b0eba556d80ed2b98707e15ffafd344
  value: HSARMTKNSRVTWFlaFrGQTmfmFBwJuiX/aaaaaaaaa+A7FbJa4V8MmWijnJknnX06ME0KrI9V8vFg==SRVT
  domain: idmsa.apple.com
  for_domain: true
  path: "/"

- !ruby/object:HTTP::Cookie
  name: myacinfo
  value: DAWTKNV26a0a6db3ae43acd203d0d03e8bc45000cd4bdc668e90953f22ca3b36eaab0e18634660a10cf28cc65d8ddf633c017de09477dfb18c8a3d6961f96cbbf064be616e80cee62d3d7f39a485bf826377c5b5dbbfc4a97dcdb462052db73a3a1d9b4a325d5bdd496190b3088878cecce17e4d6db9230e0575cfbe7a8754d1de0c937080ef84569b6e4a75237c2ec01cf07db060a11d92e7220707dd00a2a565ee9e06074d8efa6a1b7f83db3e1b2acdafb5fc0708443e77e6d71e168ae2a83b848122264b2da5cadfd9e451f9fe3f6eebc71904d4bc36acc528cc2a844d4f2eb527649a69523756ec9955457f704c28a3b6b9f97d6df900bd60044d5bc50408260f096954f03c53c16ac40a796dc439b859f882a50390b1c7517a9f4479fb1ce9ba2db241d6b8f2eb127c46ef96e0ccccccccc
  domain: apple.com
  for_domain: true
  path: "/"

`

var testAppleIDConnection = AppleIDConnection{
	AppleID:           "example@example.io",
	Password:          "highSecurityPassword",
	SessionExpiryDate: toTime("2019-04-06T12:04:59.000Z"),
	SessionCookies: map[string][]cookie{
		"https://idmsa.apple.com": {
			{
				Name:     "DES58b0eba556d80ed2b98707e15ffafd344",
				Path:     "/",
				Value:    "HSARMTKNSRVTWFlaFrGQTmfmFBwJuiX/aaaaaaaaa+A7FbJa4V8MmWijnJknnX06ME0KrI9V8vFg==SRVT",
				Domain:   "idmsa.apple.com",
				Secure:   true,
				Expires:  "2019-04-06T12:04:59Z",
				MaxAge:   2592000,
				Httponly: true,
			},
			{
				Name:     "myacinfo",
				Path:     "/",
				Value:    "DAWTKNV26a0a6db3ae43acd203d0d03e8bc45000cd4bdc668e90953f22ca3b36eaab0e18634660a10cf28cc65d8ddf633c017de09477dfb18c8a3d6961f96cbbf064be616e80cee62d3d7f39a485bf826377c5b5dbbfc4a97dcdb462052db73a3a1d9b4a325d5bdd496190b3088878cecce17e4d6db9230e0575cfbe7a8754d1de0c937080ef84569b6e4a75237c2ec01cf07db060a11d92e7220707dd00a2a565ee9e06074d8efa6a1b7f83db3e1b2acdafb5fc0708443e77e6d71e168ae2a83b848122264b2da5cadfd9e451f9fe3f6eebc71904d4bc36acc528cc2a844d4f2eb527649a69523756ec9955457f704c28a3b6b9f97d6df900bd60044d5bc50408260f096954f03c53c16ac40a796dc439b859f882a50390b1c7517a9f4479fb1ce9ba2db241d6b8f2eb127c46ef96e0ccccccccc",
				Domain:   "apple.com",
				Secure:   true,
				Httponly: true,
			},
		},
	},
}

var testConnectionWithAppleIDConnection = AppleDeveloperConnection{
	AppleIDConnection: &testAppleIDConnection,
	APIKeyConnection:  nil,
	TestDevices:       nil,
}

const testAPIKeyConnectionResponseBody = `{
    "key_id": "ASDF4H9LNQ",
    "issuer_id": "asdf1234-7325-47e3-e053-5b8c7c11a4d1",
    "private_key": "-----BEGIN PRIVATE KEY-----\nASdf1234MBMGByqGSM49AgEGCCqGSM49AwEHBHkwdwIBAQQg9O4G/HVLgSqc2i7x\nasDF12346UNzKCEwOfQ1ixC0G9agCgYIKoZIzj0DAQehRANCAARcJQItGFcefLRc\naSDf1234ka9BMpRjjr3NWyCWl817HCdXXckuc22RjnKxRnYMBBDv8zPDX0k9TbST\nacgZ04Gg\n-----END PRIVATE KEY-----"
}`

var testAPIKeyConnection = APIKeyConnection{
	KeyID:      "ASDF4H9LNQ",
	IssuerID:   "asdf1234-7325-47e3-e053-5b8c7c11a4d1",
	PrivateKey: "-----BEGIN PRIVATE KEY-----\nASdf1234MBMGByqGSM49AgEGCCqGSM49AwEHBHkwdwIBAQQg9O4G/HVLgSqc2i7x\nasDF12346UNzKCEwOfQ1ixC0G9agCgYIKoZIzj0DAQehRANCAARcJQItGFcefLRc\naSDf1234ka9BMpRjjr3NWyCWl817HCdXXckuc22RjnKxRnYMBBDv8zPDX0k9TbST\nacgZ04Gg\n-----END PRIVATE KEY-----",
}

var testConnectionWithAPIKeyConnection = AppleDeveloperConnection{
	AppleIDConnection: nil,
	APIKeyConnection:  &testAPIKeyConnection,
	TestDevices:       nil,
}

const testAppleIDAndAPIKeyConnectionResponseBody = `{
    "apple_id": "example@example.io",
    "password": "highSecurityPassword",
    "connection_expiry_date": "2019-04-06T12:04:59.000Z",
    "session_cookies": {
        "https://idmsa.apple.com": [
            {
                "name": "DES58b0eba556d80ed2b98707e15ffafd344",
                "path": "/",
                "value": "HSARMTKNSRVTWFlaFrGQTmfmFBwJuiX/aaaaaaaaa+A7FbJa4V8MmWijnJknnX06ME0KrI9V8vFg==SRVT",
                "domain": "idmsa.apple.com",
                "secure": true,
                "expires": "2019-04-06T12:04:59Z",
                "max_age": 2592000,
                "httponly": true
            },
            {
                "name": "myacinfo",
                "path": "/",
                "value": "DAWTKNV26a0a6db3ae43acd203d0d03e8bc45000cd4bdc668e90953f22ca3b36eaab0e18634660a10cf28cc65d8ddf633c017de09477dfb18c8a3d6961f96cbbf064be616e80cee62d3d7f39a485bf826377c5b5dbbfc4a97dcdb462052db73a3a1d9b4a325d5bdd496190b3088878cecce17e4d6db9230e0575cfbe7a8754d1de0c937080ef84569b6e4a75237c2ec01cf07db060a11d92e7220707dd00a2a565ee9e06074d8efa6a1b7f83db3e1b2acdafb5fc0708443e77e6d71e168ae2a83b848122264b2da5cadfd9e451f9fe3f6eebc71904d4bc36acc528cc2a844d4f2eb527649a69523756ec9955457f704c28a3b6b9f97d6df900bd60044d5bc50408260f096954f03c53c16ac40a796dc439b859f882a50390b1c7517a9f4479fb1ce9ba2db241d6b8f2eb127c46ef96e0ccccccccc",
                "domain": "apple.com",
                "secure": true,
                "httponly": true
            }
        ]
    },
    "key_id": "ASDF4H9LNQ",
    "issuer_id": "asdf1234-7325-47e3-e053-5b8c7c11a4d1",
    "private_key": "-----BEGIN PRIVATE KEY-----\nASdf1234MBMGByqGSM49AgEGCCqGSM49AwEHBHkwdwIBAQQg9O4G/HVLgSqc2i7x\nasDF12346UNzKCEwOfQ1ixC0G9agCgYIKoZIzj0DAQehRANCAARcJQItGFcefLRc\naSDf1234ka9BMpRjjr3NWyCWl817HCdXXckuc22RjnKxRnYMBBDv8zPDX0k9TbST\nacgZ04Gg\n-----END PRIVATE KEY-----",
    "test_devices":[
        {
           "id":24,
           "user_id":4,
           "device_identifier":"asdf12345ad9b298cb9a9f28555c49573d8bc322",
           "title":"iPhone 6",
           "created_at":"2015-03-13T16:16:13.665Z",
           "updated_at":"2015-03-13T16:16:13.665Z",
           "device_type":"ios"
        },
        {
            "id":28,
            "user_id":4,
            "device_identifier":"asdf12341e73b76df6e99d0d713133c3e078418f",
            "title":"iPad mini 2 (Wi-Fi)",
            "created_at":"2015-03-19T13:25:43.487Z",
            "updated_at":"2015-03-19T13:25:43.487Z",
            "device_type":"ios"
        }
    ]
}
`

var testConnectionWithAppleIDAndAPIKeyConnection = AppleDeveloperConnection{
	AppleIDConnection: &testAppleIDConnection,
	APIKeyConnection:  &testAPIKeyConnection,
	TestDevices:       testDevices,
}
