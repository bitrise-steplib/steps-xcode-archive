package xcodecommand

import "fmt"

// DiagnosticKind classifies a finding about a command's additional options.
type DiagnosticKind int

// Diagnostic kinds.
const (
	MalformedOption       DiagnosticKind = iota // unclassifiable argument, passed through; xcodebuild rejects it
	RejectedOption                              // switches xcodebuild's mode or belongs to another action
	ActionInOptions                             // a build action; the command owns its action list
	SuspiciousUserDefault                       // -NAME=value that looks like a build setting; xcodebuild ignores it silently
	RepeatedOption                              // a value option set by the command and again by the user; xcodebuild refuses it
	Override                                    // informational: a default or build setting yielded to the user's
	RedundantOption                             // informational: identical to what the command already sets
)

// Diagnostic is one finding about the additional options, for the step log.
type Diagnostic struct {
	Kind    DiagnosticKind
	Message string
}

func (d Diagnostic) String() string {
	return d.Message
}

// Validation says what a constructor does with diagnostics.
type Validation int

// Validation modes.
const (
	// Warn (default) passes every option through and reports; this matches what the
	// steps did before and keeps working setups that trip a check working.
	Warn Validation = iota
	// Fail returns an error for the first non-informational diagnostic.
	Fail
)

func (k DiagnosticKind) informational() bool {
	return k == Override || k == RedundantOption
}

func firstFailure(diagnostics []Diagnostic) error {
	for _, d := range diagnostics {
		if !d.Kind.informational() {
			return fmt.Errorf("invalid additional option: %s", d.Message)
		}
	}
	return nil
}
