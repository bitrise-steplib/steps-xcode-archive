module github.com/bitrise-steplib/steps-xcode-archive

go 1.16

require (
	github.com/bitrise-io/go-steputils/v2 v2.0.0-alpha.2
	github.com/bitrise-io/go-utils v1.0.2
	github.com/bitrise-io/go-utils/v2 v2.0.0-alpha.7
	github.com/bitrise-io/go-xcode v1.0.6
	github.com/bitrise-io/go-xcode/v2 v2.0.0-alpha.15.0.20220525082156-b43525984dc3
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/stretchr/testify v1.7.1
	golang.org/x/crypto v0.0.0-20220518034528-6f7dac969898 // indirect
	golang.org/x/sys v0.0.0-20220520151302-bc2c85ada10a // indirect
	golang.org/x/term v0.0.0-20220411215600-e5f449aeb171 // indirect
	gopkg.in/yaml.v3 v3.0.0
	howett.net/plist v1.0.0
)

replace (
	github.com/bitrise-io/go-xcode v1.0.6 => github.com/shams-ahmed/go-xcode v1.0.100
	github.com/bitrise-io/go-xcode/v2 v2.0.0-alpha.15.0.20220525082156-b43525984dc3 => github.com/shams-ahmed/go-xcode/v2 v2.0.101
)
