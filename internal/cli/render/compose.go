package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ComposeRenderer handles rendering of orchestration results
type ComposeRenderer struct {
	out io.Writer
}

// NewComposeRenderer creates a new orchestrate renderer
func NewComposeRenderer(out io.Writer) *ComposeRenderer {
	return &ComposeRenderer{
		out: out,
	}
}

// GetWriter returns the io.Writer used by this renderer
func (r *ComposeRenderer) GetWriter() io.Writer {
	return r.out
}

// RenderComposeResult renders the result of orchestration
func (r *ComposeRenderer) RenderComposeResult(result *usecase.ComposeResult) error {
	// Only show summary since plan and results are rendered in real-time
	r.renderSummary(result)
	return nil
}

// RenderExecutionPlan displays the execution plan
func (r *ComposeRenderer) RenderExecutionPlan(plan *usecase.ExecutionPlan) {
	fmt.Fprintf(r.out, "\n🎯 Orchestrating %s\n", plan.Group)
	fmt.Fprintf(r.out, "📋 Execution plan: %d components\n\n", len(plan.Components))

	color.New(color.Bold).Fprintf(r.out, "📋 Execution Plan:\n")
	fmt.Fprintf(r.out, "%s\n", strings.Repeat("─", 50))

	for i, step := range plan.Components {
		// Show step number and component name
		fmt.Fprintf(r.out, "%d. ", i+1)
		color.New(color.FgCyan).Fprintf(r.out, "%s", step.Name)

		// Show script
		fmt.Fprintf(r.out, " → ")
		color.New(color.FgGreen).Fprintf(r.out, "%s", step.Script)

		// Show dependencies if any
		if len(step.Dependencies) > 0 {
			color.New(color.FgHiBlack).Fprintf(r.out, " (depends on: %v)", step.Dependencies)
		}

		// Show environment variables if any
		if len(step.Env) > 0 {
			fmt.Fprintln(r.out)
			color.New(color.FgYellow).Fprintf(r.out, "   Env: %v", step.Env)
		}

		fmt.Fprintln(r.out)
	}

	fmt.Fprintln(r.out)
}

// renderExecutionResults displays the results of each step
func (r *ComposeRenderer) renderExecutionResults(result *usecase.ComposeResult) {
	totalSteps := len(result.Plan.Components)

	for i, stepResult := range result.ExecutedSteps {
		stepNum := i + 1

		// Display step header
		color.New(color.Bold).Fprintf(r.out, "[%d/%d] %s\n",
			stepNum, totalSteps, stepResult.Step.Name)
		fmt.Fprintf(r.out, "%s\n", strings.Repeat("─", 70))

		if stepResult.Error != nil {
			// Show error
			color.New(color.FgRed).Fprintf(r.out, "❌ Failed: %v\n", stepResult.Error)
		} else if stepResult.RunResult != nil {
			// Show success
			if stepResult.RunResult.Success {
				color.New(color.FgGreen).Fprintln(r.out, "✓ Completed successfully")

				// Show deployments if any
				if stepResult.RunResult.Changeset != nil {
					deployments := stepResult.RunResult.Changeset.Create.Deployments
					if len(deployments) > 0 {
						fmt.Fprintf(r.out, "  Deployments:\n")
						for _, dep := range deployments {
							fmt.Fprintf(r.out, "    • %s at %s\n", dep.ContractName, dep.Address)
						}
					}
				}
			} else {
				color.New(color.FgRed).Fprintln(r.out, "❌ Failed")
				if stepResult.RunResult.Error != nil {
					fmt.Fprintf(r.out, "  Error: %v\n", stepResult.RunResult.Error)
				}
			}
		}

		fmt.Fprintln(r.out)
	}
}

// RenderStepResult renders a single step result
func (r *ComposeRenderer) RenderStepResult(stepResult *usecase.StepResult) {
	if stepResult.Error != nil {
		// Show error
		color.New(color.FgRed).Fprintf(r.out, "❌ Failed: %v\n", stepResult.Error)
	} else if stepResult.RunResult != nil && stepResult.RunResult.Success {
		// Show success with basic info
		color.New(color.FgGreen).Fprintln(r.out, "✓ Step completed successfully")

		// Count deployments if any
		if stepResult.RunResult.Changeset != nil {
			deployments := stepResult.RunResult.Changeset.Create.Deployments
			if len(deployments) > 0 {
				fmt.Fprintf(r.out, "  Created %d deployment(s)\n", len(deployments))
			}
		}
	}
}

// renderSummary displays the final summary
func (r *ComposeRenderer) renderSummary(result *usecase.ComposeResult) {
	fmt.Fprintf(r.out, "%s\n", strings.Repeat("═", 70))

	if result.Success {
		color.New(color.FgGreen, color.Bold).Fprintf(r.out,
			"🎉 Successfully orchestrated %s deployment\n", result.Plan.Group)

		fmt.Fprintf(r.out, "\n📊 Summary:\n")
		fmt.Fprintf(r.out, "  • Steps executed: %d/%d\n",
			len(result.ExecutedSteps), len(result.Plan.Components))
		fmt.Fprintf(r.out, "  • Total deployments: %d\n", result.TotalDeployments)
	} else {
		color.New(color.FgRed, color.Bold).Fprintf(r.out,
			"❌ Orchestration failed\n")

		if result.FailedStep != nil {
			fmt.Fprintf(r.out, "\n📊 Summary:\n")
			fmt.Fprintf(r.out, "  • Failed at step: %s\n", result.FailedStep.Step.Name)
			fmt.Fprintf(r.out, "  • Steps completed: %d/%d\n",
				len(result.ExecutedSteps)-1, len(result.Plan.Components))

			if result.FailedStep.Error != nil {
				fmt.Fprintf(r.out, "  • Error: %v\n", result.FailedStep.Error)
			}
		}
	}
}
