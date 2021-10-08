package stepconf

import "github.com/bitrise-io/go-utils/env"

// InputParser ...
type InputParser interface {
	Parse(input interface{}) error
}

type defaultInputParser struct {
	envRepository env.Repository
}

// NewInputParser ...
func NewInputParser(envRepository env.Repository) InputParser {
	return defaultInputParser{
		envRepository: envRepository,
	}
}

// Parse ...
func (p defaultInputParser) Parse(input interface{}) error {
	return parse(input, p.envRepository)
}
