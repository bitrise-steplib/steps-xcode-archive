require 'spaceship'
require_relative 'log'

def list_devices
  devices = Spaceship::Portal.device.all(mac: false, include_disabled: false) || []

  devices_info = []
  devices.each do |d|
    devices_info.append(
      {
        id: d.id,
        udid: d.udid,
        name: d.name,
        model: d.model,
        status: map_device_status_to_api_status(d.status),
        platform: map_device_platform_to_api_platform(d.platform),
        class: map_device_class_to_api_class(d.device_type)
      }
    )
  end

  devices_info
end

def register_device(udid, name)
  device = Spaceship::Portal.device.create!(name: name, udid: udid)
  {
    device: {
        id: device.id,
        udid: device.udid,
        name: device.name,
        model: device.model,
        status: map_device_status_to_api_status(device.status),
        platform: map_device_platform_to_api_platform(device.platform),
        class: map_device_class_to_api_class(device.device_type)
    }
  }
rescue Spaceship::UnexpectedResponse, Spaceship::BasicPreferredInfoError => e
  message = preferred_error_message(e)
  { warnings: ["Failed to register device with name: #{name} udid: #{udid} error: #{message}"] }
rescue
  { warnings: ["Failed to register device with name: #{name} udid: #{udid}"] }
end

def map_device_platform_to_api_platform(platform)
  case platform
  when 'ios'
    'IOS'
  when 'mac'
    'MAC_OS'
  else
    raise "unknown device platform #{platform}"
  end
end

def map_device_status_to_api_status(status)
  case status
  when 'c'
    'ENABLED'
  when 'r'
    'DISABLED'
  #  pending
  when 'p'
    'DISABLED'
  else
    raise "invalid device status #{status}"
  end
end

def map_device_class_to_api_class(device_type)
  case device_type
  when 'iphone'
    'IPHONE'
  when 'watch'
    'APPLE_WATCH'
  when 'tvOS'
    'APPLE_TV'
  when 'ipad'
    'IPAD'
  when 'ipod'
    'IPOD'
  else
    raise "unsupported device class #{device_type}"
  end
end
