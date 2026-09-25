package xcodecommand

import (
	"fmt"
	"slices"
)

// merge lays the user's options over the derived ones; the user's come last.
//
// A repeated value option is kept and reported (xcodebuild refuses it). A spec default,
// a build setting (last wins in xcodebuild) or a switch (repeat is a no-op) yields to the
// user's, an identical repeat is redundant. Appendable keys keep both. Actions never merge.
func merge(derived, user Options, spec actionSpec) (Options, []Diagnostic) {
	userByKey := map[string]Options{}
	for _, o := range user {
		if o.Kind != Action && o.Kind != Unknown {
			userByKey[o.Key()] = append(userByKey[o.Key()], o)
		}
	}

	var merged Options
	var diagnostics []Diagnostic
	report := func(kind DiagnosticKind, format string, args ...any) {
		diagnostics = append(diagnostics, Diagnostic{Kind: kind, Message: fmt.Sprintf(format, args...)})
	}

	for _, o := range derived {
		conflicting, ok := userByKey[o.Key()]
		yields := spec.defaults[o.Key()] || o.Kind == Switch || o.Kind == BuildSetting
		switch {
		case o.Kind == Action || !ok || spec.appendable[o.Key()]:
			merged = append(merged, o)
		case yields && slices.Equal(conflicting.Args(), o.render()):
			report(RedundantOption, "%q is already set by %s; it can be removed from the additional options", o, spec.name)
		case yields:
			report(Override, "%q replaced by additional option %s", o, conflicting)
		default:
			merged = append(merged, o)
			report(RepeatedOption, "%q is set by %s and again as additional option %s; xcodebuild refuses a repeated option", o, spec.name, conflicting)
		}
	}

	return append(merged, user...), diagnostics
}
