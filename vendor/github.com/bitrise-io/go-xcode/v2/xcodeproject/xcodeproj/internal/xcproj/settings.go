package xcproj

import "strings"

const configConditionPrefix = "config="

// buildSettingKey is a build setting key split into its name and its [param=pattern] conditions,
// such as CODE_SIGN_IDENTITY[config=Release][sdk=iphoneos*].
type buildSettingKey struct {
	name string
	// conditions holds the text between the brackets of each condition, in key order, unparsed.
	conditions []string
}

// parseBuildSettingKey splits key into its name and conditions. It reports false for a key that
// is not a name followed by well-formed [...] groups, which Xcode itself ignores.
func parseBuildSettingKey(key string) (buildSettingKey, bool) {
	nameEnd := strings.IndexByte(key, '[')
	if nameEnd < 0 {
		return buildSettingKey{name: key}, key != ""
	}
	if nameEnd == 0 {
		return buildSettingKey{}, false
	}

	parsed := buildSettingKey{name: key[:nameEnd]}
	for rest := key[nameEnd:]; rest != ""; {
		if rest[0] != '[' {
			return buildSettingKey{}, false
		}

		end := strings.IndexByte(rest, ']')
		if end < 0 {
			return buildSettingKey{}, false
		}

		condition := rest[1:end]
		if !isWellFormedCondition(condition) {
			return buildSettingKey{}, false
		}

		parsed.conditions = append(parsed.conditions, condition)
		rest = rest[end+1:]
	}

	return parsed, true
}

// isWellFormedCondition reports whether condition is a nonempty parameter followed by "=" and a
// pattern, which may be empty. A nested "[" is malformed too.
func isWellFormedCondition(condition string) bool {
	return strings.Index(condition, "=") > 0 && !strings.Contains(condition, "[")
}

func (k buildSettingKey) String() string {
	var b strings.Builder
	b.WriteString(k.name)
	for _, condition := range k.conditions {
		b.WriteString("[" + condition + "]")
	}

	return b.String()
}

// configurationCondition returns the configuration the key is restricted to and the key without
// that condition. It reports false unless the key leads with a [config=...] condition, the way
// Xcode writes it.
func (k buildSettingKey) configurationCondition() (configuration string, rest buildSettingKey, ok bool) {
	if len(k.conditions) == 0 {
		return "", buildSettingKey{}, false
	}

	configuration, ok = strings.CutPrefix(k.conditions[0], configConditionPrefix)
	if !ok {
		return "", buildSettingKey{}, false
	}

	return configuration, buildSettingKey{name: k.name, conditions: k.conditions[1:]}, true
}

// hasConfigurationCondition reports whether any condition of the key is a [config=...].
func (k buildSettingKey) hasConfigurationCondition() bool {
	for _, condition := range k.conditions {
		if strings.HasPrefix(condition, configConditionPrefix) {
			return true
		}
	}

	return false
}

// SettingsForConfiguration returns the settings of one configuration from the single
// build-settings map of a project.xcproj, keyed the way pbxproj keys them.
//
// Do not assume an unconditioned key exists: once configurations differ, Xcode writes only the
// conditioned ones.
//
// This is deliberately simpler than how Xcode evaluates conditions. In swift-build, config is an
// fnmatch pattern that can sit in any position (see MacroCondition.swift and
// MacroConfigFileParser.swift in
// https://github.com/swiftlang/swift-build/tree/main/Sources/SWBMacro). Xcode only writes literal
// configuration names, first: its build settings editor nests SDK conditions under a configuration.
// swift-build is not the full picture, as how xcodebuild turns a project.xcproj into
// per-configuration settings is not public.
func SettingsForConfiguration(settings map[string]any, configuration string) map[string]any {
	result := make(map[string]any, len(settings))
	overrides := map[string]any{}

	for key, value := range settings {
		parsed, ok := parseBuildSettingKey(key)
		if !ok {
			result[key] = value
			continue
		}

		keyConfiguration, rest, ok := parsed.configurationCondition()
		if !ok {
			result[key] = value
			continue
		}

		// Xcode never writes a second config condition, and ANDed with the first it usually matches
		// no configuration, so the key is dropped rather than guessed at.
		if rest.hasConfigurationCondition() || keyConfiguration != configuration {
			continue
		}

		overrides[rest.String()] = value
	}

	for key, value := range overrides {
		result[key] = value
	}

	return result
}
