## Changelog (Current version: 1.9.1)

-----------------

### 1.9.1 (2017 Oct 06)

* [5bc1697] prepare for 1.9.1
* [3937e5a] GetBundleIdentifier fix
* [4db3664] xcode managed profile fix
* [a16a3bd] log updates
* [a064c5b] revision (#39)

### 1.9.0 (2017 Sep 29)

* [5a5ee02] Prepare for 1.9.0
* [7c0ded2] Cert profile packages (#38)

### 1.8.9 (2017 Sep 27)

* [d37f92d] Prepare for 1.8.9
* [c33f2d3] Expiry date checkings (#35)

### 1.8.8 (2017 Aug 11)

* [21c6851] Prepare for 1.8.8
* [957394d] removed os.Exit(1) when checking expiry date (#34)

### 1.8.7 (2017 Jul 24)

* [f16049e] prepare for 1.8.7
* [08c0468] print output if pem conversion failed (#33)
* [359a3ee] Cert and prov profile info prints (#31)

### 1.8.6 (2017 Jun 12)

* [fe36ec0] prepare for 1.8.6
* [3b3aa85] input grouping and reordering (#32)

### 1.8.5 (2017 Apr 13)

* [c74fb95] Prepare for 1.8.5
* [5b2e0f2] Show part of provisioned devices's ID (#29)

### 1.8.4 (2017 Mar 10)

* [ff48ad7] prepare for 1.8.4
* [75e9c12] test for developer id cert install (#28)
* [79e6f41] godeps update (#27)

### 1.8.3 (2017 Mar 03)

* [f58688f] Prepare for 1.8.3
* [b08a0e1] Keychain error print (#26)

### 1.8.2 (2016 Dec 20)

* [2705ec8] prepare for 1.8.2
* [a47069a] Project type tag (#25)

### 1.8.1 (2016 Oct 28)

* [b2d36e2] prepare for 1.8.1
* [c0f279f] sierra keychain prompt fix (#22)

### 1.8.0 (2016 Oct 25)

* [694e2f0] prepare for 1.8.0
* [bb10316] security command Sierra fix, redact provisioned devices as well (#21)

### 1.7.0 (2016 Sep 15)

* [2b84a29] prepare for 1.7.0
* from now you can specify multiple `certificate_url` separated with pipe character
* from now you can specify multiple `certificate_passphrase` separated with pipe character
* printing installed profile's infos 
* [f267551] Toolkit (#19)

### 1.6.0 (2016 Jul 12)

* [df0f7d1] prepare for 1.6.0
* [2941a37] Merge pull request #18 from bitrise-io/export_certificates
* [46742b9] export certificates fix
* [7bf5067] export user cert
* [0d7cbcb] export certificate infos
* [3cdb6e9] Merge pull request #17 from bitrise-io/log_review
* [d85b0ae] log updates

### 1.5.2 (2016 Jun 07)

* [a4de9ff] prepare for 1.5.2
* [47e3d69] Merge pull request #16 from bitrise-io/input_validation
* [6a61447] cert and profile count validation

### 1.5.1 (2016 Jun 07)

* [c9c9925] prepare for 1.5.1
* [b7abc0b] Merge pull request #15 from bitrise-io/required_fix
* [82264aa] Trim fix
* [6aa0b6d] secure inputs
* [337fdbc] do not require certificate_url & provisioning_profile_url

### 1.5.0 (2016 Jun 01)

* [39c4f21] relelase configs
* [28b168a] Merge pull request #14 from bitrise-io/keychain_list_fix
* [8fc6c09] keychain list fix
* [b0709c1] STEP_GIT_VERION_TAG_TO_SHARE: 1.4.2

### 1.4.2 (2016 Apr 22)

* [2f5b45a] testing BITRISE_PROVISIONING_PROFILE_PATH output in bitrise run test
* [2bf9e04] logging fix/revision - related to "profileCount"
* [de2a0df] Merge pull request #13 from rymir/export-prov-prof-path
* [0d0b8a6] Export BITRISE_PROVISIONING_PROFILE_PATH too

### 1.4.1 (2016 Apr 21)

* [78f3f0d] STEP_GIT_VERION_TAG_TO_SHARE: 1.4.1
* [fd595eb] Merge pull request #12 from olegoid/master
* [3289f63] Add 3rd party mac certificates recognition

### 1.4.0 (2015 Dec 11)

* [5da704a] security list-keychains : verbose removed
* [df8b903] an example fix
* [d63b807] URL inputs: note about file:// scheme
* [598137c] removed unnecessary check
* [c1fbb89] quotation
* [02ed136] MY_STEPLIB_REPO_FORK_GIT_URL: $MY_STEPLIB_REPO_FORK_GIT_URL
* [8b9adab] STEP_GIT_VERION_TAG_TO_SHARE: 1.4.0
* [f9d9a9d] Merge pull request #8 from godrei/golang
* [4e70008] download retry fixes
* [691b017] PR fix
* [3b2a409] PrintFatallnf moved to main
* [a4c6326] security
* [6efeba1] PR fixes
* [ce45ad4] retry download, copy instead of move files
* [963f6d1] removed references
* [33567dd] removed log
* [f4e3bfe] fixed search for imported cert & golang
* [398c89c] share 1.3.0

### 1.3.0 (2015 Nov 23)

* [49a3fc2] grep revision (added -e)
* [812cfed] grep fix
* [9eb8942] logging
* [30aefa5] logging
* [f486d8a] certificate info - Mac specific code added & full available list
* [1561adc] err handling in ProvProfile download
* [8acdc45] tmp dir path revision, extension priority revision
* [70c07f7] Merge pull request #7 from vasarhelyia/master
* [1bcc684] Using provisionprofile extension as fallback for mac support

### 1.2.2 (2015 Nov 06)

* [f61d71f] bitrise.yml revision & added share-this-step workflow
* [ba6c97c] further log revision 2
* [36c0ab1] further log revision
* [7c94bb7] bit of logging revision & do not print the passphrase of the certificate
* [0444ad3] Merge pull request #5 from bazscsa/patch-1
* [9704721] Update step.yml

### 1.2.1 (2015 Sep 24)

* [57f127a] minor logging fixes
* [8982d36] executable file flag removed from bitrise.yml LICENSE and README

### 1.2.0 (2015 Sep 14)

* [38cc056] exporting BITRISE_PROVISIONING_PROFILE_ID and BITRISE_CODE_SIGN_IDENTITY
* [2e578f6] Merge pull request #3 from gkiki90/input_fix
* [b5ec7ef] yml fix

### 1.1.1 (2015 Sep 12)

* [dc2daac] Merge pull request #2 from gkiki90/input_fix
* [99e0a8f] input fix

### 1.1.0 (2015 Sep 08)

* [70d1670] bitrise stack related update - README update
* [643de77] bitrise stack related update - removed old step.yml
* [098b5cc] bitrise stack related update
* [306b00f] Merge pull request #1 from gkiki90/update
* [9b6b2ee] update
* [57bad79] debug log
* [ab222dd] fix: don't fail if keychain already exists, instead we should get the keychain password as an input
* [23d8980] step.yml switch : old is now .yml.old and the new one is just step.yml, as required for the new bitwise-cli tool

-----------------

Updated: 2017 Oct 06