package xcproj

import (
	"bytes"
	"fmt"
	"slices"
	"unicode"

	"github.com/tailscale/hujson"
)

const indentStep = "  "

// Editor changes a project.xcproj in place, so every byte it does not change stays as Xcode
// wrote it.
type Editor struct {
	root           hujson.Value
	configurations []string
}

// NewEditor needs the names of the project's configurations to split a shared key into one key
// per configuration.
func NewEditor(content []byte, configurations []string) (*Editor, error) {
	// The parsed tree aliases its input.
	root, err := hujson.Parse(bytes.Clone(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse project.xcproj: %w", err)
	}

	return &Editor{root: root, configurations: configurations}, nil
}

// SetTargetBuildSetting sets key for one configuration of a target. key may carry conditions
// other than [config=...], such as CODE_SIGN_IDENTITY[sdk=iphoneos*].
//
// A key that applies to every configuration is split into one key per configuration first,
// which is how Xcode records a setting that differs between configurations.
func (e *Editor) SetTargetBuildSetting(targetName, configuration, key, value string) error {
	if !slices.Contains(e.configurations, configuration) {
		return fmt.Errorf("project has no configuration %s", configuration)
	}

	parsed, ok := parseBuildSettingKey(key)
	if !ok || parsed.hasConfigurationCondition() {
		return fmt.Errorf("invalid build setting key: %s", key)
	}

	target, err := e.target(targetName)
	if err != nil {
		return err
	}

	configuredKey := configurationKey(parsed, configuration)
	settings, ok := valueOf[*hujson.Object](memberValue(target, "build-settings"))
	if !ok || len(settings.Members) == 0 {
		addFirstBuildSetting(target, configuredKey, value)
		return nil
	}

	if existing := memberValue(settings, configuredKey); existing != nil {
		existing.Value = hujson.String(value)
		return nil
	}

	if memberIndex(settings, key) >= 0 {
		e.splitSharedKey(settings, parsed, configuration, value)
		return nil
	}

	insertMember(settings, configuredKey, hujson.String(value))
	return nil
}

// Content returns the edited project.xcproj.
func (e *Editor) Content() []byte {
	return e.root.Pack()
}

func (e *Editor) target(targetName string) (*hujson.Object, error) {
	targets, ok := valueOf[*hujson.Array](e.root.Find("/targets"))
	if !ok {
		return nil, fmt.Errorf("project has no targets")
	}

	for i := range targets.Elements {
		target, ok := targets.Elements[i].Value.(*hujson.Object)
		if !ok {
			continue
		}
		name, ok := valueOf[hujson.Literal](memberValue(target, "name"))
		if !ok || name.String() != targetName {
			continue
		}
		return target, nil
	}

	return nil, fmt.Errorf("project has no target %s", targetName)
}

func valueOf[T hujson.ValueTrimmed](value *hujson.Value) (T, bool) {
	if value == nil {
		var zero T
		return zero, false
	}
	typed, ok := value.Value.(T)
	return typed, ok
}

func memberValue(object *hujson.Object, name string) *hujson.Value {
	if index := memberIndex(object, name); index >= 0 {
		return &object.Members[index].Value
	}
	return nil
}

func memberIndex(object *hujson.Object, name string) int {
	return slices.IndexFunc(object.Members, func(member hujson.ObjectMember) bool {
		return memberName(member) == name
	})
}

func memberName(member hujson.ObjectMember) string {
	literal, ok := member.Name.Value.(hujson.Literal)
	if !ok {
		return ""
	}
	return literal.String()
}

// configurationKey puts [config=...] right after the setting name, the order Xcode writes
// conditions in.
func configurationKey(key buildSettingKey, configuration string) string {
	key.conditions = append([]string{configConditionPrefix + configuration}, key.conditions...)
	return key.String()
}

// Xcode writes build-settings last among the common target members, so before an external
// build system target's build-tool members.
func addFirstBuildSetting(target *hujson.Object, key, value string) {
	layout := target.Members[len(target.Members)-1]
	indent := lineBreakAndIndent(layout.Name.BeforeExtra)
	settings := &hujson.Object{
		Members: []hujson.ObjectMember{{
			Name: hujson.Value{
				BeforeExtra: append(bytes.Clone(indent), indentStep...),
				Value:       hujson.String(key),
				AfterExtra:  bytes.Clone(layout.Name.AfterExtra),
			},
			Value: hujson.Value{
				BeforeExtra: bytes.Clone(layout.Value.BeforeExtra),
				Value:       hujson.String(value),
				AfterExtra:  hujson.Extra{},
			},
		}},
		AfterExtra: indent,
	}

	if existing := memberValue(target, "build-settings"); existing != nil {
		// A comment inside an empty object is kept in the object's closing whitespace.
		if empty, ok := existing.Value.(*hujson.Object); ok {
			name := &settings.Members[0].Name
			name.BeforeExtra = append(bytes.Clone(bytes.TrimRightFunc(empty.AfterExtra, unicode.IsSpace)), name.BeforeExtra...)
		}
		existing.Value = settings
		return
	}

	added := hujson.ObjectMember{
		Name: hujson.Value{
			BeforeExtra: bytes.Clone(indent),
			Value:       hujson.String("build-settings"),
			AfterExtra:  bytes.Clone(layout.Name.AfterExtra),
		},
		Value: hujson.Value{
			BeforeExtra: bytes.Clone(layout.Value.BeforeExtra),
			Value:       settings,
		},
	}

	index := memberIndex(target, "build-tool-path")
	if index < 0 {
		index = len(target.Members)
		moveTrailingComma(&target.Members[index-1].Value, &added.Value)
	}

	target.Members = slices.Insert(target.Members, index, added)
}

// splitSharedKey replaces the shared key with one key per configuration. Every other
// configuration keeps the shared value, unless it already has its own key.
func (e *Editor) splitSharedKey(settings *hujson.Object, key buildSettingKey, configuration, value string) {
	// Copied, as inserting shifts the members in place.
	shared := settings.Members[memberIndex(settings, key.String())]

	newKeys := []string{configurationKey(key, configuration)}
	for _, other := range e.configurations {
		otherKey := configurationKey(key, other)
		if other != configuration && memberValue(settings, otherKey) == nil {
			insertMember(settings, otherKey, shared.Value.Value)
			newKeys = append(newKeys, otherKey)
		}
	}
	insertMember(settings, newKeys[0], hujson.String(value))
	removeMember(settings, memberIndex(settings, key.String()))

	// A comment in front of the shared key describes the setting, so it moves with it.
	first := slices.IndexFunc(settings.Members, func(member hujson.ObjectMember) bool {
		return slices.Contains(newKeys, memberName(member))
	})
	settings.Members[first].Name.BeforeExtra = shared.Name.BeforeExtra
}

// insertMember keeps the members sorted, as Xcode writes them, and copies the layout of the
// last member, which unlike the first follows a comma. object must not be empty.
func insertMember(object *hujson.Object, name string, value hujson.ValueTrimmed) {
	layout := object.Members[len(object.Members)-1]
	inserted := hujson.ObjectMember{
		Name: hujson.Value{
			BeforeExtra: lineBreakAndIndent(layout.Name.BeforeExtra),
			Value:       hujson.String(name),
			AfterExtra:  bytes.Clone(layout.Name.AfterExtra),
		},
		Value: hujson.Value{
			BeforeExtra: bytes.Clone(layout.Value.BeforeExtra),
			Value:       value,
		},
	}

	index := slices.IndexFunc(object.Members, func(member hujson.ObjectMember) bool {
		return memberName(member) > name
	})
	if index < 0 {
		index = len(object.Members)
		moveTrailingComma(&object.Members[index-1].Value, &inserted.Value)
	}

	object.Members = slices.Insert(object.Members, index, inserted)
}

// lineBreakAndIndent drops any comment in front of a member, so a new member does not repeat it.
// It cuts at the first comment, as a line break after that may be inside the comment.
func lineBreakAndIndent(extra hujson.Extra) hujson.Extra {
	if end := bytes.IndexFunc(extra, func(r rune) bool { return !unicode.IsSpace(r) }); end >= 0 {
		extra = extra[:end]
	}
	if i := bytes.LastIndexByte(extra, '\n'); i >= 0 {
		extra = extra[i:]
	}
	return bytes.Clone(extra)
}

func removeMember(object *hujson.Object, index int) {
	if last := len(object.Members) - 1; index == last && index > 0 {
		moveTrailingComma(&object.Members[index].Value, &object.Members[index-1].Value)
	}
	object.Members = slices.Delete(object.Members, index, index+1)
}

// moveTrailingComma hands the object's trailing comma to a new last member: hujson writes one
// only while the last value's AfterExtra is non-nil.
func moveTrailingComma(from, to *hujson.Value) {
	to.AfterExtra = from.AfterExtra
	if from.AfterExtra != nil {
		from.AfterExtra = []byte{}
	}
}
