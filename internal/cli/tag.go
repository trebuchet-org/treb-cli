package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/internal/cli/renderers"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// tagFlags holds command-specific flags
type tagFlags struct {
	add    string
	remove string
}

// NewTagCmd creates the tag command
func NewTagCmd() *cobra.Command {
	flags := &tagFlags{}

	cmd := &cobra.Command{
		Use:   "tag <deployment|address>",
		Short: "Manage deployment tags",
		Long: `Add or remove version tags on deployments.
Without flags, shows current tags.

Examples:
  treb tag Counter:v1                  # Show current tags
  treb tag Counter:v1 --add v1.0.0     # Add a tag
  treb tag Counter:v1 --remove v1.0.0  # Remove a tag`,
		Args: cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTag(cmd, args[0], flags)
		},
	}

	// Add flags
	cmd.Flags().StringVar(&flags.add, "add", "", "Add a tag to the deployment")
	cmd.Flags().StringVar(&flags.remove, "remove", "", "Remove a tag from the deployment")

	return cmd
}

// runTag executes the tag command
func runTag(cmd *cobra.Command, identifier string, flags *tagFlags) error {
	// Get app instance
	app, err := getApp(cmd)
	if err != nil {
		return err
	}

	// Validate flags
	if flags.add != "" && flags.remove != "" {
		return fmt.Errorf("cannot use --add and --remove together")
	}

	// Determine operation
	operation := "show"
	tag := ""
	if flags.add != "" {
		operation = "add"
		tag = flags.add
	} else if flags.remove != "" {
		operation = "remove"
		tag = flags.remove
	}

	// Execute tag operation
	params := usecase.TagDeploymentParams{
		Identifier: identifier,
		Tag:        tag,
		Operation:  operation,
		Namespace:  app.Config.Namespace,
	}
	
	// Add ChainID if network is configured
	if app.Config.Network != nil {
		params.ChainID = app.Config.Network.ChainID
	}

	result, err := app.TagDeployment.Execute(cmd.Context(), params)
	if err != nil {
		// Handle multiple matches error for interactive selection
		if strings.Contains(err.Error(), "multiple deployments found") && !app.Config.NonInteractive {
			// Try interactive selection
			chainID := uint64(0)
			if app.Config.Network != nil {
				chainID = app.Config.Network.ChainID
			}
			
			deployment, err := app.TagDeployment.FindDeploymentInteractive(
				cmd.Context(),
				identifier,
				chainID,
				app.Config.Namespace,
				app.Selector,
			)
			if err != nil {
				return err
			}

			// Re-execute with specific deployment ID
			params.Identifier = deployment.ID
			result, err = app.TagDeployment.Execute(cmd.Context(), params)
			if err != nil {
				// Handle tag already exists/doesn't exist errors as warnings
				if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "does not exist") {
					fmt.Println(renderers.FormatWarning(err.Error()))
					return nil
				}
				return err
			}
		} else {
			// Handle tag already exists/doesn't exist errors as warnings
			if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "does not exist") {
				fmt.Println(renderers.FormatWarning(err.Error()))
				return nil
			}
			return err
		}
	}

	// Render the result
	renderer := renderers.NewTagRenderer(app.Config)
	return renderer.Render(result)
}

