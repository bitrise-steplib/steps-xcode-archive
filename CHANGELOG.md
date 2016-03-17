## Changelog (Current version: 1.7.0)

-----------------

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

Updated: 2016 Mar 17