require 'optparse'
require 'plist'
require 'json'

# -----------------------
# --- Functions
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
# --- Main
# -----------------------

# Input validation
options = {
  export_options_path: nil,
  archive_path: nil,

  export_method: nil,
  upload_bitcode: nil,
  compile_bitcode: nil
}

parser = OptionParser.new do |opts|
  opts.banner = 'Usage: step.rb [options]'
  opts.on('-o', '--export_options_path path', 'Export options path') { |o| options[:export_options_path] = o unless o.to_s == '' }
  opts.on('-a', '--archive_path path', 'Archive path') { |a| options[:archive_path] = a unless a.to_s == '' }

  opts.on('-m', '--export_method string', 'Export method') { |m| options[:export_method] = m unless m.to_s == '' }
  opts.on('-u', '--upload_bitcode string', 'Upload bitcode') { |u| options[:upload_bitcode] = u unless u.to_s == '' }
  opts.on('-c', '--compile_bitcode string', 'Recompile from bitcode') { |c| options[:compile_bitcode] = c unless c.to_s == '' }

  opts.on('-h', '--help', 'Displays Help') do
    puts opts
    exit
  end
end
parser.parse!

log_info('Configs:')
log_details("* export_options_path: #{options[:export_options_path]}")
log_details("* archive_path: #{options[:archive_path]}")

log_details("* export_method: #{options[:export_method]}")
log_details("* upload_bitcode: #{options[:upload_bitcode]}")
log_details("* compile_bitcode: #{options[:compile_bitcode]}")

log_fail('export_options_path not specified') if options[:export_options_path].to_s == ''

if options[:export_method].to_s.empty? && options[:archive_path].to_s.empty?
  log_fail('failed to determin export-method: no archive_path nor export_method provided')
end


mobileprovision_content = collect_provision_info(options[:archive_path])

method = options[:export_method]
method = export_method(mobileprovision_content) if method.to_s.empty?

log_fail('failed to detect export-method or no export_method provided') if method.to_s.empty?

export_options = {}
export_options[:method] = method unless method.to_s.empty?
export_options[:uploadBitcode] = options[:upload_bitcode] if method == 'app-store' && !options[:upload_bitcode].to_s.empty?
export_options[:compileBitcode] = options[:compile_bitcode] if method != 'app-store' && !options[:compile_bitcode].to_s.empty?

plist_content = Plist::Emit.dump(export_options)
log_details('* plist_content:')
puts plist_content.to_s

File.write(options[:export_options_path], plist_content)
