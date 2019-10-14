package cache

import "github.com/bitrise-io/go-steputils/tools"
import "os"
import "strings"

// GlobalCachePathsEnvironmentKey ...
const GlobalCachePathsEnvironmentKey = "BITRISE_CACHE_INCLUDE_PATHS"

// GlobalCacheIgnorePathsEnvironmentKey ...
const GlobalCacheIgnorePathsEnvironmentKey = "BITRISE_CACHE_EXCLUDE_PATHS"

// Cache ...
type Cache struct {
	include []string
	exclude []string
}

// New ...
func New() Cache {
	return Cache{}
}

// IncludePath ...
func (cache *Cache) IncludePath(item string) {
	cache.include = append(cache.include, item)
}

// ExcludePath ...
func (cache *Cache) ExcludePath(item string) {
	cache.exclude = append(cache.exclude, item)
}

// Commit ...
func (cache *Cache) Commit() error {
	err := appendCacheItem(cache.include)
	if err != nil {
		return err
	}
	return appendCacheIgnoreItem(cache.exclude)
}

func appendCacheItem(values []string) error {
	return combineEnvContent(GlobalCachePathsEnvironmentKey, values)
}

func appendCacheIgnoreItem(values []string) error {
	return combineEnvContent(GlobalCacheIgnorePathsEnvironmentKey, values)
}

func combineEnvContent(envVar string, values []string) error {
	content := os.Getenv(envVar)

	content += "\n" + strings.Join(values, "\n") + "\n"

	// Set envirmonet varible so that an other cache usage does not override
	if err := os.Setenv(envVar, content); err != nil {
		return err
	}

	return tools.ExportEnvironmentWithEnvman(envVar, content)
}
