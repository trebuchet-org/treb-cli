package progress

import (
	"context"
	"fmt"

	"github.com/trebuchet-org/treb-cli/internal/cli/render"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ComposeProgress handles progress events for compose/orchestrate operations
type ComposeProgress struct {
	composeRenderer *render.ComposeRenderer
	scriptRenderer  *render.ScriptRenderer
	spinner         *SpinnerProgressReporter

	// Track state for proper rendering
	planRendered bool
	currentStep  string
}

// NewComposeProgress creates a new compose progress reporter
func NewComposeProgress(composeRenderer *render.ComposeRenderer, scriptRenderer *render.ScriptRenderer) *ComposeProgress {
	return &ComposeProgress{
		composeRenderer: composeRenderer,
		scriptRenderer:  scriptRenderer,
		spinner:         NewSpinnerProgressReporter(),
		planRendered:    false,
	}
}

// OnProgress handles progress events for compose operations
func (p *ComposeProgress) OnProgress(ctx context.Context, event usecase.ProgressEvent) {
	switch event.Stage {
	case "plan_created":
		// Render the execution plan once it's created
		if plan, ok := event.Metadata.(*usecase.ExecutionPlan); ok && !p.planRendered {
			p.composeRenderer.RenderExecutionPlan(plan)
			p.planRendered = true
		}

	case "step_starting":
		// Show step header when starting a new step
		if stepInfo, ok := event.Metadata.(map[string]interface{}); ok {
			if stepName, ok := stepInfo["name"].(string); ok {
				p.currentStep = stepName
				totalSteps := stepInfo["total"].(int)
				currentNum := stepInfo["current"].(int)

				// Clear spinner and show step header
				p.spinner.spinner.Stop()
				fmt.Fprintf(p.composeRenderer.GetWriter(), "\n[%d/%d] Starting %s\n",
					currentNum, totalSteps, stepName)
			}
		}

	case string(usecase.StageSimulating):
		// When script starts simulating, show its deployment banner
		if config, ok := event.Metadata.(*usecase.RunScriptConfig); ok {
			p.scriptRenderer.PrintDeploymentBanner(config)
		}

	case "step_completed":
		// Render the step execution result
		if stepResult, ok := event.Metadata.(*usecase.StepResult); ok {
			p.composeRenderer.RenderStepResult(stepResult)

			// If the step has a run result, render the execution summary
			if stepResult.RunResult != nil && stepResult.RunResult.RunResult != nil {
				if err := p.scriptRenderer.RenderExecution(stepResult.RunResult); err != nil {
					panic(err)
				}
			}
		}

	case "compose_completed":
		// Final summary is rendered by the CLI command after this returns
		p.spinner.spinner.Stop()

	default:
		// Pass through to spinner for other progress events
		p.spinner.OnProgress(ctx, event)
	}
}

// Info forwards info messages to the spinner
func (p *ComposeProgress) Info(message string) {
	p.spinner.Info(message)
}

// Error forwards error messages to the spinner
func (p *ComposeProgress) Error(message string) {
	p.spinner.Error(message)
}

// Ensure ComposeProgress implements ProgressSink
var _ usecase.ProgressSink = (*ComposeProgress)(nil)
