package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/internal/adapters/progress"
	"github.com/trebuchet-org/treb-cli/internal/app"
	"github.com/trebuchet-org/treb-cli/internal/config"
)

// contextKey is the type for context keys
type contextKey string

const (
	// appKey is the context key for the app instance
	appKey contextKey = "app"
)

// NewRootCmd creates the root command for the v2 CLI
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "treb",
		Short: "Smart contract deployment orchestrator for Foundry",
		Long: `Trebuchet (treb) orchestrates Foundry script execution for deterministic 
smart contract deployments using CreateX factory contracts.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Skip for help/version/completion commands
			if cmd.Name() == "version" || cmd.Name() == "help" || cmd.Name() == "completion" {
				return nil
			}

			// Find project root
			projectRoot, err := config.FindProjectRoot()
			if err != nil {
				// Some commands might not need a project (like init)
				if cmd.Name() != "init" {
					return err
				}
				projectRoot = "."
			}

			// Set up viper
			v := config.SetupViper(projectRoot, cmd)

			// Create progress sink (can be replaced with proper implementation later)
			sink := progress.NewNopSink()

			// Initialize app with DI
			app, err := app.InitApp(v, sink)
			if err != nil {
				return fmt.Errorf("failed to initialize app: %w", err)
			}

			// Store app in context
			ctx := context.WithValue(cmd.Context(), appKey, app)

			// Add timeout if configured
			if app.Config.Timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, app.Config.Timeout)
				// Store cancel func to be called on command completion
				cmd.PostRun = func(cmd *cobra.Command, args []string) {
					cancel()
				}
			}

			cmd.SetContext(ctx)

			return nil
		},
	}

	// Global flags
	rootCmd.PersistentFlags().Bool("non-interactive", false, "Disable interactive prompts")

	// Add command groups
	rootCmd.AddGroup(&cobra.Group{
		ID:    "main",
		Title: "Main Commands",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "management",
		Title: "Management Commands",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "deployment",
		Title: "Deployment Commands",
	})

	// Add deployment commands group
	deploymentCmd := newDeploymentCommands()
	rootCmd.AddCommand(deploymentCmd)

	// Add main commands
	initCmd := NewInitCmd()
	initCmd.GroupID = "main"
	rootCmd.AddCommand(initCmd)

	listCmd := NewListCmd()
	listCmd.GroupID = "main"
	rootCmd.AddCommand(listCmd)

	showCmd := NewShowCmd()
	showCmd.GroupID = "main"
	rootCmd.AddCommand(showCmd)

	generateCmd := NewGenerateCmd()
	generateCmd.GroupID = "main"
	rootCmd.AddCommand(generateCmd)

	runCmd := NewRunCmd()
	runCmd.GroupID = "main"
	rootCmd.AddCommand(runCmd)

	verifyCmd := NewVerifyCmd()
	verifyCmd.GroupID = "main"
	rootCmd.AddCommand(verifyCmd)

	orchestrateCmd := NewOrchestrateCmd()
	orchestrateCmd.GroupID = "main"
	rootCmd.AddCommand(orchestrateCmd)

	syncCmd := NewSyncCmd()
	syncCmd.GroupID = "management"
	rootCmd.AddCommand(syncCmd)

	tagCmd := NewTagCmd()
	tagCmd.GroupID = "management"
	rootCmd.AddCommand(tagCmd)

	devCmd := NewDevCmd()
	devCmd.GroupID = "management"
	rootCmd.AddCommand(devCmd)

	// Management commands
	networksCmd := NewNetworksCmd()
	networksCmd.GroupID = "management"
	rootCmd.AddCommand(networksCmd)

	pruneCmd := NewPruneCmd()
	pruneCmd.GroupID = "management"
	rootCmd.AddCommand(pruneCmd)

	configCmd := NewConfigCmd()
	configCmd.GroupID = "management"
	rootCmd.AddCommand(configCmd)

	// Version command
	versionCmd := NewVersionCmd()
	rootCmd.AddCommand(versionCmd)

	return rootCmd
}

// newDeploymentCommands creates the deployment command group
func newDeploymentCommands() *cobra.Command {
	deployCmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deployment commands",
		Long:  "Commands for managing smart contract deployments",
	}

	// Add deployment subcommands (placeholder for now)
	// deployCmd.AddCommand(newRunCmd())
	// deployCmd.AddCommand(newGenerateCmd())
	// deployCmd.AddCommand(newVerifyCmd())

	return deployCmd
}

// getApp retrieves the app instance from the command context
func getApp(cmd *cobra.Command) (*app.App, error) {
	appInstance := cmd.Context().Value(appKey)
	if appInstance == nil {
		return nil, fmt.Errorf("app not initialized")
	}

	app, ok := appInstance.(*app.App)
	if !ok {
		return nil, fmt.Errorf("invalid app instance")
	}

	return app, nil
}
