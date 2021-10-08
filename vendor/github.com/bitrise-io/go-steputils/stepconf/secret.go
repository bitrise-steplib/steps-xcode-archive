package stepconf

// Secret variables are not shown in the printed output.
type Secret string

const secret = "*****"

// String implements fmt.Stringer.String.
// When a Secret is printed, it's masking the underlying string with asterisks.
func (s Secret) String() string {
	if s == "" {
		return ""
	}
	return secret
}
