package render

import (
	"fmt"
	"io"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// VerifyRenderer handles rendering of verification results
type VerifyRenderer struct {
	out         io.Writer
	interactive bool
}

// NewVerifyRenderer creates a new verify renderer
func NewVerifyRenderer(out io.Writer, interactive bool) *VerifyRenderer {
	return &VerifyRenderer{
		out:         out,
		interactive: interactive,
	}
}

// RenderVerifyAllResult renders the result of verifying all deployments
func (r *VerifyRenderer) RenderVerifyAllResult(result *usecase.VerifyAllResult, options usecase.VerifyOptions) error {
	// Show skipped contracts first
	if len(result.Skipped) > 0 {
		color.New(color.FgCyan, color.Bold).Fprintf(r.out, "Skipping %d pending/undeployed contracts:\n", len(result.Skipped))
		for _, skipped := range result.Skipped {
			displayName := r.getDisplayName(skipped.Deployment)
			fmt.Fprintf(r.out, "  ⏭️  chain:%d/%s/%s (%s)\n",
				skipped.Deployment.ChainID,
				skipped.Deployment.Namespace,
				displayName,
				skipped.Reason,
			)
		}
		fmt.Fprintln(r.out)
	}

	if len(result.ToVerify) == 0 {
		if options.Force {
			color.New(color.FgYellow).Fprintln(r.out, "No deployed contracts found to verify.")
		} else {
			color.New(color.FgYellow).Fprintln(r.out, "No unverified deployed contracts found. Use --force to re-verify all contracts.")
		}
		return nil
	}

	// Show contracts to verify
	if options.Force {
		color.New(color.FgCyan, color.Bold).Fprintf(r.out, "Found %d deployed contracts to verify (including verified ones with --force):\n", len(result.ToVerify))
	} else {
		color.New(color.FgCyan, color.Bold).Fprintf(r.out, "Found %d unverified deployed contracts to verify:\n", len(result.ToVerify))
	}

	// Show verification progress and results
	for i, verifyResult := range result.Results {
		deployment := verifyResult.Deployment
		displayName := r.getDisplayName(deployment)

		// Show status indicator
		statusIcon := r.getStatusIcon(deployment.Verification.Status)
		fmt.Fprintf(r.out, "  %s chain:%d/%s/%s\n",
			statusIcon,
			deployment.ChainID,
			deployment.Namespace,
			displayName,
		)

		// Show result
		if verifyResult.Success {
			color.New(color.FgGreen).Fprintf(r.out, "    ✓ Verification completed\n")
		} else {
			for _, err := range verifyResult.Errors {
				color.New(color.FgRed).Fprintf(r.out, "    ✗ %s\n", err)
			}
		}

		// Add spacing between items except for the last one
		if i < len(result.Results)-1 {
			fmt.Fprintln(r.out)
		}
	}

	// Show summary
	fmt.Fprintf(r.out, "\nVerification complete: %d/%d successful\n", result.SuccessCount, len(result.ToVerify))
	return nil
}

// RenderVerifyResult renders the result of verifying a specific deployment
func (r *VerifyRenderer) RenderVerifyResult(result *usecase.VerifyResult, options usecase.VerifyOptions) error {
	deployment := result.Deployment
	displayName := r.getDisplayName(deployment)

	if result.Success && len(result.Errors) == 1 && result.Errors[0] == "Already verified. Use --force to re-verify." {
		// Already verified case
		color.New(color.FgYellow).Fprintf(r.out, "Contract %s is already verified. Use --force to re-verify.\n", displayName)
		return nil
	}

	// Show what we're verifying
	if options.ContractPath != "" {
		color.New(color.FgYellow).Fprintf(r.out, "Using manual contract path: %s\n", options.ContractPath)
	}

	if !options.Debug && r.interactive {
		// Show spinner during verification
		s := r.createSpinner(fmt.Sprintf("Verifying chain:%d/%s/%s...",
			deployment.ChainID, deployment.Namespace, displayName))
		// In real implementation, we'd need to handle async verification
		// For now, we just stop the spinner immediately
		s.Stop()
	}

	if result.Success {
		color.New(color.FgGreen).Fprintln(r.out, "✓ Verification completed successfully!")

		// Show verification status
		r.showVerificationStatus(deployment)
	} else {
		for _, err := range result.Errors {
			color.New(color.FgRed).Fprintf(r.out, "✗ Verification failed: %s\n", err)
		}
	}

	return nil
}

// getDisplayName returns the display name for a deployment
func (r *VerifyRenderer) getDisplayName(deployment *models.Deployment) string {
	if deployment.Label != "" {
		return fmt.Sprintf("%s:%s", deployment.ContractName, deployment.Label)
	}
	return deployment.ContractName
}

// getStatusIcon returns the appropriate status icon for a verification status
func (r *VerifyRenderer) getStatusIcon(status models.VerificationStatus) string {
	switch status {
	case models.VerificationStatusVerified:
		return "🔄" // Re-verifying
	case models.VerificationStatusFailed:
		return "⚠️" // Retrying failed
	case models.VerificationStatusPartial:
		return "🔁" // Retrying partial
	case models.VerificationStatusUnverified:
		return "⏳" // First attempt
	default:
		return "🆕" // New verification
	}
}

// createSpinner creates a new spinner with the given message
func (r *VerifyRenderer) createSpinner(message string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + message
	_ = s.Color("cyan", "bold")
	s.Start()
	return s
}

// showVerificationStatus displays the verification status details
func (r *VerifyRenderer) showVerificationStatus(deployment *models.Deployment) {
	if deployment.Verification.Verifiers == nil {
		return
	}

	fmt.Fprintln(r.out, "\nVerification Status:")
	for verifier, status := range deployment.Verification.Verifiers {
		switch status.Status {
		case "verified":
			color.New(color.FgGreen).Fprintf(r.out, "  %s: ✓ Verified",
				cases.Title(language.English).String(verifier))
			if status.URL != "" {
				fmt.Fprintf(r.out, " - %s", status.URL)
			}
			fmt.Fprintln(r.out)
		case "failed":
			color.New(color.FgRed).Fprintf(r.out, "  %s: ✗ Failed",
				cases.Title(language.English).String(verifier))
			if status.Reason != "" {
				fmt.Fprintf(r.out, " - %s", status.Reason)
			}
			fmt.Fprintln(r.out)
		case "pending":
			color.New(color.FgYellow).Fprintf(r.out, "  %s: ⏳ Pending\n",
				cases.Title(language.English).String(verifier))
		}
	}
}
