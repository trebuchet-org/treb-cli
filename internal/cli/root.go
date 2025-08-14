package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
			// Skip for help/version commands
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
			v := config.SetupViper(projectRoot)

			// Bind global flags that have been set
			bindGlobalFlags(v, cmd)

			// Create progress sink (can be replaced with proper implementation later)
			sink := progress.NewNopSink()

			// Initialize app with DI
			appInstance, err := app.InitApp(v, sink)
			if err != nil {
				return fmt.Errorf("failed to initialize app: %w", err)
			}

			// Store app in context
			ctx := context.WithValue(cmd.Context(), appKey, appInstance)
			
			// Add timeout if configured
			if appInstance.Config.Timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, appInstance.Config.Timeout)
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
    rootCmd.PersistentFlags().Bool("debug", false, "Enable debug output")
    rootCmd.PersistentFlags().Bool("non-interactive", false, "Disable interactive prompts")
	rootCmd.PersistentFlags().StringP("namespace", "s", "", "Deployment namespace (defaults to 'default')")
	rootCmd.PersistentFlags().StringP("network", "n", "", "Network to use (e.g., mainnet, sepolia)")

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

	// TODO: Add more commands as they are migrated
	// deployCmd := NewDeployCmd(baseCfg)
	// deployCmd.GroupID = "main"
	// rootCmd.AddCommand(deployCmd)

	// genCmd := NewGenCmd(baseCfg)
	// genCmd.GroupID = "main"
	// rootCmd.AddCommand(genCmd)

	// verifyCmd := NewVerifyCmd(baseCfg)
	// verifyCmd.GroupID = "main"
	// rootCmd.AddCommand(verifyCmd)

	// initCmd := NewInitCmd(baseCfg)
	// initCmd.GroupID = "main"
	// rootCmd.AddCommand(initCmd)

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

	// syncCmd := NewSyncCmd(baseCfg)
	// syncCmd.GroupID = "management"
	// rootCmd.AddCommand(syncCmd)

	// tagCmd := NewTagCmd(baseCfg)
	// tagCmd.GroupID = "management"
	// rootCmd.AddCommand(tagCmd)

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

// bindGlobalFlags binds command flags to viper
func bindGlobalFlags(v *viper.Viper, cmd *cobra.Command) {
	// Only bind flags that exist and have been changed
	if f := cmd.Flag("debug"); f != nil && f.Changed {
		v.Set("debug", f.Value.String())
	}
	if f := cmd.Flag("non-interactive"); f != nil && f.Changed {
		v.Set("non_interactive", f.Value.String())
	}
    // Intentionally omit --json to preserve v1 compatibility in usage output
	if f := cmd.Flag("namespace"); f != nil && f.Changed {
		v.Set("namespace", f.Value.String())
	}
	if f := cmd.Flag("network"); f != nil && f.Changed {
		v.Set("network", f.Value.String())
	}
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