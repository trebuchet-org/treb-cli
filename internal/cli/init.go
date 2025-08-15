package cli

import (
	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/internal/cli/render"
)

// NewInitCmd creates the init command
func NewInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize treb in a Foundry project",
		Long: `Initialize treb in an existing Foundry project by installing dependencies
and creating the deployment registry.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd)
		},
	}

	return cmd
}

// runInit executes the init command
func runInit(cmd *cobra.Command) error {
	// Get app instance
	app, err := getApp(cmd)
	if err != nil {
		return err
	}

	// Execute initialization
	result, err := app.InitProject.Execute(cmd.Context())
	if err != nil {
		// Still render partial results even on error
		if result != nil {
			renderer := render.NewInitRenderer()
			_ = renderer.Render(result)
		}
		return err
	}

	// Render the result
	renderer := render.NewInitRenderer()
	return renderer.Render(result)
}