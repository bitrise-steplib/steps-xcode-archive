package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/log"
)

// Inputs ...
type Inputs struct {
	Envs          string `env:"envs"`
	Files         string `env:"files"`
	Dirs          string `env:"dirs"`
	DeployDir     string `env:"deploy_dir"`
	DeployedFiles string `env:"deployed_files"`
	DeployedDirs  string `env:"deployed_dirs"`
}

func parseList(s string) map[string]string {
	keys := strings.Split(s, "\n")
	m := map[string]string{}
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k != "" {
			v := os.Getenv(k)
			m[k] = v
		}
	}
	return m
}

func checkEnvs(m map[string]string) error {
	for k, v := range m {
		if v == "" {
			return fmt.Errorf("(%s=%s) is empty", k, v)
		}
	}
	return nil
}

func checkFiles(m map[string]string, isDir bool, deployDir string) error {
	if err := checkEnvs(m); err != nil {
		return err
	}

	for k, v := range m {
		fileInf, err := os.Lstat(v)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("(%s=%s) is not exist", k, v)
			}

			return fmt.Errorf("issue with (%s=%s): %s", k, v, err)
		}

		if isDir && !fileInf.IsDir() {
			return fmt.Errorf("(%s=%s) is not a directory", k, v)
		}

		if !isDir && fileInf.IsDir() {
			return fmt.Errorf("(%s=%s) is not a file", k, v)
		}

		if deployDir != "" {
			if !strings.HasPrefix(v, deployDir) {
				return fmt.Errorf("(%s=%s) is not inside the Deploy Dir (%s)", k, v, deployDir)
			}
		}
	}
	return nil
}

func run() error {
	var inputs Inputs
	if err := stepconf.Parse(&inputs); err != nil {
		return fmt.Errorf("issue with inputs: %s", err)
	}
	stepconf.Print(inputs)
	fmt.Println()

	if err := checkEnvs(parseList(inputs.Envs)); err != nil {
		return err
	}
	if err := checkFiles(parseList(inputs.Files), false, ""); err != nil {
		return err
	}
	if err := checkFiles(parseList(inputs.Dirs), true, ""); err != nil {
		return err
	}
	if err := checkFiles(parseList(inputs.DeployedFiles), false, inputs.DeployDir); err != nil {
		return err
	}
	if err := checkFiles(parseList(inputs.DeployedDirs), true, inputs.DeployDir); err != nil {
		return err
	}

	log.Donef("All Outputs passed")
	return nil
}

func main() {
	if err := run(); err != nil {
		log.Errorf(err.Error())
		os.Exit(1)
	}
}
