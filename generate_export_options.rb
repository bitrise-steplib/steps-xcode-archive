require 'optparse'
require 'plist'
require 'json'

# -----------------------
# --- functions
# -----------------------

def fail_with_message(message)
  puts "\e[31m#{message}\e[0m"
  exit(1)
end

def collect_provision_info(archive_path)
  applications_path = File.join(archive_path, '/Products/Applications')
  mobileprovision_path = Dir[File.join(applications_path, '*.app/embedded.mobileprovision')].first

  fail_with_message('No mobileprovision_path found') if mobileprovision_path.nil?

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

puts

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

fail_with_message('export_options_path not specified') unless options[:export_options_path]
puts "(i) export_options_path: #{options[:export_options_path]}"

fail_with_message('archive_path not specified') unless options[:archive_path]
puts "(i) archive_path: #{options[:archive_path]}"

puts
puts '==> Collect infos from mobileprovision'

mobileprovision_content = collect_provision_info(options[:archive_path])
# team_id = mobileprovision_content['TeamIdentifier'].first
method = export_method(mobileprovision_content)

puts
puts '==> Create export options'

export_options = {}
# export_options[:teamID] = team_id unless team_id.nil?
export_options[:method] = method unless method.nil?

puts
puts " (i) export_options: #{export_options}"
plist_content = Plist::Emit.dump(export_options)
puts " (i) plist_content: #{plist_content}"
puts " (i) saving into file: #{options[:export_options_path]}"
File.write(options[:export_options_path], plist_content)
