## Changelog (Current version: 2.3.7)

-----------------

### 2.3.7 (2017 Oct 26)

* [b9bc10a] prepare for 2.3.7
* [f1de1f7] dep update (#92)

### 2.3.6 (2017 Oct 20)

* [0a46556] prepare for 2.3.6
* [e8f361f] filter for capabilities (#90)
* [fa1400f] Dep (#88)

### 2.3.5 (2017 Oct 11)

* [624321d] prepare for 2.3.5
* [340f3f6] strip custom export options, fixed scheme - targets mapping, fixed xc… (#85)
* [e958690] Trim CustomExportOptionsPlistContent (#84)

### 2.3.4 (2017 Oct 09)

* [68da17e] Prepare for 2.3.4
* [cc78dcd] Default profile priority (#83)

### 2.3.3 (2017 Oct 06)

* [f39eb06] prepare for 2.3.3
* [2d7b421] Resolve profiles (#82)

### 2.3.2 (2017 Sep 29)

* [69dceea] prepare for 2.3.2
* [6b42c29] Xcode9 (#81)

### 2.3.1 (2017 Sep 21)

* [3e843e1] prepare for 2.3.1
* [9dbe78a] export option provisioningProfiles node fix (#79)
* [04ae6f2] Update main.go (#78)
* [0c68ced] Spellcheck: "searching" instead of "seraching" (#77)

### 2.3.0 (2017 Sep 15)

* [c2d9712] prepare for 2.3.0
* [829d9de] Export options (#75)
* [9591b13] force_team_id, force_code_sign_identity, force_provisioning_profile_specifier and force_provisioning_profile are not deprecated options, in some cases they are required! (#74)

### 2.2.1 (2017 Sep 04)

* [70a6614] prepare for 2.2.1
* [bdd56b8] exportOptions fix (#72)

### 2.2.0 (2017 Aug 30)

* [83e1273] prepare for 2.2.0
* [afd465d] Xcode9 export (#70)

### 2.1.1 (2017 Aug 14)

* [c5316cc] prepare for 2.1.1
* [53e50f7] react-native project_type_tag added (#69)

### 2.1.0 (2017 Jul 24)

* [62a0064] prepare for 2.1.0
* [846199d] optional dsyms, godeps-update (#68)

### 2.0.6 (2017 Jun 08)

* [ff1289e] prepare for 2.0.6
* [28f2c71] input grouping and reordering (#64)
* [d8ea1e6] godeps update (#65)

### 2.0.5 (2017 Feb 08)

* [d25cda5] Prepare for 2.0.5
* [325feed] Godeps-update, print DistributionLog (#62)

### 2.0.4 (2016 Dec 06)

* [a881d37] prepare for 2.0.4
* [1808ef8] rvm fix (#60)

### 2.0.3 (2016 Nov 29)

* [e8c26d0] prepare for 2.0.3
* [0cbd8b2] export all dsyms by default (#59)

### 2.0.2 (2016 Nov 23)

* [985d991] prepare for 2.0.2
* [34d32b5] use go-xcode package (#57)
* [7fa9e7d] url fix / update (#58)
* [627ae81] typo fixes

### 2.0.1 (2016 Nov 18)

* [7e82157] prepare for 2.0.1
* [960741a] fixed xcodebuild force options, ensure xcpretty close (#56)

### 2.0.0 (2016 Nov 18)

* [db93fd5] prepare for 2.0.0
* [6e8fe18] Go toolkit (#55)
* [f341f67] one more env to unset, to work around Xcode's bug (#52)

### 1.10.1 (2016 Oct 11)

* [b1cb39f] prepare for 1.10.1
* [74c06eb] custom export options (#51)

### 1.10.0 (2016 Sep 23)

* [278cf86] prepare for 1.10.0
* [fabf268] Xcodebuild output (#50)

### 1.9.2 (2016 Sep 19)

* [fead6c2] prepare for 1.9.2
* [3992b09] remove template from mktemp (#49)

### 1.9.1 (2016 Sep 06)

* [6b80213] prepare for 1.9.1
* [98d229a] Export options (#48)

### 1.9.0 (2016 Aug 11)

* [cb70d37] prepare for 1.9.0
* [c2de143] export all dsyms (#46)
* [2cd20c3] go get releaseman - no need for verbose

### 1.8.5 (2016 Jul 26)

* [c4f9425] release prep - v1.8.5
* [1742332] step.yml revision

### 1.8.4 (2016 Jul 26)

* [70fde9c] bitrise.yml revision & v1.8.4
* [c4ab5f9] Merge pull request #45 from bitrise-io/feature/debug-print-paths
* [e85a01c] debug print the exposed paths

### 1.8.3 (2016 Jul 05)

* [0640cdf] prepare for 1.8.3
* [278f32d] Merge pull request #42 from bitrise-io/validate_archive
* [e92de96] check for generated xcarchive

### 1.8.2 (2016 Jul 01)

* [1199337] prepare for 1.8.2
* [60664e5] Merge pull request #41 from bitrise-io/viktorbenei-patch-1
* [0a90888] move env unset

### 1.8.1 (2016 Jun 23)

* [91ee69d] prepare for 1.8.1
* [e958fd8] Merge pull request #40 from bitrise-io/deprecated
* [e4a033c] typo fix
* [052df29] allow to use deprecated export method: `use_deprecated_export `
* [f107700] added more info about the format of the Identity
* [65bbb90] step.yml update

### 1.8.0 (2016 Jun 06)

* [c7b1510] prepare for 1.8.0
* [9ab1165] Merge pull request #36 from bitrise-io/force_code_sign_fix
* [ab9f0cc] PR fix
* [9aebd3a] PR update
* [d5ccfb2] force code sign updates

### 1.7.3 (2016 May 20)

* [9b50305] v1.7.3
* [3f641bd] Merge branch 'master' of github.com:bitrise-io/steps-xcode-archive
* [7c48a3b] STEP_GIT_VERION_TAG_TO_SHARE: 1.7.3
* [ab91731] Merge pull request #35 from bitrise-io/gem_home_fix
* [fd775d0] gem_home fix

### 1.7.2 (2016 May 05)

* [ffd6d65] prepare for release
* [fdabae8] Merge pull request #34 from bitrise-io/export_app_dir
* [085e6ab] PR fix
* [0965b76] print xcodebuild full version, export .app directory

### 1.7.1 (2016 Mar 31)

* [df5e98d] prepare for release
* [872f4c7] Merge pull request #31 from bitrise-io/typo
* [461097a] typo fix

### 1.7.0 (2016 Mar 17)

* [532ef37] release config fix
* [104fa9d] version bump
* [dadd071] Merge pull request #29 from bitrise-io/persist_xcarchive
* [01ae31b] persist xcarchive
* [2e5f92b] Merge pull request #28 from bitrise-io/step_review
* [2945620] ensure clean git before share
* [a17a46d] subshell fix, step.yml fix
* [7e38887] bitrise.yml updates
* [aa396ca] workdir fix
* [98d5ebc] step review, log inprovements

### 1.6.1 (2016 Feb 23)

* [df8379c] xcodebuild additional options
* [a917a21] bitrise.yml : MY_STEPLIB_REPO_FORK_GIT_URL fix for Bitrise CLI 1.3.0
* [a587a6d] bitrise.yml : v1.6.0 & step audit

### 1.6.0 (2016 Jan 25)

* [fede0be] is_clean_build : default "no"
* [e49bf4a] bitrise.yml update
* [5ab54f3] white space fix
* [781835e] STEP_GIT_VERION_TAG_TO_SHARE: 1.5.2

### 1.5.2 (2015 Dec 04)

* [206e1a2] printing xcpretty & xcodebuild versions in Configs
* [29a5fa1] removed a space from log
* [4fbfe44] STEP_GIT_VERION_TAG_TO_SHARE: 1.5.1

### 1.5.1 (2015 Dec 04)

* [08e2058] Minor log enhancement
* [5c7db73] output_tool: xcodebuild - added for testing
* [aeb58b9] STEP_GIT_VERION_TAG_TO_SHARE: 1.5.0

### 1.5.0 (2015 Dec 03)

* [769f31f] xcpretty fix
* [4f3ade9] missing xcpretty hint
* [a4be686] xcpretty
* [4dff30e] added `share-this-step` workflow

### 1.4.1 (2015 Nov 06)

* [4a6f035] don't store the .xcarchive in the deploy dir as it's usually significantly larger than any other generated artifact file and it's rarely useful to have as an artifact; no need to deploy it
* [fe1302a] bitrise.yml format version update
* [31d201d] readme

### 1.4.0 (2015 Nov 05)

* [7ec8fa8] added notes to the new `configuration` input's description
* [c2101e2] bit of code cleanup & log revision
* [a50550c] bitrise.yml updated for better testing
* [315e36b] bit of code style and logging change; step.yml updated to use the new `deps` dependency definition
* [3b7fd83] remove old default configuration from step option file
* [5d63a5b] repair indent breaking and add configuration completely optional (It doesn't appear in command if it will be not specified)
* [8ccac59] remove unused env variable
* [35e8c31] make configuration optional and set it to release by default
* [5c21e0c] xctool
* [34256fd] xctool dep
* [1ac177a] add xcode configuration to step configuration
* [115c6b3] build_tool
* [fc2f40c] pre ElCap OS X related fix

### 1.3.6 (2015 Nov 03)

* [4f870c6] ipa_path fix

### 1.3.5 (2015 Nov 03)

* [e494546] ipa export fix
* [7454288] PR fix
* [767d071] ipa path fix
* [8d9d52e] `is_expand: true` for export_options_path
* [758096e] detect_ad_hoch

### 1.3.4 (2015 Oct 29)

* [120c83e] detect ad-hoch distribution
* [57bee7c] less verbose log

### 1.3.3 (2015 Oct 28)

* [7bde2ea] less verbose log

### 1.3.2 (2015 Oct 28)

* [2ddd655] less verbose log
* [1a10ac5] moved the RVM fix right before xcodebuild
* [42f8f35] rvm fix
* [e67d8df] plist gen & save to file change
* [4c5de9b] a bit more debug log
* [704e8b1] debug logging
* [65a6018] Gemfile
* [326d7ff] fix: in case the archive path contains whitespace / space

### 1.3.1 (2015 Oct 28)

* [81a9892] is_clean_build option added + minimal configs log format revision/improvement

### 1.3.0 (2015 Oct 28)

* [47a2f63] archive_fix
* [a4aa514] Update step.yml
* [96ab8bf] force code sign mode

### 1.2.0 (2015 Sep 14)

* [15c1f1c] dSYM debug log typo fix

### 1.1.1 (2015 Sep 11)

* [97bc3ed] minimal bitrise.yml and step.yml update

### 1.1.0 (2015 Sep 10)

* [051c2d0] fix
* [b68ddce] output dir
* [f2ce479] project_path invisible char fix!!

### 1.0.1 (2015 Sep 09)

* [e7e9adb] project_path fix

### 1.0.0 (2015 Sep 09)

* [f6850d6] update
* [f890ab9] multiple embedded.mobileprovision detection fix

### 0.9.3 (2015 Sep 03)

* [775a2c5] step.yml title revision

### 0.9.2 (2015 Aug 18)

* [85f8335] revision: IPA_PATH output, step.yml update, full bitrise.yml for testing
* [a774c0f] misspell fix, instead of archive path -> build_path
* [46b757a] fork_url updated to source_code_url
* [1ba9910] initial beta version

-----------------

Updated: 2017 Oct 26