# Xcode Archive & Export for iOS

[![Step changelog](https://shields.io/github/v/release/bitrise-steplib/steps-xcode-archive?include_prereleases&label=changelog&color=blueviolet)](https://github.com/bitrise-steplib/steps-xcode-archive/releases)

Run the Xcode archive command and then export an .ipa from the archive.

<details>
<summary>Description</summary>


The Step archives your Xcode project by running the `xcodebuild archive` command and then exports the archive into an .ipa file with the `xcodebuild -exportArchive` command. This .ipa file can be shared, installed on test devices, or uploaded to the App Store Connect.

### Configuring the Step

Before you can use the Step, you need code signing files. Certificates must be uploaded to Bitrise while provisioning profiles should be either uploaded or, if using the iOS Auto Provisioning Step, downloaded from the Apple Developer Portal or generated automatically.

To configure the Step:

1. Make sure the **Project path** input points to the correct location.

   By default, you do not have to change this.
2. Set the correct value to the **Distribution method** input. If you use the **iOS Auto Provision** Step, the value of this input should be the same as the **Distribution type** input of that Step.
3. Make sure the target scheme is a valid, existing Xcode scheme.
4. Optionally, you can define a configuration type to be used (such as Debug or Release) in the **Build configuration** input.

   By default, the selected Xcode scheme determines which configuration will be used. This option overwrites the configuration set in the scheme.
5. If you wish to use a different Developer portal team than the one set in your Xcode project, enter the ID in the **Developer Portal team** input.

### Troubleshooting

If the Step fails, check your code signing files first. Make sure they are the right type for your export method. For example, an `app-store` distribution method requires an App Store type provisioning profile and a Distribution certificate.

Check **Debugging** for additional options to run the Step. The **Additional options for xcodebuild command** input allows you add any flags that the `xcodebuild` command supports.

Make sure the **Scheme** and **Build configuration** inputs contain values that actually exist in your Xcode project.

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
| `project_path` | Xcode Project (`.xcodeproj`) or Workspace (`.xcworkspace`) path.  The input value sets xcodebuild's `-project` or `-workspace` option. | required | `$BITRISE_PROJECT_PATH` |
| `scheme` | Xcode Scheme name.  The input value sets xcodebuild's `-scheme` option. | required | `$BITRISE_SCHEME` |
| `distribution_method` | Describes how Xcode should export the archive. | required | `development` |
| `automatic_code_signing` | This input determines which Bitrise Apple service connection should be used for automatic Code Signing.  Available values: - `off`: Do not manage Code Signing. - `api-key`: Bitrise Apple Service connection with API Key. - `apple-id`: Bitrise Apple Service connection with Apple ID. | required | `off` |
| `configuration` | Xcode Build Configuration.  If not specified, the default Build Configuration will be used.  The input value sets xcodebuild's `-configuration` option. |  |  |
| `xcconfig_content` | Build settings to override the project's build settings.  Build settings must be separated by newline character (`\n`).  Example:  ``` COMPILER_INDEX_STORE_ENABLE = NO ONLY_ACTIVE_ARCH[config=Debug][sdk=*][arch=*] = YES ```  The input value sets xcodebuild's `-xcconfig` option. |  | `COMPILER_INDEX_STORE_ENABLE = NO` |
| `perform_clean_action` | If this input is set, `clean` xcodebuild action will be performed besides the `archive` action. | required | `no` |
| `xcodebuild_options` | Additional options to be added to the executed xcodebuild command. |  |  |
| `log_formatter` | Defines how `xcodebuild` command's log is formatted.  Available options:  - `xcpretty`: The xcodebuild command's output will be prettified by xcpretty. - `xcodebuild`: Only the last 20 lines of raw xcodebuild output will be visible in the build log.  The raw xcodebuild log will be exported in both cases. | required | `xcpretty` |
| `export_development_team` | The Developer Portal team to use for this export  Defaults to the team used to build the archive. |  |  |
| `compile_bitcode` | For __non-App Store__ exports, should Xcode re-compile the app from bitcode? | required | `yes` |
| `upload_bitcode` | For __App Store__ exports, should the package include bitcode? | required | `yes` |
| `icloud_container_environment` | If the app is using CloudKit, this configures the `com.apple.developer.icloud-container-environment` entitlement.  Available options vary depending on the type of provisioning profile used, but may include: `Development` and `Production`. |  |  |
| `export_options_plist_content` | Specifies a plist file content that configures archive exporting.  If not specified, the Step will auto-generate it. |  |  |
| `output_dir` | This directory will contain the generated artifacts. | required | `$BITRISE_DEPLOY_DIR` |
| `export_all_dsyms` | Export additional dSYM files besides the app dSYM file for Frameworks. | required | `yes` |
| `artifact_name` | This name will be used as basename for the generated Xcode Archive, App, IPA and dSYM files.  If not specified, the Product Name (`PRODUCT_NAME`) Build settings value will be used. If Product Name is not specified, the Scheme will be used. |  |  |
| `cache_level` | Defines what cache content should be automatically collected.  Available options:  - `none`: Disable collecting cache content - `swift_packages`: Collect Swift PM packages added to the Xcode project | required | `swift_packages` |
| `verbose_log` | If this input is set, the Step will print additional logs for debugging. | required | `no` |
| `certificate_url_list` | URL of the code signing certificate to download.  Multiple URLs can be specified, separated by a pipe (\|) character.  Local file path can be specified, using the file:// URL scheme. | required, sensitive | `$BITRISE_CERTIFICATE_URL` |
| `passphrase_list` | Passphrases for the provided code signing certificates.  Specify as many passphrases as many Code signing certificate URL provided, separated by a pipe (\|) character. | required, sensitive | `$BITRISE_CERTIFICATE_PASSPHRASE` |
| `keychain_path` | Path to the Keychain where the code signing certificates will be installed. | required | `$HOME/Library/Keychains/login.keychain` |
| `keychain_password` | Password for the provided Keychain. | required, sensitive | `$BITRISE_KEYCHAIN_PASSWORD` |
</details>

<details>
<summary>Outputs</summary>

| Environment Variable | Description |
| --- | --- |
| `BITRISE_IPA_PATH` | Local path of the created .ipa file |
| `BITRISE_APP_DIR_PATH` | Local path of the generated `.app` directory |
| `BITRISE_DSYM_DIR_PATH` | This Environment Variable points to the path of the directory which contains the dSYMs files. If `export_all_dsyms` is set to `yes`, the Step will collect every dSYM (app dSYMs and framwork dSYMs). |
| `BITRISE_DSYM_PATH` | This Environment Variable points to the path of the zip file which contains the dSYM files. If `export_all_dsyms` is set to `yes`, the Step will also collect framework dSYMs in addition to app dSYMs. |
| `BITRISE_XCARCHIVE_PATH` | The created .xcarchive file's path |
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
