#!/bin/bash

set -e

#
# Required parameters
if [ -z "${project_path}" ] ; then
	echo "[!] Missing required input: project_path"
	exit 1
fi

if [ -z "${scheme}" ] ; then
	echo "[!] Missing required input: scheme"
	exit 1
fi

if [ -z "${output_dir}" ] ; then
	echo "[!] Missing required input: output_dir"
	exit 1
fi

#
# Project-or-Workspace flag
if [[ "${project_path}" == *".xcodeproj" ]]; then
	CONFIG_xcode_project_action="-project"
elif [[ "${project_path}" == *".xcworkspace" ]]; then
	CONFIG_xcode_project_action="-workspace"
else
	echo "Failed to get valid project file (invalid project file): ${project_path}"
	exit 1
fi

# abs out dir pth
mkdir -p "${output_dir}"
cd "${output_dir}"
output_dir="$(pwd)"
cd -

archive_path="${output_dir}/${scheme}.xcarchive"
ipa_path="${output_dir}/${scheme}.ipa"

if [ -z "${workdir}" ] ; then
	workdir="$(pwd)"
fi

#
# Print configs
echo " * CONFIG_xcode_project_action: ${CONFIG_xcode_project_action}"
echo " * project_path: ${project_path}"
echo " * scheme: ${scheme}"
echo " * workdir: ${workdir}"
echo " * output_dir: ${output_dir}"
echo " * archive_path: ${archive_path}"
echo " * ipa_path: ${ipa_path}"

if [ ! -z "${workdir}" ] ; then
	echo
	echo " -> Switching to working directory: ${workdir}"
	cd "${workdir}"
fi


#
# Cleanup function
function finalcleanup {
	local fail_msg="$1"

	echo "-> finalcleanup"

	if [ ! -z "${fail_msg}" ] ; then
		echo " [!] ERROR: ${fail_msg}"
		exit 1
	fi
}


#
# Main

#
# Bit of cleanup
if [ -f "${ipa_path}" ] ; then
	echo " (!) IPA at path (${ipa_path}) already exists - removing it"
	rm "${ipa_path}"
fi

set -v

#
# Create the Archive with Xcode Command Line tools
xcodebuild ${CONFIG_xcode_project_action} "${project_path}" \
	-scheme "${scheme}" \
	clean archive -archivePath "${archive_path}" \
	-verbose

set +v

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
		finalcleanup "More than one \`embedded.mobileprovision\` found in \`${archive_path}/Products/Applications/*.app\`"
		exit 1
	fi
	embedded_mobile_prov_path="${a_emb_path}"
done
unset IFS

if [ -z "${embedded_mobile_prov_path}" ] ; then
	finalcleanup "No \`embedded.mobileprovision\` found in \`${archive_path}/Products/Applications/*.app\`"
	exit 1
fi

#
# We have the mobileprovision file - let's get the Profile name from it
profile_name=`/usr/libexec/PlistBuddy -c 'Print :Name' /dev/stdin <<< $(security cms -D -i "${embedded_mobile_prov_path}")`
if [ $? -ne 0 ] ; then
	finalcleanup "Missing embedded mobileprovision in xcarchive"
fi

echo " (i) Found Profile Name for signing: ${profile_name}"

set -v

#
# Use the Provisioning Profile name to export the IPA
xcodebuild -exportArchive \
	-exportFormat ipa \
	-archivePath "${archive_path}" \
	-exportPath "${ipa_path}" \
	-exportProvisioningProfile "${profile_name}"

set +v

echo " (i) The IPA is now available at: ${ipa_path}"
envman add --key BITRISE_IPA_PATH --value "${ipa_path}"
echo ' (i) The IPA path is now available in the Environment Variable: $BITRISE_IPA_PATH'

exit 0
