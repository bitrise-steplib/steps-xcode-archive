package step

// LogFormatterErr is returned when the selected log formatter is not available.
type LogFormatterErr struct{}

func (LogFormatterErr) Error() string {
	return "Selected log formatter is not available."
}
