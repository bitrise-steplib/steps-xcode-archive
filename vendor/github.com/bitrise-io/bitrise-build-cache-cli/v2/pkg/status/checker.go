// Package status provides a public API for querying which build cache features
// are currently active on this machine. The signals are derived from the same
// on-disk artifacts the activate commands produce, so `status` stays consistent
// with `bitrise-build-cache <feature> activate` without a separate state store.
package status

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/bitrise-io/go-utils/v2/log"

	ccacheconfig "github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/config/ccache"
	rnconfig "github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/config/reactnative"
	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/config/xcelerate"
	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/utils"
)

// Feature names accepted by IsEnabled and the `--feature` flag.
// Bazel activation has no reliable on-disk marker yet, so it's intentionally
// absent from this set — `status --feature=bazel` returns ErrUnknownFeature
// (exit 2) rather than silently lying about disabled state.
const (
	FeatureGradle      = "gradle"
	FeatureXcode       = "xcode"
	FeatureCpp         = "cpp"
	FeatureReactNative = "react-native"
)

// ErrUnknownFeature is returned by IsEnabled when the feature name is not
// recognised. The exit-code layer translates this into a status-2 exit.
var ErrUnknownFeature = errors.New("unknown feature")

// Status is the machine-readable shape returned by Checker.Status and the
// `--json` output of the cobra command.
type Status struct {
	Gradle      bool `json:"gradle"`
	Xcode       bool `json:"xcode"`
	Cpp         bool `json:"cpp"`
	ReactNative bool `json:"reactNative"`
}

// CheckerParams holds the dependencies for a Checker.
type CheckerParams struct {
	Logger         log.Logger
	OsProxy        utils.OsProxy
	DecoderFactory utils.DecoderFactory
}

// Checker inspects the filesystem for cache-feature activation signals.
type Checker struct {
	logger         log.Logger
	osProxy        utils.OsProxy
	decoderFactory utils.DecoderFactory
}

// NewChecker creates a Checker, filling in production defaults for any nil
// fields on params.
func NewChecker(p CheckerParams) *Checker {
	logger := p.Logger
	if logger == nil {
		logger = log.NewLogger()
	}

	osProxy := p.OsProxy
	if osProxy == nil {
		osProxy = utils.DefaultOsProxy{}
	}

	decoderFactory := p.DecoderFactory
	if decoderFactory == nil {
		decoderFactory = utils.DefaultDecoderFactory{}
	}

	return &Checker{
		logger:         logger,
		osProxy:        osProxy,
		decoderFactory: decoderFactory,
	}
}

// Status reports the current enablement of every known build cache feature.
// Missing files or decode errors are interpreted as "disabled" — we never
// return an error here.
func (c *Checker) Status() Status {
	return Status{
		Gradle:      c.gradleEnabled(),
		Xcode:       c.xcodeEnabled(),
		Cpp:         c.cppEnabled(),
		ReactNative: c.reactNativeEnabled(),
	}
}

// IsEnabled returns the enablement of a single feature by name.
// Returns ErrUnknownFeature for unsupported names.
func (c *Checker) IsEnabled(feature string) (bool, error) {
	switch feature {
	case FeatureGradle:
		return c.gradleEnabled(), nil
	case FeatureXcode:
		return c.xcodeEnabled(), nil
	case FeatureCpp:
		return c.cppEnabled(), nil
	case FeatureReactNative:
		return c.reactNativeEnabled(), nil
	default:
		return false, fmt.Errorf("%w: %q", ErrUnknownFeature, feature)
	}
}

// ---------------------------------------------------------------------------
// Private — per-feature detection
// ---------------------------------------------------------------------------

func (c *Checker) gradleEnabled() bool {
	home, err := c.osProxy.UserHomeDir()
	if err != nil {
		return false
	}

	initFile := filepath.Join(home, ".gradle", "init.d", "bitrise-build-cache.init.gradle.kts")
	if _, err := c.osProxy.Stat(initFile); err != nil {
		return false
	}

	return true
}

func (c *Checker) xcodeEnabled() bool {
	cfg, err := xcelerate.ReadConfig(c.osProxy, c.decoderFactory)
	if err != nil {
		return false
	}

	return cfg.BuildCacheEnabled
}

func (c *Checker) cppEnabled() bool {
	cfg, err := ccacheconfig.ReadConfig(c.osProxy, c.decoderFactory)
	if err != nil {
		return false
	}

	return cfg.Enabled
}

func (c *Checker) reactNativeEnabled() bool {
	cfg, err := rnconfig.ReadConfig(c.osProxy, c.decoderFactory)
	if err != nil {
		return false
	}

	return cfg.Enabled
}
