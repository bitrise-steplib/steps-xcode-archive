### Examples

Build a development IPA:
```yaml
- xcode-archive:
    inputs:
    - project_path: ./ios-sample/ios-sample.xcodeproj
    - scheme: ios-sample
    - distribution_method: development
```

Build a development IPA with custom xcconfig content:
```yaml
- xcode-archive:
    inputs:
    - project_path: ./ios-sample/ios-sample.xcodeproj
    - scheme: ios-sample
    - distribution_method: development
    - xcconfig_content: |
        CODE_SIGN_IDENTITY = Apple Development
```

Build a development IPA with overridden code signing setting:
```yaml
- xcode-archive:
    inputs:
    - project_path: ./ios-sample/ios-sample.xcodeproj
    - scheme: ios-sample
    - distribution_method: development
    - xcconfig_content: |
        DEVELOPMENT_TEAM = XXXXXXXXXX
        CODE_SIGN_IDENTITY = Apple Development
        PROVISIONING_PROFILE_SPECIFIER = 12345678-90ab-cdef-0123-4567890abcde
```

Build a development IPA with custom xcconfig file path:
```yaml
- xcode-archive:
    inputs:
    - project_path: ./ios-sample/ios-sample.xcodeproj
    - scheme: ios-sample
    - distribution_method: development
    - xcconfig_content: ./ios-sample/ios-sample/Configurations/Dev.xcconfig
```