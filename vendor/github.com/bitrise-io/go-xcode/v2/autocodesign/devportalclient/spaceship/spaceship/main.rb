require_relative 'portal/auth_client'
require_relative 'certificates'
require_relative 'profiles'
require_relative 'app'
require_relative 'devices'
require_relative 'log'
require 'optparse'

begin
  options = {}
  OptionParser.new do |opt|
    opt.on('--username USERNAME') { |o| options[:username] = o }
    opt.on('--password PASSWORD') { |o| options[:password] = o }
    opt.on('--session SESSION') { |o| options[:session] = Base64.decode64(o) }
    opt.on('--team-id TEAM_ID') { |o| options[:team_id] = o }
    opt.on('--subcommand SUBCOMMAND') { |o| options[:subcommand] = o }
    opt.on('--bundle-id BUNDLE_ID') { |o| options[:bundle_id] = o }
    opt.on('--bundle-id-name BUNDLE_ID_NAME') { |o| options[:bundle_id_name] = o }
    opt.on('--id ID') { |o| options[:id] = o }
    opt.on('--name NAME') { |o| options[:name] = o }
    opt.on('--certificate-id CERTIFICATE') { |o| options[:certificate_id] = o }
    opt.on('--profile-name PROFILE_NAME') { |o| options[:profile_name] = o }
    opt.on('--profile-type PROFILE_TYPE') { |o| options[:profile_type] = o }
    opt.on('--entitlements ENTITLEMENTS') { |o| options[:entitlements] = Base64.decode64(o) }
    opt.on('--udid UDID') { |o| options[:udid] = o }
  end.parse!

  FastlaneCore::Globals.verbose = true

  result = '{}'

  if options[:subcommand] == 'login'
    begin
      team_id = Portal::AuthClient.login(options[:username], options[:password], options[:session], options[:team_id])
      result = team_id
    rescue => e
      puts "\nApple ID authentication failed: #{e}"
      exit(1)
    end
  else
    Portal::AuthClient.restore_from_session(options[:username], options[:team_id])

    case options[:subcommand]
    when 'list_dev_certs'
      client = CertificateHelper.new
      result = client.list_dev_certs
    when 'list_dist_certs'
      client = CertificateHelper.new
      result = client.list_dist_certs
    when 'list_profiles'
      result = list_profiles(options[:profile_type], options[:profile_name])
    when 'get_app'
      result = get_app(options[:bundle_id])
    when 'create_app'
      result = create_app(options[:bundle_id], options[:bundle_id_name])
    when 'delete_profile'
      delete_profile(options[:id])
      result = { status: 'OK' }
    when 'create_profile'
      result = create_profile(options[:profile_type], options[:bundle_id], options[:certificate_id], options[:profile_name])
    when 'check_bundleid'
      entitlements = JSON.parse(options[:entitlements])
      check_bundleid(options[:bundle_id], entitlements)
    when 'sync_bundleid'
      entitlements = JSON.parse(options[:entitlements])
      sync_bundleid(options[:bundle_id], entitlements)
    when 'list_devices'
      result = list_devices
    when 'register_device'
      result = register_device(options[:udid], options[:name])
    else
      raise "Unknown subcommand: #{options[:subcommand]}"
    end
  end

  response = { data: result }
  puts response.to_json.to_s
rescue RetryNeeded => e
  result = { retry: true, error: "#{e.cause}" }
  puts result.to_json.to_s
rescue Spaceship::BasicPreferredInfoError, Spaceship::UnexpectedResponse => e
  result = { error: "#{e.preferred_error_info&.join("\n") || e.to_s}, stacktrace: #{e.backtrace.join("\n")}" }
  puts result.to_json.to_s
rescue => e
  result = { error: "#{e}, stacktrace: #{e.backtrace.join("\n")}" }
  puts result.to_json.to_s
end
