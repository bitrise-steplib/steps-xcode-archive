require 'optparse'
require 'plist'
require 'json'

# -----------------------
# --- functions
# -----------------------
def log_fail(message)
  puts
  puts "\e[31m#{message}\e[0m"
  exit(1)
end

def log_info(message)
  puts
  puts "\e[34m#{message}\e[0m"
end

def log_details(message)
  puts "  #{message}"
end

def collect_provision_info(archive_path)
  applications_path = File.join(archive_path, '/Products/Applications')
  mobileprovision_path = Dir[File.join(applications_path, '*.app/embedded.mobileprovision')].first

  log_fail('No mobileprovision_path found') if mobileprovision_path.nil?

  content = {}
  plist = Plist.parse_xml(`security cms -D -i "#{mobileprovision_path}"`)

  plist.each do |key, value|
    next if key == 'DeveloperCertificates'

    parse_value = nil
    case value
    when Hash
      parse_value = value
    when Array
      parse_value = value
    else
      parse_value = value.to_s
    end

    content[key] = parse_value
  end

  content
end

def export_method(mobileprovision_content)
  # if ProvisionedDevices: !nil & "get-task-allow": true -> development
  # if ProvisionedDevices: !nil & "get-task-allow": false -> ad-hoc
  # if ProvisionedDevices: nil & "ProvisionsAllDevices": "true" -> enterprise
  # if ProvisionedDevices: nil & ProvisionsAllDevices: nil -> app-store
  if mobileprovision_content['ProvisionedDevices'].nil?
    return 'enterprise' if !mobileprovision_content['ProvisionsAllDevices'].nil? && (mobileprovision_content['ProvisionsAllDevices'] == true || mobileprovision_content['ProvisionsAllDevices'] == 'true')
    return 'app-store'
  else
    unless mobileprovision_content['Entitlements'].nil?
      entitlements = mobileprovision_content['Entitlements']
      return 'development' if !entitlements['get-task-allow'].nil? && (entitlements['get-task-allow'] == true || entitlements['get-task-allow'] == 'true')
      return 'ad-hoc'
    end
  end
  return 'development'
end

# -----------------------
# --- main
# -----------------------

# Input validation
options = {
  export_options_path: nil,
  archive_path: nil
}

parser = OptionParser.new do|opts|
  opts.banner = 'Usage: step.rb [options]'
  opts.on('-o', '--export_options_path path', 'Export options path') { |o| options[:export_options_path] = o unless o.to_s == '' }
  opts.on('-a', '--archive_path path', 'Archive path') { |a| options[:archive_path] = a unless a.to_s == '' }
  opts.on('-h', '--help', 'Displays Help') do
    puts opts
    exit
  end
end
parser.parse!

log_fail('export_options_path not specified') if options[:export_options_path].to_s == ''
log_fail('archive_path not specified') if options[:archive_path].to_s == ''

log_info('Configs:')
log_details("* export_options_path: #{options[:export_options_path]}")

mobileprovision_content = collect_provision_info(options[:archive_path])
method = export_method(mobileprovision_content)

export_options = {}
export_options[:method] = method unless method.nil?

log_details("* export_options: #{export_options}")

plist_content = Plist::Emit.dump(export_options)
log_details('* plist_content:')
puts "#{plist_content}"

File.write(options[:export_options_path], plist_content)
