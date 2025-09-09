package progress

import (
	"context"
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// SpinnerProgressReporter implements progress reporting with a spinner
type SpinnerProgressReporter struct {
	spinner        *spinner.Spinner
	stages         []stageInfo
	currentStage   usecase.ExecutionStage
	stageStartTime time.Time
}

type stageInfo struct {
	Stage     usecase.ExecutionStage
	StartTime time.Time
	EndTime   time.Time
	Status    string
	Message   string
}

// NewSpinnerProgressReporter creates a new spinner-based progress reporter
func NewSpinnerProgressReporter() *SpinnerProgressReporter {
	// Create custom spinner with colors (matching v1 implementation)
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.HideCursor = false

	return &SpinnerProgressReporter{
		spinner: s,
		stages:  []stageInfo{},
	}
}

// ReportStage reports the current execution stage
func (r *SpinnerProgressReporter) ReportStage(ctx context.Context, stage usecase.ExecutionStage) {
	// Complete previous stage if any
	if r.currentStage != "" {
		r.completeCurrentStage()
	}

	// Special handling for different stages
	switch stage {
	case usecase.StageSimulating:
		// Start spinner for simulation
		if !r.spinner.Active() {
			r.spinner.Start()
		}
	case usecase.StageBroadcasting:
		// Continue spinner for broadcasting
		// The v1 implementation transitions from Simulating to Broadcasting
	case usecase.StageCompleted:
		// Stop spinner when completed
		r.spinner.Stop()
	}

	// Record new stage
	r.currentStage = stage
	r.stageStartTime = time.Now()
	r.stages = append(r.stages, stageInfo{
		Stage:     stage,
		StartTime: time.Now(),
		Status:    "running",
	})

	// Update spinner display
	r.updateSpinnerDisplay()
}

// OnProgress handles progress events
func (r *SpinnerProgressReporter) OnProgress(ctx context.Context, event usecase.ProgressEvent) {
	// Handle spinner states
	if event.Spinner {
		if !r.spinner.Active() {
			r.spinner.Start()
		}
		r.spinner.Suffix = " " + event.Message
	} else if r.spinner.Active() {
		r.spinner.Stop()
	}

	// Update current stage info if needed
	if len(r.stages) > 0 {
		r.stages[len(r.stages)-1].Message = event.Message
	}
}

// Info prints an info message
func (r *SpinnerProgressReporter) Info(message string) {
	// Stop spinner temporarily
	wasActive := false
	if r.spinner != nil && r.spinner.Active() {
		wasActive = true
		r.spinner.Stop()
	}

	color.New(color.FgCyan).Println(message)

	// Restart spinner if it was active
	if wasActive {
		r.spinner.Start()
	}
}

// Error prints an error message
func (r *SpinnerProgressReporter) Error(message string) {
	// Stop spinner temporarily
	wasActive := false
	if r.spinner != nil && r.spinner.Active() {
		wasActive = true
		r.spinner.Stop()
	}

	color.New(color.FgRed).Println(message)

	// Restart spinner if it was active
	if wasActive {
		r.spinner.Start()
	}
}

// completeCurrentStage marks the current stage as completed
func (r *SpinnerProgressReporter) completeCurrentStage() {
	if len(r.stages) > 0 {
		idx := len(r.stages) - 1
		r.stages[idx].EndTime = time.Now()
		r.stages[idx].Status = "completed"
	}
}

// updateSpinnerDisplay updates the spinner suffix with stage information
func (r *SpinnerProgressReporter) updateSpinnerDisplay() {
	var display string

	// Map our stages to v1 stage names for compatibility
	for i, stage := range r.stages {
		var stageName string
		var icon string
		var stageColor *color.Color

		// Map stage names
		switch stage.Stage {
		case usecase.StageSimulating:
			stageName = "Simulating"
		case usecase.StageBroadcasting:
			stageName = "Broadcasting"
		case usecase.StageCompleted:
			stageName = "Completed"
		default:
			// Skip other stages in display
			continue
		}

		// Determine icon and color based on status
		switch stage.Status {
		case "completed":
			icon = "✓"
			stageColor = color.New(color.FgGreen)
		case "running":
			icon = "●"
			stageColor = color.New(color.FgYellow)
		case "skipped":
			icon = "⊘"
			stageColor = color.New(color.FgWhite, color.Faint)
		default:
			icon = "○"
			stageColor = color.New(color.FgWhite)
		}

		// Calculate duration
		duration := ""
		if !stage.EndTime.IsZero() {
			duration = fmt.Sprintf(" (%s)", stage.EndTime.Sub(stage.StartTime).Round(time.Millisecond))
		} else if stage.Status == "running" {
			duration = fmt.Sprintf(" (%s)", time.Since(stage.StartTime).Round(time.Second))
		}

		// Add to display
		if i > 0 {
			display += " → "
		}
		display += fmt.Sprintf("%s %s%s", icon, stageColor.Sprint(stageName), duration)
	}

	r.spinner.Suffix = " " + display
}

// NopProgressReporter is a no-op progress reporter for non-interactive mode
type NopProgressReporter struct{}

// NewNopProgressReporter creates a new no-op progress reporter
func NewNopProgressReporter() *NopProgressReporter {
	return &NopProgressReporter{}
}

// ReportStage does nothing
func (r *NopProgressReporter) ReportStage(ctx context.Context, stage usecase.ExecutionStage) {}

// ReportProgress does nothing
func (r *NopProgressReporter) ReportProgress(ctx context.Context, event usecase.ProgressEvent) {}

// Ensure SpinnerProgressReporter implements ProgressSink
var _ usecase.ProgressSink = (*SpinnerProgressReporter)(nil)
