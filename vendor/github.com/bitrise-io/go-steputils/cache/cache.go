package cache

import (
	"os"
	"strings"

	"github.com/bitrise-io/go-steputils/tools"
)

// CacheIncludePathsEnvKey ...
const CacheIncludePathsEnvKey = "BITRISE_CACHE_INCLUDE_PATHS"

// CacheExcludePathsEnvKey ...
const CacheExcludePathsEnvKey = "BITRISE_CACHE_EXCLUDE_PATHS"

// VariableSetter ...
type VariableSetter interface {
	Set(key, value string) error
}

// OSVariableSetter ...
type OSVariableSetter struct{}

// NewOSVariableSetter ...
func NewOSVariableSetter() VariableSetter {
	return OSVariableSetter{}
}

// Set ...
func (e OSVariableSetter) Set(key, value string) error {
	return os.Setenv(key, value)
}

// EnvmanVariableSetter ...
type EnvmanVariableSetter struct {
}

// NewEnvmanVariableSetter ...
func NewEnvmanVariableSetter() VariableSetter {
	return EnvmanVariableSetter{}
}

// Set ...
func (e EnvmanVariableSetter) Set(key, value string) error {
	return tools.ExportEnvironmentWithEnvman(key, value)
}

// VariableGetter ...
type VariableGetter interface {
	Get(key string) (string, error)
}

// OSVariableGetter ...
type OSVariableGetter struct{}

// NewOSVariableGetter ...
func NewOSVariableGetter() VariableGetter {
	return OSVariableGetter{}
}

// Get ...
func (e OSVariableGetter) Get(key string) (string, error) {
	return os.Getenv(key), nil
}

// Cache ...
type Cache struct {
	variableGetter  VariableGetter
	variableSetters []VariableSetter

	include []string
	exclude []string
}

// Config ...
type Config struct {
	VariableGetter  VariableGetter
	VariableSetters []VariableSetter
}

// NewCache ...
func (c Config) NewCache() Cache {
	return Cache{variableGetter: c.VariableGetter, variableSetters: c.VariableSetters}
}

// New ...
func New() Cache {
	defaultConfig := Config{NewOSVariableGetter(), []VariableSetter{NewOSVariableSetter(), NewEnvmanVariableSetter()}}
	return defaultConfig.NewCache()
}

// IncludePath ...
func (cache *Cache) IncludePath(item ...string) {
	cache.include = append(cache.include, item...)
}

// ExcludePath ...
func (cache *Cache) ExcludePath(item ...string) {
	cache.exclude = append(cache.exclude, item...)
}

// Commit ...
func (cache *Cache) Commit() error {
	commitCachePath := func(key string, values []string) error {
		content, err := cache.variableGetter.Get(key)
		if err != nil {
			return err
		}

		if content != "" {
			content += "\n"
		}

		content += strings.Join(values, "\n")
		content += "\n"

		for _, setter := range cache.variableSetters {
			if err := setter.Set(key, content); err != nil {
				return err
			}
		}
		return nil
	}

	if err := commitCachePath(CacheIncludePathsEnvKey, cache.include); err != nil {
		return err
	}

	if err := commitCachePath(CacheExcludePathsEnvKey, cache.exclude); err != nil {
		return err
	}
	return nil
}
