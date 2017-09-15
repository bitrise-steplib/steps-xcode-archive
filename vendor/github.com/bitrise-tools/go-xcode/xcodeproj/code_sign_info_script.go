package xcodeproj

const codeSignInfoScriptContent = `require 'xcodeproj'
require 'json'

def contained_projects(project_or_workspace_pth)
  if File.extname(project_or_workspace_pth) == '.xcodeproj'
    [project_or_workspace_pth]
  else
    workspace_contents_pth = File.join(project_or_workspace_pth, 'contents.xcworkspacedata')
    workspace_contents = File.read(workspace_contents_pth)

    project_paths = workspace_contents.scan(/\"group:(.*)\"/).collect do |current_match|
      File.join(File.expand_path('..', project_or_workspace_pth), current_match.first)
    end

    project_paths.find_all do |current_match|
      # skip cocoapods projects
      !current_match.end_with?('Pods/Pods.xcodeproj')
    end
  end
end

def read_scheme(project_path, scheme_name, user_name)
  shared_schemes = Xcodeproj::Project.schemes(project_path) || []
  is_shared = shared_schemes.include? scheme_name

  scheme_pth = ''
  if is_shared
    scheme_pth = File.join(project_path, 'xcshareddata', 'xcschemes', scheme_name + '.xcscheme')
  else
    scheme_pth = File.join(project_path, 'xcuserdata', user_name + '.xcuserdatad', 'xcschemes', scheme_name + '.xcscheme')
  end

  return nil unless File.exist? scheme_pth
  Xcodeproj::XCScheme.new(scheme_pth)
end

def find_project_with_scheme(project_or_workspace_pth, scheme_name, user_name)
  project_paths = contained_projects(project_or_workspace_pth)
  project_path = project_paths.find do |project_path|
    scheme = read_scheme(project_path, scheme_name, user_name)
    !scheme.nil?
  end
  return nil unless project_path
  Xcodeproj::Project.open(project_path)
end

def find_build_action_target(project, scheme)
  build_action = scheme.build_action
  return nil unless build_action

  entries = build_action.entries || []
  return nil unless entries.count > 0

  entry = entries[0]
  buildable_references = entry.buildable_references || []
  return nil unless buildable_references.count > 0

  buildable_reference = buildable_references[0]
  target_name = buildable_reference.target_name

  project.targets.find { |t| t.name == target_name }
end

def find_archive_action_build_configuration_name(scheme)
  archive_action = scheme.archive_action
  return nil unless archive_action

  archive_action.build_configuration
end

def read_target_attributes(project, target)
  attributes = project.root_object.attributes['TargetAttributes']
  attributes[target.uuid]
end

def collect_dependent_targets(target, dependent_targets)
  dependent_targets.push(target)

  dependencies = target.dependencies || []
  return dependent_targets if dependencies.empty?

  dependencies.each do |dependency|
    dependent_target = dependency.target
    collect_dependent_targets(dependent_target, dependent_targets)
  end

  dependent_targets
end

def read_code_sign_infos(project_or_workspace_pth, scheme_name, user_name, build_configuration_name)
  project = find_project_with_scheme(project_or_workspace_pth, scheme_name, user_name)
  raise "project does not contain scheme: #{scheme_name}" unless project

  scheme = read_scheme(project.path, scheme_name, user_name)
  raise "project does not contain scheme: #{scheme_name}" unless scheme

  target = find_build_action_target(project, scheme)
  raise 'scheme does not contain buildable target' unless target

  targets = []
  targets = collect_dependent_targets(target, targets)
  raise 'failed to collect targets to analyze' if targets.to_a.empty?

  target_code_sign_infos = {}

  targets.each do |target|
    target_attributes = read_target_attributes(project, target)
    raise "not target attributes found for target (#{target_name})" unless target_attributes
  
    provisioning_style = target_attributes['ProvisioningStyle'] || ''
  
    if build_configuration_name.to_s.empty?
      build_configuration_name = find_archive_action_build_configuration_name(scheme)
      raise 'no default configuration found for archive action' unless build_configuration_name
    end
  
    build_configuration = target.build_configuration_list.build_configurations.find { |c| c.name == build_configuration_name }
    raise "no build configuration found with name: #{build_configuration_name}" unless build_configuration
  
    bundle_id = build_configuration.resolve_build_setting('PRODUCT_BUNDLE_IDENTIFIER') || ''
    code_sign_identity = build_configuration.resolve_build_setting('CODE_SIGN_IDENTITY') || ''
    provisioning_profile_specifier = build_configuration.resolve_build_setting('PROVISIONING_PROFILE_SPECIFIER') || ''
    provisioning_profile = build_configuration.resolve_build_setting('PROVISIONING_PROFILE') || ''
    info_plist_file = build_configuration.resolve_build_setting('INFOPLIST_FILE') || ''
    info_plist_file = File.join(File.dirname(project_or_workspace_pth), info_plist_file) unless info_plist_file.empty?

    code_sign_info = {
      info_plist_file: info_plist_file,

      configuration: build_configuration.name,

      provisioning_style: provisioning_style,
      bundle_id: bundle_id,
      code_sign_identity: code_sign_identity,
      provisioning_profile_specifier: provisioning_profile_specifier,
      provisioning_profile: provisioning_profile
    }

    target_code_sign_infos[target.name] = code_sign_info
  end

  target_code_sign_infos
end

begin
  project_path = ENV['project']
  scheme_name = ENV['scheme']
  configuration = ENV['configuration']
  user_name = ENV['user']

  mapping = read_code_sign_infos(project_path, scheme_name, user_name, configuration)
  result = {
    data: mapping
  }
  result_json = JSON.pretty_generate(result).to_s
  puts result_json
rescue => e
  error_message = e.to_s + "\n" + e.backtrace.join("\n")
  result = {
    error: error_message
  }
  result_json = result.to_json.to_s
  puts result_json
  exit(1)
end
`

const gemfileContent = `source "https://rubygems.org"
gem "xcodeproj"
gem "json"
`
