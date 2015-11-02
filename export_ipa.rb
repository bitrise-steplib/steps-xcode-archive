require 'plist'

# -----------------------
# --- functions
# -----------------------

def fail_with_message(message)
  puts "\e[31m#{message}\e[0m"
  exit(1)
end

# -----------------------
# --- main
# -----------------------

archive_path = ARGV[0]
output_dir = ARGV[1]

fail_with_message('No archive_path specified') unless archive_path
fail_with_message('No output_dir specified') unless output_dir

info_plist_path = File.join(archive_path, 'Info.plist')
unless File.exist?(info_plist_path)
  puts '(!) No Info.plist found, search for other plist'
  info_plist_paths = Dir[archive_path, '*.plist']
  fail_with_message('More then 1 plist found') unless info_plist_paths.nil? || info_plist_paths.count == 1

  info_plist_path = info_plist_paths.first
end

fail_with_message('No Info.plist found') unless info_plist_path

infos = Plist.parse_xml(info_plist_path)
fail_with_message('Failed to read ipa name') if infos.nil? || infos['Name'].nil?

ipa_name = infos['Name']
ipa_path = File.join(output_dir, "#{ipa_name}.ipa")
unless File.exist?(ipa_path)
  puts "(!) No #{ipa_name}.ipa found, search for other .ipa"
  ipa_paths = Dir[output_dir, '*.ipa']
  fail_with_message('More then 1 ipa found') unless ipa_paths.nil? || ipa_paths.count == 1

  ipa_path = ipa_paths.first
end

fail_with_message('No ipa found') unless ipa_path
puts " (i) The IPA is now available at: #{ipa_path}"

system("envman add --key BITRISE_IPA_PATH --value \"#{ipa_path}\"")
puts ' (i) The IPA path is now available in the Environment Variable: $BITRISE_IPA_PATH'
