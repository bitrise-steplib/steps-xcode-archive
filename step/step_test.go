package step

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/bitrise-io/go-utils/v2/log"

	"github.com/bitrise-io/go-utils/fileutil"
	"gopkg.in/yaml.v3"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-xcode/models"
	"github.com/stretchr/testify/require"
)

func TestXcodeArchiveStep_ProcessInputs(t *testing.T) {
	tests := []struct {
		name string
		envs map[string]string
		want Config
		err  string
	}{
		{
			name: "project_path should be and .xcodeproj or .xcworkspace path",
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
			envRepository := MockEnvRepository{envs: tt.envs}
			s := XcodebuildArchiver{
				xcodeVersionProvider: NewMockXcodeVersionProvider(models.XcodebuildVersionModel{MajorVersion: 11}),
				stepInputParser:      stepconf.NewInputParser(envRepository),
				logger:               log.NewLogger(),
			}

			config, err := s.ProcessInputs()
			gotErr := err != nil
			wantErr := tt.err != ""
			require.Equal(t, wantErr, gotErr, fmt.Sprintf("Step.ValidateConfig() error = %v, wantErr %v", err, tt.err))
			require.Equal(t, tt.want, config)
		})
	}
}

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

func thisStepInputs(t *testing.T) map[string]string {
	_, filename, _, _ := runtime.Caller(1)
	thisPackageDir := filepath.Dir(filename)
	rootDir := filepath.Dir(thisPackageDir)
	stepYMLPth := filepath.Join(rootDir, "step.yml")
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

type MockEnvRepository struct {
	envs map[string]string
}

func (r MockEnvRepository) List() []string {
	var keyValuePairs []string
	for key, value := range r.envs {
		keyValuePairs = append(keyValuePairs, key+"="+value)
	}
	return keyValuePairs
}

func (r MockEnvRepository) Unset(key string) error {
	delete(r.envs, key)
	return nil
}

func (r MockEnvRepository) Set(key, value string) error {
	r.envs[key] = value
	return nil
}

func (r MockEnvRepository) Get(key string) string {
	return r.envs[key]
}
