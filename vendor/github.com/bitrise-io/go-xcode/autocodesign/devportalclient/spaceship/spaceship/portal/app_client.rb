require 'spaceship'

require_relative 'common'

module Portal
  # AppClient ...
  class AppClient
    ON_OFF_SERVICES_BY_KEY = {
      'com.apple.security.application-groups' =>               Spaceship::Portal.app_service.app_group,
      'com.apple.developer.in-app-payments' =>                 Spaceship::Portal.app_service.apple_pay,
      'com.apple.developer.associated-domains' =>              Spaceship::Portal.app_service.associated_domains,
      'com.apple.developer.healthkit' =>                       Spaceship::Portal.app_service.health_kit,
      'com.apple.developer.homekit' =>                         Spaceship::Portal.app_service.home_kit,
      'com.apple.developer.networking.HotspotConfiguration' => Spaceship::Portal.app_service.hotspot,
      'com.apple.InAppPurchase' =>                             Spaceship::Portal.app_service.in_app_purchase,
      'inter-app-audio' =>                                     Spaceship::Portal.app_service.inter_app_audio,
      'com.apple.developer.networking.multipath' =>            Spaceship::Portal.app_service.multipath,
      'com.apple.developer.networking.networkextension' =>     Spaceship::Portal.app_service.network_extension,
      'com.apple.developer.nfc.readersession.formats' =>       Spaceship::Portal.app_service.nfc_tag_reading,
      'com.apple.developer.networking.vpn.api' =>              Spaceship::Portal.app_service.vpn_configuration,
      'aps-environment' =>                                     Spaceship::Portal.app_service.push_notification,
      'com.apple.developer.siri' =>                            Spaceship::Portal.app_service.siri_kit,
      'com.apple.developer.pass-type-identifiers' =>           Spaceship::Portal.app_service.passbook,
      'com.apple.external-accessory.wireless-configuration' => Spaceship::Portal.app_service.wireless_accessory
    }.freeze

    ON_OFF_SERVICE_NAME_BY_KEY = {
      'com.apple.security.application-groups' =>               'App Groups',
      'com.apple.developer.in-app-payments' =>                 'Apple Pay',
      'com.apple.developer.associated-domains' =>              'Associated Domains',
      'com.apple.developer.healthkit' =>                       'HealthKit',
      'com.apple.developer.homekit' =>                         'HomeKit',
      'com.apple.developer.networking.HotspotConfiguration' => 'Hotspot',
      'com.apple.InAppPurchase' =>                             'In-App Purchase',
      'inter-app-audio' =>                                     'Inter-App Audio',
      'com.apple.developer.networking.multipath' =>            'Multipath',
      'com.apple.developer.networking.networkextension' =>     'Network Extensions',
      'com.apple.developer.nfc.readersession.formats' =>       'NFC Tag Reading',
      'com.apple.developer.networking.vpn.api' =>              'Personal VPN',
      'aps-environment' =>                                     'Push Notifications',
      'com.apple.developer.siri' =>                            'SiriKit',
      'com.apple.developer.pass-type-identifiers' =>           'Wallet',
      'com.apple.external-accessory.wireless-configuration' => 'Wireless Accessory Configuration'
    }.freeze

    ON_OFF_FEATURE_NAME_BY_KEY = {
      'com.apple.security.application-groups' =>               'APG3427HIY',
      'com.apple.developer.in-app-payments' =>                 'OM633U5T5G',
      'com.apple.developer.associated-domains' =>              'SKC3T5S89Y',
      'com.apple.developer.healthkit' =>                       'HK421J6T7P',
      'com.apple.developer.homekit' =>                         'homeKit',
      'com.apple.developer.networking.HotspotConfiguration' => 'HSC639VEI8',
      'com.apple.InAppPurchase' =>                             'inAppPurchase',
      'inter-app-audio' =>                                     'IAD53UNK2F',
      'com.apple.developer.networking.multipath' =>            'MP49FN762P',
      'com.apple.developer.networking.networkextension' =>     'NWEXT04537',
      'com.apple.developer.nfc.readersession.formats' =>       'NFCTRMAY17',
      'com.apple.developer.networking.vpn.api' =>              'V66P55NK2I',
      'aps-environment' =>                                     'push',
      'com.apple.developer.siri' =>                            'SI015DKUHP',
      'com.apple.developer.pass-type-identifiers' =>           'passbook',
      'com.apple.external-accessory.wireless-configuration' => 'WC421J6T7P'
    }.freeze

    def self.all_services_enabled?(app, entitlements)
      entitlements ||= {}
      app_features = app.details.features

      # on-off services
      entitlements.each_key do |key|
        on_off_app_service = ON_OFF_SERVICES_BY_KEY[key]
        next unless on_off_app_service
        return false unless AppClient.feature_enabled?(key, app_features)
      end

      # Data Protection
      feature_value = app_features['dataProtection']

      data_protection_value = entitlements['com.apple.developer.default-data-protection']
      if data_protection_value == 'NSFileProtectionComplete'
        return false unless feature_value == 'complete'
      elsif data_protection_value == 'NSFileProtectionCompleteUnlessOpen'
        return false unless feature_value == 'unless_open'
      elsif data_protection_value == 'NSFileProtectionCompleteUntilFirstUserAuthentication'
        return false unless feature_value == 'until_first_auth'
      end

      # iCloud
      uses_key_value_storage = !entitlements['com.apple.developer.ubiquity-kvstore-identifier'].nil?
      uses_cloud_documents = false
      uses_cloudkit = false

      icloud_services = entitlements['com.apple.developer.icloud-services']
      unless icloud_services.to_a.empty?
        uses_cloud_documents = icloud_services.include?('CloudDocuments')
        uses_cloudkit = icloud_services.include?('CloudKit')
      end

      if uses_key_value_storage || uses_cloud_documents || uses_cloudkit
        return false unless app_features['cloudKitVersion'].to_i == 2
        return false unless app_features['iCloud']
      end

      true
    end

    def self.sync_app_services(app, entitlements)
      entitlements ||= {}

      details = app.details
      app_features = details.features

      # on-off services
      entitlements.each_key do |key|
        on_off_app_service = ON_OFF_SERVICES_BY_KEY[key]
        next unless on_off_app_service

        service_name = ON_OFF_SERVICE_NAME_BY_KEY[key]

        if AppClient.feature_enabled?(key, app_features)
          Log.print("#{service_name} already enabled")
        else
          Log.success("set #{service_name}: on")
          app = app.update_service(on_off_app_service.on)
        end
      end

      # Data Protection
      feature_value = app_features['dataProtection']

      data_protection_value = entitlements['com.apple.developer.default-data-protection']
      if data_protection_value == 'NSFileProtectionComplete'
        if feature_value == 'complete'
          Log.print('Data Protection: complete already set')
        else
          Log.success('set Data Protection: complete')
          app = app.update_service(Spaceship::Portal.app_service.data_protection.complete)
        end
      elsif data_protection_value == 'NSFileProtectionCompleteUnlessOpen'
        if feature_value == 'unless_open'
          Log.print('Data Protection: unless_open already set')
        else
          Log.success('set Data Protection: unless_open')
          app = app.update_service(Spaceship::Portal.app_service.data_protection.unless_open)
        end
      elsif data_protection_value == 'NSFileProtectionCompleteUntilFirstUserAuthentication'
        if feature_value == 'until_first_auth'
          Log.print('Data Protection: until_first_auth already set')
        else
          Log.success('set Data Protection: until_first_auth')
          app = app.update_service(Spaceship::Portal.app_service.data_protection.until_first_auth)
        end
      end

      # iCloud
      uses_key_value_storage = !entitlements['com.apple.developer.ubiquity-kvstore-identifier'].nil?
      uses_cloud_documents = false
      uses_cloudkit = false

      icloud_services = entitlements['com.apple.developer.icloud-services']
      unless icloud_services.to_a.empty?
        uses_cloud_documents = icloud_services.include?('CloudDocuments')
        uses_cloudkit = icloud_services.include?('CloudKit')
      end

      if uses_key_value_storage || uses_cloud_documents || uses_cloudkit
        if app_features['cloudKitVersion'].to_i == 2
          Log.print('CloudKit: already set')
        else
          Log.success('set CloudKit: on')
          app = app.update_service(Spaceship::Portal.app_service.cloud_kit.cloud_kit)
        end

        if app_features['iCloud']
          Log.print('iCloud: already set')
        else
          Log.success('set iCloud: on')
          app = app.update_service(Spaceship::Portal.app_service.icloud.on)
        end
      end

      app
    end

    def self.feature_enabled?(entitlement_key, app_features)
      feature_key = ON_OFF_FEATURE_NAME_BY_KEY[entitlement_key]
      raise 'not on-off app service key provided' unless feature_key

      feature_value = app_features[feature_key]
      feature_value || false
    end
  end
end
