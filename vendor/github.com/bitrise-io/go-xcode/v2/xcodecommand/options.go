package xcodecommand

// Kind classifies one xcodebuild command line argument.
type Kind int

// The argument forms a command derives from its params.
const (
	Switch       Kind = iota // -quiet
	ValueOption              // -destination <spec>
	ColonOption              // -only-testing:<id>
	BuildSetting             // CODE_SIGNING_ALLOWED=NO
	Action                   // clean, build, archive, ...
)

// Option is one xcodebuild argument.
type Option struct {
	Kind  Kind
	Name  string
	Value string
}

// args is the option as xcodebuild arguments: one for a switch, an action or a joined
// form, two for a flag with a value.
func (o Option) args() []string {
	switch o.Kind {
	case ValueOption:
		return []string{o.Name, o.Value}
	case ColonOption:
		return []string{o.Name + ":" + o.Value}
	case BuildSetting:
		return []string{o.Name + "=" + o.Value}
	default:
		return []string{o.Name}
	}
}

// Options is an ordered list of xcodebuild arguments.
type Options []Option

// Args renders the options as xcodebuild arguments.
func (opts Options) Args() []string {
	var args []string
	for _, o := range opts {
		args = append(args, o.args()...)
	}
	return args
}
