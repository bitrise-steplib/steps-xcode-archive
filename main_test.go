package main

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-xcode/models"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type MockXcodeVersionProvider struct {
	version models.XcodebuildVersionModel
}

func NewMockXcodeVersionProvider(version models.XcodebuildVersionModel) MockXcodeVersionProvider {
	return MockXcodeVersionProvider{
		version: version,
	}
}

func (p MockXcodeVersionProvider) GetXcodeVersion() (models.XcodebuildVersionModel, error) {
	return p.version, nil
}

type MockEnvProvider struct {
	envs map[string]string
}

func NewMockEnvProvider(envs map[string]string) MockEnvProvider {
	return MockEnvProvider{envs: envs}
}

func (p MockEnvProvider) Getenv(key string) string {
	return p.envs[key]
}

func thisStepInputs(t *testing.T) map[string]string {
	_, filename, _, _ := runtime.Caller(1)
	dir := filepath.Dir(filename)
	stepYMLPth := filepath.Join(dir, "step.yml")
	b, err := fileutil.ReadBytesFromFile(stepYMLPth)
	require.NoError(t, err)

	var s struct {
		Inputs []map[string]interface{} `yaml:"inputs"`
	}
	require.NoError(t, yaml.Unmarshal(b, &s))

	inputKeyValues := map[string]string{}
	for _, in := range s.Inputs {
		for k, v := range in {
			if k != "opts" {
				if v == nil {
					inputKeyValues[k] = ""
				} else {
					v, ok := v.(string)
					require.True(t, ok)
					inputKeyValues[k] = v

				}
				break
			}
		}
	}

	return inputKeyValues
}

func override(orig, new map[string]string) map[string]string {
	inputs := map[string]string{}
	for k, v := range orig {
		inputs[k] = v
	}

	for k, v := range new {
		inputs[k] = v
	}

	return inputs
}

func TestXcodeArchiveStep_ProcessInputs(t *testing.T) {
	tests := []struct {
		name                 string
		xcodeVersionProvider xcodeVersionProvider
		envs                 map[string]string
		want                 Config
		err                  string
	}{
		{
			name:                 "project_path should be and .xcodeproj or .xcworkspace path",
			xcodeVersionProvider: NewMockXcodeVersionProvider(models.XcodebuildVersionModel{MajorVersion: 11}),
			envs: override(thisStepInputs(t), map[string]string{
				"project_path": ".",
				"scheme":       "My Scheme",
				"workdir":      "",
			}),
			want: Config{},
			err:  "issue with input ProjectPath: should be and .xcodeproj or .xcworkspace path",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := XcodeArchiveStep{
				xcodeVersionProvider: tt.xcodeVersionProvider,
				stepInputParser:      stepconf.NewEnvParser(NewMockEnvProvider(tt.envs)),
			}

			config, err := s.ProcessInputs()
			gotErr := (err != nil)
			wantErr := (tt.err != "")
			require.Equal(t, wantErr, gotErr, fmt.Sprintf("Step.ValidateConfig() error = %v, wantErr %v", err, tt.err))
			require.Equal(t, tt.want, config)
		})
	}
}
