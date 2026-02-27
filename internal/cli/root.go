package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/internal/app"
	"github.com/trebuchet-org/treb-cli/internal/config"
	domainconfig "github.com/trebuchet-org/treb-cli/internal/domain/config"
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

			// Initialize app with DI
			app, err := app.InitApp(v, cmd)
			if err != nil {
				return fmt.Errorf("failed to initialize app: %w", err)
			}

			// Show deprecation warning for legacy foundry.toml config
			showLegacyConfigWarning(cmd, app.Config)

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

	composeCmd := NewComposeCmd()
	composeCmd.GroupID = "main"
	rootCmd.AddCommand(composeCmd)

	syncCmd := NewSyncCmd()
	syncCmd.GroupID = "management"
	rootCmd.AddCommand(syncCmd)

	tagCmd := NewTagCmd()
	tagCmd.GroupID = "management"
	rootCmd.AddCommand(tagCmd)

	registerCmd := NewRegisterCmd()
	registerCmd.GroupID = "management"
	rootCmd.AddCommand(registerCmd)

	forkCmd := NewForkCmd()
	forkCmd.GroupID = "main"
	rootCmd.AddCommand(forkCmd)

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

	resetCmd := NewResetCmd()
	resetCmd.GroupID = "management"
	rootCmd.AddCommand(resetCmd)

	configCmd := NewConfigCmd()
	configCmd.GroupID = "management"
	rootCmd.AddCommand(configCmd)

	migrateConfigCmd := NewMigrateConfigCmd()
	migrateConfigCmd.GroupID = "management"
	rootCmd.AddCommand(migrateConfigCmd)

	// Version command
	versionCmd := NewVersionCmd()
	rootCmd.AddCommand(versionCmd)

	return rootCmd
}

// isForkActiveForCurrentNetwork checks if fork mode is active for the currently configured network.
// Returns true and the network name if a fork is active.
func isForkActiveForCurrentNetwork(ctx context.Context, a *app.App) (bool, string) {
	if a.Config.Network == nil {
		return false, ""
	}
	state, err := a.ForkStateStore.Load(ctx)
	if err != nil {
		return false, ""
	}
	return state.IsForkActive(a.Config.Network.Name), a.Config.Network.Name
}

// suppressedCommands are commands that should not show the deprecation warning
var suppressedCommands = map[string]bool{
	"version":        true,
	"help":           true,
	"completion":     true,
	"init":           true,
	"migrate-config": true,
}

// showLegacyConfigWarning prints a deprecation warning to stderr when
// sender config is detected in foundry.toml and treb.toml does not exist.
func showLegacyConfigWarning(cmd *cobra.Command, cfg *domainconfig.RuntimeConfig) {
	if !shouldShowDeprecationWarning(cmd.Name(), cfg) {
		return
	}
	yellow := color.New(color.FgYellow)
	_, _ = yellow.Fprintln(os.Stderr, "Warning: treb config detected in foundry.toml — this is deprecated.")
	_, _ = yellow.Fprintln(os.Stderr, "Run `treb migrate-config` to move your config to treb.toml.")
}

// shouldShowDeprecationWarning determines if the deprecation warning should be shown.
func shouldShowDeprecationWarning(cmdName string, cfg *domainconfig.RuntimeConfig) bool {
	if suppressedCommands[cmdName] {
		return false
	}
	if cfg.JSON {
		return false
	}
	if cfg.ConfigSource != "foundry.toml" {
		return false
	}
	return hasLegacyTrebConfig(cfg.FoundryConfig)
}

// hasLegacyTrebConfig returns true if any foundry.toml profile has treb sender config.
func hasLegacyTrebConfig(fc *domainconfig.FoundryConfig) bool {
	if fc == nil {
		return false
	}
	for _, profile := range fc.Profile {
		if profile.Treb != nil {
			return true
		}
	}
	return false
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
