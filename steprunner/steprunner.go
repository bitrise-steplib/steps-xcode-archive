package steprunner

import (
	"fmt"
	"github.com/bitrise-io/go-utils/v2/log"
)

type Step[C any, R any] interface { // todo: handle optional functions
	ProcessInputs() (C, error)
	EnsureDependencies(C) error
	Run(C) (R, error)
	ExportOutput(C, R) error
}

type StepRunner[C any, R any] struct {
	logger log.Logger
}

func NewStepRunner[C any, R any](logger log.Logger) StepRunner[C, R] {
	return StepRunner[C, R]{
		logger: logger,
	}
}

func (r StepRunner[C, R]) Run(step Step[C, R]) int {
	config, err := step.ProcessInputs()
	if err != nil {
		r.logger.Errorf(formattedError(fmt.Errorf("processing Step Inputs failed: %w", err)))
		return 1
	}

	if err := step.EnsureDependencies(config); err != nil {
		// todo: add EnsureDependencies failure handler to StepRunner
		//var xcprettyInstallErr step.XCPrettyInstallError
		//if errors.As(err, &xcprettyInstallErr) {
		//	logger.Warnf("Installing xcpretty failed: %s", err)
		//	logger.Warnf("Switching to xcodebuild for log formatter")
		//	config.LogFormatter = "xcodebuild"
		//} else {
		r.logger.Errorf(formattedError(fmt.Errorf("installing Step Dependencies failed: %w", err)))
		return 1
		//}
	}

	exitCode := 0
	result, err := step.Run(config)
	if err != nil {
		r.logger.Errorf(formattedError(fmt.Errorf("step run failed: %w", err)))
		exitCode = 1
		// don't return as step outputs needs to be exported even in case of failure (for example the xcodebuild logs)
	}

	if err := step.ExportOutput(config, result); err != nil {
		r.logger.Errorf(formattedError(fmt.Errorf("exporting Step Outputs failed: %w", err)))
		return 1
	}

	return exitCode
}
