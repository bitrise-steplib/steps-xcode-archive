require 'spaceship'

require_relative 'common'

module Portal
  # CertificateClient ...
  class CertificateClient
    def self.download_development_certificates
      development_certificates = []
      run_or_raise_preferred_error_message do
        development_certificates = Spaceship::Portal.certificate.development.all
        development_certificates.concat(Spaceship::Portal.certificate.apple_development.all)
      end

      certificates = []
      development_certificates.each do |cert|
        if cert.can_download
          certificates.push(cert)
        else
          Log.debug("development certificate: #{cert.name} is not downloadable, skipping...")
        end
      end

      certificates
    end

    def self.download_production_certificates
      production_certificates = []
      run_or_raise_preferred_error_message do
        production_certificates = Spaceship::Portal.certificate.production.all
        production_certificates.concat(Spaceship::Portal.certificate.apple_distribution.all)
      end

      certificates = []
      production_certificates.each do |cert|
        if cert.can_download
          certificates.push(cert)
        else
          Log.debug("production certificate: #{cert.name} is not downloadable, skipping...")
        end
      end

      if production_certificates.to_a.empty?
        run_or_raise_preferred_error_message { production_certificates = Spaceship::Portal.certificate.in_house.all }

        production_certificates.each do |cert|
          if cert.can_download
            certificates.push(cert)
          else
            Log.debug("production certificate: #{cert.name} is not downloadable, skipping...")
          end
        end
      end

      certificates
    end
  end
end
