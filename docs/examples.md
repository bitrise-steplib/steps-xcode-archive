### Examples

Build a development IPA with custom xcconfig content:
```yaml
- xcode-archive:
    inputs:
    - project_path: ./_tmp/ios-simple-objc/ios-simple-objc.xcodeproj
    - scheme: ios-simple-objc
    - distribution_method: development
    - xcconfig_content: |
        CODE_SIGN_IDENTITY = Apple Development
```

Build a development IPA with custom xcconfig file path:
```yaml
- xcode-archive:
    inputs:
    - project_path: ./ios-sample/ios-sample.xcodeproj
    - scheme: ios-sample
    - distribution_method: development
    - xcconfig_content: ./ios-sample/Configurations/development.xcconfig
```