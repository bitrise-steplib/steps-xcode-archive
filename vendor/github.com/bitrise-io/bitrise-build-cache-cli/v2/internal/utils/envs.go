package utils

import (
	"os"
	"strings"
)

func AllEnvs() map[string]string {
	envs := map[string]string{}
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) < 2 {
			continue
		}

		envs[pair[0]] = pair[1]
	}

	return envs
}
