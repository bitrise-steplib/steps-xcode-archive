#!/bin/bash

THIS_SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

set -e

#=======================================
# Functions
#=======================================

RESTORE='\033[0m'
RED='\033[00;31m'
YELLOW='\033[00;33m'
BLUE='\033[00;34m'
GREEN='\033[00;32m'

function color_echo {
	color=$1
	msg=$2
	echo -e "${color}${msg}${RESTORE}"
}

function echo_fail {
	msg=$1
	echo
	color_echo "${RED}" "${msg}"
	exit 1
}

function echo_warn {
	msg=$1
	color_echo "${YELLOW}" "${msg}"
}

function echo_info {
	msg=$1
	echo
	color_echo "${BLUE}" "${msg}"
}

function echo_details {
	msg=$1
	echo "  ${msg}"
}

function echo_done {
	msg=$1
	color_echo "${GREEN}" "  ${msg}"
}

function validate_required_input {
	key=$1
	value=$2
	if [ -z "${value}" ] ; then
		echo_fail "[!] Missing required input: ${key}"
	fi
}

function validate_required_input_with_options {
	key=$1
	value=$2
	options=$3

	validate_required_input "${key}" "${value}"

	found="0"
	for option in "${options[@]}" ; do
		if [ "${option}" == "${value}" ] ; then
			found="1"
		fi
	done

	if [ "${found}" == "0" ] ; then
		echo_fail "Invalid input: (${key}) value: (${value}), valid options: ($( IFS=$", "; echo "${options[*]}" ))"
	fi
}

#=======================================
# Main
#=======================================

#
# Validate parameters
echo_info "Configs:"
echo_details "* workdir: $workdir"
echo_details "* project_path: $project_path"
echo_details "* scheme: $scheme"
echo_details "* configuration: $configuration"
echo_details "* output_dir: $output_dir"
echo_details "* force_provisioning_profile: $force_provisioning_profile"
echo_details "* force_provisioning_profile_specifier: $force_provisioning_profile_specifier"
echo_details "* force_code_sign_identity: $force_code_sign_identity"
echo_details "* export_options_path: $export_options_path"
echo_details "* is_clean_build: $is_clean_build"
echo_details "* output_tool: $output_tool"
echo_details "* xcodebuild_options: $xcodebuild_options"
echo_details "* is_export_xcarchive_zip: $is_export_xcarchive_zip"

echo

validate_required_input "project_path" $project_path
validate_required_input "scheme" $scheme
validate_required_input "is_clean_build" $is_clean_build
validate_required_input "output_dir" $output_dir
validate_required_input "output_tool" $output_tool
validate_required_input "is_export_xcarchive_zip" $is_export_xcarchive_zip

options=("xcpretty"  "xcodebuild")
validate_required_input_with_options "output_tool" $output_tool "${options[@]}"

options=("yes"  "no")
validate_required_input_with_options "is_clean_build" $is_clean_build "${options[@]}"
validate_required_input_with_options "is_export_xcarchive_zip" $is_export_xcarchive_zip "${options[@]}"

# Detect Xcode major version
xcode_major_version=""
major_version_regex="Xcode ([0-9]).[0-9]"
out=$(xcodebuild -version)
if [[ "${out}" =~ ${major_version_regex} ]] ; then
	xcode_major_version="${BASH_REMATCH[1]}"
fi

if [ "${xcode_major_version}" -lt "6" ] ; then
	echo_fail "Invalid xcode major version: ${xcode_major_version}, should be greater then 6"
fi

IFS=$'\n'
xcodebuild_version_split=($out)
unset IFS

xcodebuild_version="${xcodebuild_version_split[0]} (${xcodebuild_version_split[1]})"
echo_details "* xcodebuild_version: $xcodebuild_version"

# Detect xcpretty version
xcpretty_version=""
if [[ "${output_tool}" == "xcpretty" ]] ; then
	xcpretty_version=$(xcpretty --version)
	exit_code=$?
	if [[ $exit_code != 0 || -z "$xcpretty_version" ]] ; then
		echo_fail "xcpretty is not installed
		For xcpretty installation see: 'https://github.com/supermarin/xcpretty',
		or use 'xcodebuild' as 'output_tool'.
		"
	fi

	echo_details "* xcpretty_version: $xcpretty_version"
fi

# export_options_path & Xcode 6
if [ ! -z "${export_options_path}" ] && [[ "${xcode_major_version}" == "6" ]] ; then
	echo_warn "xcode_major_version = 6, export_options_path only used if xcode_major_version > 6"
	export_options_path=""
fi

# force_provisioning_profile & force_provisioning_profile_specifier
if [ ! -z "${force_provisioning_profile_specifier}" ] && [[ "${xcode_major_version}" -lt "8" ]] ; then
	echo_warn "xcode_major_version = ${xcode_major_version}, force_provisioning_profile_specifier used if xcode_major_version >= 8"
	force_provisioning_profile_specifier=""
fi

if [ ! -z "${force_provisioning_profile_specifier}" ] && [ ! -z "${force_provisioning_profile}" ] ; then
	echo_warn "both force_provisioning_profile_specifier and force_provisioning_profile defined, using force_provisioning_profile_specifier..."
	force_provisioning_profile=""
fi

# Project-or-Workspace flag
if [[ "${project_path}" == *".xcodeproj" ]]; then
	CONFIG_xcode_project_action="-project"
elif [[ "${project_path}" == *".xcworkspace" ]]; then
	CONFIG_xcode_project_action="-workspace"
else
	echo_fail "Failed to get valid project file (invalid project file): ${project_path}"
fi
echo_details "* CONFIG_xcode_project_action: $CONFIG_xcode_project_action"

echo

# abs out dir pth
mkdir -p "${output_dir}"
cd "${output_dir}"
output_dir="$(pwd)"
cd -

# output files
archive_tmp_dir=$(mktemp -d -t bitrise-xcarchive)

archive_path="${archive_tmp_dir}/${scheme}.xcarchive"
echo_details "* archive_path: $archive_path"

ipa_path="${output_dir}/${scheme}.ipa"
echo_details "* ipa_path: $ipa_path"

dsym_zip_path="${output_dir}/${scheme}.dSYM.zip"
echo_details "* dsym_zip_path: $dsym_zip_path"

# work dir
if [ ! -z "${workdir}" ] ; then
	echo_info "Switching to working directory: ${workdir}"
	cd "${workdir}"
fi

#
# Main

#
# Bit of cleanup
if [ -f "${ipa_path}" ] ; then
	echo_warn "IPA at path (${ipa_path}) already exists - removing it"
	rm "${ipa_path}"
fi

#
# Create the Archive with Xcode Command Line tools
echo_info "Create the Archive ..."

archive_cmd="xcodebuild ${CONFIG_xcode_project_action} \"${project_path}\""
archive_cmd="$archive_cmd -scheme \"${scheme}\""

if [ ! -z "${configuration}" ] ; then
	archive_cmd="$archive_cmd -configuration \"${configuration}\""
fi

if [[ "${is_clean_build}" == "yes" ]] ; then
	archive_cmd="$archive_cmd clean"
fi

archive_cmd="$archive_cmd archive -archivePath \"${archive_path}\""

if [[ ! -z "${force_provisioning_profile_specifier}" ]] ; then
	echo_details "Forcing Provisioning Profile Specifier: ${force_provisioning_profile_specifier}"

	archive_cmd="$archive_cmd PROVISIONING_PROFILE_SPECIFIER=\"${force_provisioning_profile_specifier}\""
fi

if [[ ! -z "${force_provisioning_profile}" ]] ; then
	echo_details "Forcing Provisioning Profile: ${force_provisioning_profile}"

	archive_cmd="$archive_cmd PROVISIONING_PROFILE=\"${force_provisioning_profile}\""
fi


if [[ ! -z "${force_code_sign_identity}" ]] ; then
	echo_details "Forcing Code Signing Identity: ${force_code_sign_identity}"

	archive_cmd="$archive_cmd CODE_SIGN_IDENTITY=\"${force_code_sign_identity}\""
fi

if [ ! -z "${xcodebuild_options}" ] ; then
	archive_cmd="$archive_cmd ${xcodebuild_options}"
fi

if [[ "${output_tool}" == "xcpretty" ]] ; then
	archive_cmd="set -o pipefail && $archive_cmd | xcpretty"
fi

echo_details "$ $archive_cmd"
echo

eval $archive_cmd

#
# Exporting the ipa with Xcode Command Line tools

# You'll get a "Error Domain=IDEDistributionErrorDomain Code=14 "No applicable devices found."" error
# if $GEM_HOME is set and the project's directory includes a Gemfile - to fix this
# we'll unset GEM_HOME as that's not required for xcodebuild anyway.
# This probably fixes the RVM issue too, but that still should be tested.
unset GEM_HOME

#
export_command="xcodebuild -exportArchive"

if [[ "${xcode_major_version}" == "6" ]] ; then
	echo_info "Exporting IPA from generated Archive ..."

	#
	# Xcode major version = 6
	#

	#
	# Get the name of the profile which was used for creating the archive
	# --> Search for embedded.mobileprovision in the xcarchive.
	#     It should contain a .app folder in the xcarchive folder
	#     under the Products/Applications folder
	embedded_mobile_prov_path=""

	# We need -maxdepth 2 because of the `*.app` directory
	IFS=$'\n'
	for a_emb_path in $(find "${archive_path}/Products/Applications" -type f -maxdepth 2 -ipath '*.app/embedded.mobileprovision')
	do
		echo " * embedded.mobileprovision: ${a_emb_path}"
		if [ ! -z "${embedded_mobile_prov_path}" ] ; then
			echo_fail "More than one \`embedded.mobileprovision\` found in \`${archive_path}/Products/Applications/*.app\`"
		fi
		embedded_mobile_prov_path="${a_emb_path}"
	done
	unset IFS

	if [ -z "${embedded_mobile_prov_path}" ] ; then
		echo_fail "No \`embedded.mobileprovision\` found in \`${archive_path}/Products/Applications/*.app\`"
	fi

	#
	# We have the mobileprovision file - let's get the Profile name from it
	profile_name=`/usr/libexec/PlistBuddy -c 'Print :Name' /dev/stdin <<< $(security cms -D -i "${embedded_mobile_prov_path}")`
	if [ $? -ne 0 ] ; then
		echo_fail "Missing embedded mobileprovision in xcarchive"
	fi
	echo_details "Found Profile Name for signing: ${profile_name}"

	#
	# Use the Provisioning Profile name to export the IPA
	export_command="$export_command -exportFormat ipa"
	export_command="$export_command -archivePath \"${archive_path}\""
	export_command="$export_command -exportPath \"${ipa_path}\""
	export_command="$export_command -exportProvisioningProfile \"${profile_name}\""

	if [[ "${output_tool}" == "xcpretty" ]] ; then
		export_command="set -o pipefail && $export_command | xcpretty"
	fi

	echo_details "$ $export_command"
	echo

	eval $export_command
else
	#
	# Xcode major version > 6
	#

	echo_info "Generating exportOptionsPlist..."

	if [ -z "${export_options_path}" ] ; then
		export_options_path="${output_dir}/export_options.plist"
		curr_pwd="$(pwd)"
		cd "${THIS_SCRIPT_DIR}"
		bundle install
		bundle exec ruby "./generate_export_options.rb" \
			-o "${export_options_path}" \
			-a "${archive_path}"
		cd "${curr_pwd}"
	fi

	echo_info "Exporting IPA from generated Archive..."
	#
	# Because of an RVM issue which conflicts with `xcodebuild`'s new
	#  `-exportOptionsPlist` option
	# link: https://github.com/bitrise-io/steps-xcode-archive/issues/13
	command_exists () {
		command -v "$1" >/dev/null 2>&1 ;
	}
	if command_exists rvm ; then
		echo_warn "Applying RVM 'fix'"

		[[ -s "$HOME/.rvm/scripts/rvm" ]] && source "$HOME/.rvm/scripts/rvm"
		rvm use system
	fi

	tmp_dir=$(mktemp -d -t bitrise-xcarchive)

	export_command="$export_command -archivePath \"${archive_path}\""
	export_command="$export_command -exportPath \"${tmp_dir}\""
	export_command="$export_command -exportOptionsPlist \"${export_options_path}\""

	if [[ "${output_tool}" == "xcpretty" ]] ; then
		export_command="set -o pipefail && $export_command | xcpretty"
	fi

	echo_details "$ $export_command"
	echo

	eval $export_command
	echo

	# Searching for ipa
	exported_ipa_path=""
	IFS=$'\n'
	for a_file_path in $(find "${tmp_dir}" -maxdepth 1 -mindepth 1)
	do
		filename=$(basename "$a_file_path")

		mv "${a_file_path}" "${output_dir}"

		regex=".*.ipa"
		if [[ "${filename}" =~ $regex ]]; then
			if [[ -z "${exported_ipa_path}" ]] ; then
				exported_ipa_path="${output_dir}/${filename}"
			else
				echo_warn "More than one ipa file found"
			fi
		fi
	done
	unset IFS

	if [[ -z "${exported_ipa_path}" ]] ; then
		echo_fail "No ipa file found"
	fi

	if [ ! -e "${exported_ipa_path}" ] ; then
		echo_fail "Failed to move ipa to output dir"
	fi

	ipa_path="${exported_ipa_path}"
fi

#
# Export *.ipa path
envman add --key BITRISE_IPA_PATH --value "${ipa_path}"
echo_done 'The IPA path is now available in the Environment Variable: $BITRISE_IPA_PATH'

#
# Export app directory
echo_info "Exporting .app directory..."

IFS=$'\n'
app_directory=""
for a_app_directory in $(find "${archive_path}/Products/Applications" -type d -name '*.app')
do
	echo " * a_app_directory: ${a_app_directory}"
	if [ ! -z "${app_directory}" ] ; then
		echo_warn "More than one \`.app directory\` found in \`${archive_path}/Products/Applications\`"
	fi
	app_directory="${a_app_directory}"
done
unset IFS

envman add --key BITRISE_APP_DIR_PATH --value "${app_directory}"
echo_done 'The .app directory is now available in the Environment Variable: $BITRISE_APP_DIR_PATH'

#
# dSYM handling
# get the .app.dSYM folders from the dSYMs archive folder
echo_info "Exporting dSym from generated Archive..."

archive_dsyms_folder="${archive_path}/dSYMs"
ls "${archive_dsyms_folder}"
app_dsym_count=0
app_dsym_path=""

IFS=$'\n'
for a_app_dsym in $(find "${archive_dsyms_folder}" -type d -name "*.app.dSYM") ; do
  app_dsym_count=$[app_dsym_count + 1]
  app_dsym_path="${a_app_dsym}"
done
unset IFS

DSYM_PATH=""
if [ ${app_dsym_count} -eq 1 ] ; then
  echo_details "dSYM found at: ${app_dsym_path}"

  if [ -d "${app_dsym_path}" ] ; then
    DSYM_PATH="${app_dsym_path}"
  else
    echo_warn "Found dSYM path is not a directory!"
  fi
else
  if [ ${app_dsym_count} -eq 0 ] ; then
    echo_warn "No dSYM found!"
		echo_details "To generate debug symbols (dSYM) go to your Xcode Project's Settings - *Build Settings - Debug Information Format* and set it to *DWARF with dSYM File*."
  else
    echo_warn "More than one dSYM found!"
  fi
fi

# Generate dSym zip
if [[ ! -z "${DSYM_PATH}" && -d "${DSYM_PATH}" ]] ; then
  echo_info "Generating zip for dSym..."

  dsym_parent_folder=$( dirname "${DSYM_PATH}" )
  dsym_fold_name=$( basename "${DSYM_PATH}" )
  # cd into dSYM parent to not to store full
  #  paths in the ZIP
  cd "${dsym_parent_folder}"
  /usr/bin/zip -rTy \
    "${dsym_zip_path}" \
    "${dsym_fold_name}"
	cd -

	envman add --key BITRISE_DSYM_PATH --value "${dsym_zip_path}"
	echo_done 'The dSYM path is now available in the Environment Variable: $BITRISE_DSYM_PATH'
else
	echo_warn "No dSYM found (or not a directory: ${DSYM_PATH})"
fi

#
# Export *.xcarchive path
if [[ "$is_export_xcarchive_zip" == "yes" ]] ; then
	echo_info "Exporting the Archive..."

	xcarchive_parent_folder=$( dirname "${archive_path}" )
	xcarchive_fold_name=$( basename "${archive_path}" )
	xcarchive_zip_path="${output_dir}/${scheme}.xcarchive.zip"
	# cd into dSYM parent to not to store full
	#  paths in the ZIP
	cd "${xcarchive_parent_folder}"
	/usr/bin/zip -rTy \
		"${xcarchive_zip_path}" \
		"${xcarchive_fold_name}"
	cd -

	envman add --key BITRISE_XCARCHIVE_PATH --value "${xcarchive_zip_path}"
	echo_done 'The xcarchive path is now available in the Environment Variable: $BITRISE_XCARCHIVE_PATH'
fi

exit 0
