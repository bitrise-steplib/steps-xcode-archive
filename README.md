# Xcode Archive & Export for iOS

[![Step changelog](https://shields.io/github/v/release/bitrise-steplib/steps-xcode-archive?include_prereleases&label=changelog&color=blueviolet)](https://github.com/bitrise-steplib/steps-xcode-archive/releases)

Run the Xcode archive command and then export an .ipa from the archive.

<details>
<summary>Description</summary>


The Step archives your Xcode project by running the `xcodebuild archive` command and then exports the archive into an .ipa file with the `xcodebuild -exportArchive` command. This .ipa file can be shared, installed on test devices, or uploaded to the App Store Connect.

### Configuring the Step

Before you can use the Step, you need code signing files. Certificates must be uploaded to Bitrise while provisioning profiles should be either uploaded or, if using the iOS Auto Provisioning Step, downloaded from the Apple Developer Portal or generated automatically.

To configure the Step:

1. Make sure the **Project (or Workspace) path** input points to the correct location.

   By default, you do not have to change this.
1. Set the correct value to the **Select method for export** input. If you use the **iOS Auto Provision** Step, the value of this input should be the same as the **Distribution type** input of that Step.
1. Make sure the target scheme is a valid, existing Xcode scheme.
1. Optionally, you can define a configuration type to be used (such as Debug or Release) in the **Configuration name** input.

   By default, the selected Xcode scheme determines which configuration will be used. This option overwrites the configuration set in the scheme.
1. If you wish to use a different Developer portal team than the one set in your Xcode project, enter the ID in the **he Developer Portal team to use for this export** input.

### Troubleshooting

If the Step fails, check your code signing files first. Make sure they are the right type for your export method. For example, an `app-store` export method requires an App Store type provisioning profile and a Distribution certificate.

Check **Debug** for additional options to run the Step. The **Additional options for xcodebuild call** input allows you add any flags that the `xcodebuild` command supports.  

Make sure the **Scheme name** and **Configuration name** inputs contain values that actually exist in your Xcode project.

### Useful links

- https://devcenter.bitrise.io/code-signing/ios-code-signing/create-signed-ipa-for-xcode/
- https://devcenter.bitrise.io/code-signing/ios-code-signing/resigning-an-ipa/
- https://devcenter.bitrise.io/deploy/ios-deploy/ios-deploy-index/

### Related Steps

- [Certificate and profile installer](https://www.bitrise.io/integrations/steps/certificate-and-profile-installer)
- [iOS Auto Provision](https://www.bitrise.io/integrations/steps/ios-auto-provision)
- [Deploy to iTunesConnect](https://www.bitrise.io/integrations/steps/deploy-to-itunesconnect-deliver)
</details>

## 🧩 Get started

Add this step directly to your workflow in the [Bitrise Workflow Editor](https://devcenter.bitrise.io/steps-and-workflows/steps-and-workflows-index/).

You can also run this step directly with [Bitrise CLI](https://github.com/bitrise-io/bitrise).

## ⚙️ Configuration

<details>
<summary>Inputs</summary>

| Key | Description | Flags | Default |
| --- | --- | --- | --- |
| `project_path` | A `.xcodeproj` or `.xcworkspace` path. | required | `$BITRISE_PROJECT_PATH` |
| `scheme` | The Scheme to use. | required | `$BITRISE_SCHEME` |
| `configuration` | (optional) The configuration to use. By default, your Scheme defines which configuration (Debug, Release, ...) should be used, but you can overwrite it with this option.  **Make sure that the Configuration you specify actually exists in your Xcode Project**. If it does not (for example, if you have a typo in the value of this input), Xcode will simply use the Configuration specified by the Scheme and will silently ignore this parameter! |  |  |
| `export_method` | `auto-detect` option is **DEPRECATED** - use direct export methods!  Describes how Xcode should export the archive.  If you select `auto-detect`, the step will figure out proper export method based on the provisioning profile embedded into the generated xcodearchive. | required | `auto-detect` |
| `team_id` | The Developer Portal team to use for this export.  Optional, only required if you want to use a different team for distribution, not the one you have set in your Xcode project.  Format example:  - `1MZX23ABCD4` |  |  |
| `compile_bitcode` | For __non-App Store__ exports, should Xcode re-compile the app from bitcode?  | required | `yes` |
| `upload_bitcode` | For __App Store__ exports, should the package include bitcode? | required | `yes` |
| `icloud_container_environment` | If the app is using CloudKit, this configures the "com.apple.developer.icloud-container-environment" entitlement.   Available options vary depending on the type of provisioning profile used, but may include: Development and Production. |  |  |
| `disable_index_while_building` | Could make the build faster by adding `COMPILER_INDEX_STORE_ENABLE=NO` flag to the `xcodebuild` command which will disable the indexing during the build.  Indexing is needed for  * Autocomplete * Ability to quickly jump to definition * Get class and method help by alt clicking.  Which are not needed in CI environment.  **Note:** In Xcode you can turn off the `Index-WhileBuilding` feature  by disabling the `Enable Index-WhileBuilding Functionality` in the `Build Settings`.<br/> In CI environment you can disable it by adding `COMPILER_INDEX_STORE_ENABLE=NO` flag to the `xcodebuild` command. | required | `yes` |
| `cache_level` | Available options: - `none` : Disable caching - `swift_packages` : Cache Swift PM packages added to the Xcode project | required | `swift_packages` |
| `force_team_id` | Used for Xcode version 8 and above.  Force xcodebuild to use the specified Development Team (`DEVELOPMENT_TEAM`).  Format example:  - `1MZX23ABCD4` |  |  |
| `force_code_sign_identity` | Force xcodebuild to use specified Code Signing Identity (`CODE_SIGN_IDENTITY`).  Specify Code Signing Identity as full ID (e.g. `iPhone Developer: Bitrise Bot (VV2J4SV8V4)`) or specify code signing group ( `iPhone Developer` or `iPhone Distribution` ).  You also have to **specify the Identity in the format it's stored in Xcode project settings**, and **not how it's presented in the Xcode.app GUI**! This means that instead of `iOS` (`iOS Distribution/Development`) you have to use `iPhone` (`iPhone Distribution` or `iPhone Development`). **The input is case sensitive**: `iPhone Distribution` works but `iphone distribution` does not! |  |  |
| `force_provisioning_profile_specifier` | Used for Xcode version 8 and above.  Force xcodebuild to use specified Provisioning Profile (`PROVISIONING_PROFILE_SPECIFIER`).  How to get your Provisioning Profile Specifier:  - In Xcode make sure you disabled `Automatically manage signing` on your project's `General` tab - Now you can select your Provisioning Profile Specifier's name as `Provisioning Profile` input value on your project's `General` tab - `force_provisioning_profile_specifier` input value build up by the Team ID and the Provisioning Profile Specifier name, separated with slash character ('/'): `TEAM_ID/PROFILE_SPECIFIER_NAME`  Format example:  - `1MZX23ABCD4/My Provisioning Profile` |  |  |
| `force_provisioning_profile` | Force xcodebuild to use specified Provisioning Profile (`PROVISIONING_PROFILE`).  Use Provisioning Profile's UUID. The profile's name is not accepted by xcodebuild.  How to get your UUID:  - In xcode select your project -> Build Settings -> Code Signing - Select the desired Provisioning Profile, then scroll down in profile list and click on Other... - The popup will show your profile's UUID.  Format example:  - `c5be4123-1234-4f9d-9843-0d9be985a068` |  |  |
| `custom_export_options_plist_content` | Used for Xcode version 7 and above.  Specifies a custom export options plist content that configures archive exporting. If empty, the step generates these options based on provisioning profile, with default values.  Auto generated export options available for export methods:  - app-store - ad-hoc - enterprise - development  If the step doesn't find an export method based on the provisioning profile, the development method will be used.  Call `xcodebuild -help` for available export options. |  |  |
| `artifact_name` | This name will be used as basename for the generated .xcarchive, .ipa and .dSYM.zip files. |  | `${scheme}` |
| `xcodebuild_options` | Options added to the end of the xcodebuild call.  You can use multiple options, separated by a space character. Example: `-xcconfig PATH -verbose` |  |  |
| `workdir` | Working directory of the step. You can leave it empty to leave the working directory unchanged. |  | `$BITRISE_SOURCE_DIR` |
| `output_dir` | This directory will contain the generated .ipa and .dSYM.zip files. | required | `$BITRISE_DEPLOY_DIR` |
| `is_clean_build` |  | required | `no` |
| `output_tool` | If set to `xcpretty`, the xcodebuild output will be prettified by xcpretty.   If set to `xcodebuild`, only the last 20 lines of raw xcodebuild output will be visible in the build log. The build log will always be added as an artifact. | required | `xcpretty` |
| `export_all_dsyms` | If this input is set to `yes` step will collect every dsym (.app dsym and framwork dsyms) in a directory, zip it and export the zipped directory path. Otherwise only .app dsym will be zipped and the zip path exported. | required | `yes` |
| `verbose_log` | Enable verbose logging? | required | `no` |
</details>

<details>
<summary>Outputs</summary>

| Environment Variable | Description |
| --- | --- |
| `BITRISE_IPA_PATH` |  |
| `BITRISE_APP_DIR_PATH` |  |
| `BITRISE_DSYM_DIR_PATH` | This Environment Variable points to the path of the directory which contains the dSYMs files. If `export_all_dsyms` is set to `yes`, the Step will collect every dSYM (app dSYMs and framwork dSYMs). |
| `BITRISE_DSYM_PATH` | This Environment Variable points to the path of the zip file which contains the dSYM files. If `export_all_dsyms` is set to `yes`, the Step will also collect framework dSYMs in addition to app dSYMs. |
| `BITRISE_XCARCHIVE_PATH` |  |
| `BITRISE_XCARCHIVE_ZIP_PATH` | The created .xcarchive.zip file's path. |
| `BITRISE_XCODEBUILD_ARCHIVE_LOG_PATH` | The file path of the raw `xcodebuild archive` command log. The log is placed into the `Output directory path`. |
| `BITRISE_XCODEBUILD_EXPORT_ARCHIVE_LOG_PATH` | The file path of the raw `xcodebuild -exportArchive` command log. The log is placed into the `Output directory path`. |
| `BITRISE_IDEDISTRIBUTION_LOGS_PATH` | Exported when `xcodebuild -exportArchive` command fails. |
</details>

## 🙋 Contributing

We welcome [pull requests](https://github.com/bitrise-steplib/steps-xcode-archive/pulls) and [issues](https://github.com/bitrise-steplib/steps-xcode-archive/issues) against this repository.

For pull requests, work on your changes in a forked repository and use the Bitrise CLI to [run step tests locally](https://devcenter.bitrise.io/bitrise-cli/run-your-first-build/).

**Note:** this step's end-to-end tests (defined in `e2e/bitrise.yml`) are working with secrets which are intentionally not stored in this repo. External contributors won't be able to run those tests. Don't worry, if you open a PR with your contribution, we will help with running tests and make sure that they pass.

Learn more about developing steps:

- [Create your own step](https://devcenter.bitrise.io/contributors/create-your-own-step/)
- [Testing your Step](https://devcenter.bitrise.io/contributors/testing-and-versioning-your-steps/)
