package stepconf

import "github.com/bitrise-io/go-utils/v2/env"

// InputParser ...
type InputParser interface {
	Parse(input interface{}) error
}

type inputParser struct {
	envRepository env.Repository
}

// NewInputParser ...
func NewInputParser(envRepository env.Repository) InputParser {
	return inputParser{
		envRepository: envRepository,
	}
}

// Parse ...
func (p inputParser) Parse(input interface{}) error {
	return parse(input, p.envRepository)
}
