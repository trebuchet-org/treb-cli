package progress

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// VerifyProgress implements progress reporting for contract verification
type VerifyProgress struct {
	out         io.Writer
	interactive bool
	spinner     *spinner.Spinner
	startTime   time.Time
}

// NewVerifyProgress creates a new verification progress reporter
func NewVerifyProgress(out io.Writer, interactive bool) *VerifyProgress {
	return &VerifyProgress{
		out:         out,
		interactive: interactive,
		startTime:   time.Now(),
	}
}

// OnProgress handles progress events
func (v *VerifyProgress) OnProgress(ctx context.Context, event usecase.ProgressEvent) {
	if !v.interactive {
		// In non-interactive mode, just print the message
		if event.Message != "" {
			fmt.Fprintln(v.out, event.Message)
		}
		return
	}

	// Handle spinner states
	if event.Spinner {
		if v.spinner == nil {
			v.spinner = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			v.spinner.Writer = v.out
			_ = v.spinner.Color("cyan", "bold")
		}

		// Update spinner message
		v.spinner.Suffix = " " + event.Message

		if !v.spinner.Active() {
			v.spinner.Start()
		}
	} else if v.spinner != nil && v.spinner.Active() {
		v.spinner.Stop()
	}

	// Handle stage-specific messages
	switch event.Stage {
	case "gathering":
		// Initial gathering phase
		if !v.interactive {
			fmt.Fprintln(v.out, "🔍 Gathering deployments to verify...")
		}

	case "verifying":
		// Show progress for multiple verifications
		if event.Total > 0 && !v.interactive {
			fmt.Fprintf(v.out, "📝 [%d/%d] %s\n", event.Current, event.Total, event.Message)
		}

	case "network-resolve":
		// Network resolution phase
		if !v.interactive {
			fmt.Fprintf(v.out, "🌐 %s\n", event.Message)
		}

	case "verification":
		// Actual verification submission
		if !v.interactive {
			fmt.Fprintf(v.out, "🚀 %s\n", event.Message)
		}

	case "completed":
		// Stop any active spinner
		if v.spinner != nil && v.spinner.Active() {
			v.spinner.Stop()
		}

		// Show completion time
		duration := time.Since(v.startTime)
		color.New(color.FgGreen).Fprintf(v.out, "✅ Verification completed in %s\n", duration.Round(time.Millisecond))
	}
}

// Info prints an info message
func (v *VerifyProgress) Info(message string) {
	// Stop spinner temporarily
	wasActive := false
	if v.spinner != nil && v.spinner.Active() {
		wasActive = true
		v.spinner.Stop()
	}

	color.New(color.FgCyan).Fprintln(v.out, "ℹ️  "+message)

	// Restart spinner if it was active
	if wasActive && v.spinner != nil {
		v.spinner.Start()
	}
}

// Error prints an error message
func (v *VerifyProgress) Error(message string) {
	// Stop spinner temporarily
	wasActive := false
	if v.spinner != nil && v.spinner.Active() {
		wasActive = true
		v.spinner.Stop()
	}

	color.New(color.FgRed).Fprintln(v.out, "❌ "+message)

	// Restart spinner if it was active
	if wasActive && v.spinner != nil {
		v.spinner.Start()
	}
}

// Ensure it implements the interface
var _ usecase.ProgressSink = (*VerifyProgress)(nil)
