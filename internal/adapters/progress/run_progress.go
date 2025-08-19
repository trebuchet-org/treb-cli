package progress

import (
	"context"

	"github.com/trebuchet-org/treb-cli/internal/cli/render"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

type RunProgress struct {
	renderer *render.ScriptRenderer
	spinner  *SpinnerProgressReporter
}

func NewRunProgress(renderer *render.ScriptRenderer) *RunProgress {
	return &RunProgress{
		renderer: renderer,
		spinner:  NewSpinnerProgressReporter(),
	}
}

// OnProgress does nothing with progress events
func (n *RunProgress) OnProgress(ctx context.Context, event usecase.ProgressEvent) {
	if event.Stage == string(usecase.StageSimulating) {
		if config, ok := event.Metadata.(*usecase.ScriptExecutionConfig); ok {
			n.renderer.PrintDeploymentBanner(config)
		} else {
			n.spinner.Info("Warning: wrong data-type in execution config")
		}
	}

	n.spinner.OnProgress(ctx, event)
}

// Info does nothing with info messages
func (n *RunProgress) Info(message string) {
	n.spinner.Info(message)
}

// Error does nothing with error messages
func (n *RunProgress) Error(message string) {
	n.spinner.Error(message)
}

// Ensure RunProgress implements ProgressSink
var _ usecase.ProgressSink = (*RunProgress)(nil)
