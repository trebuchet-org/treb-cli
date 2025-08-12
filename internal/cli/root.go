package cli

import (
	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/internal/app"
)

var (
	nonInteractive bool
)

// NewRootCmd creates the root command for the v2 CLI
func NewRootCmd() *cobra.Command {
	// Create base config that will be passed to all commands
	baseCfg := &app.Config{
		ProjectRoot: ".",
		DataDir:     ".treb",
	}

	rootCmd := &cobra.Command{
		Use:   "treb",
		Short: "Smart contract deployment orchestrator for Foundry",
		Long: `Trebuchet (treb) orchestrates Foundry script execution for deterministic 
smart contract deployments using CreateX factory contracts.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Update config with flag values
			baseCfg.NonInteractive = nonInteractive
		},
	}

	// Global flags
	rootCmd.PersistentFlags().BoolVar(&nonInteractive, "non-interactive", false, "Disable interactive prompts")

	// Add command groups
	rootCmd.AddGroup(&cobra.Group{
		ID:    "main",
		Title: "Main Commands",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "management",
		Title: "Management Commands",
	})

	// Add commands
	listCmd := NewListCmd(baseCfg)
	listCmd.GroupID = "main"
	rootCmd.AddCommand(listCmd)

	showCmd := NewShowCmd(baseCfg)
	showCmd.GroupID = "main"
	rootCmd.AddCommand(showCmd)

	// TODO: Add more commands as they are migrated
	// deployCmd := NewDeployCmd(baseCfg)
	// deployCmd.GroupID = "main"
	// rootCmd.AddCommand(deployCmd)

	// runCmd := NewRunCmd(baseCfg)
	// runCmd.GroupID = "main"
	// rootCmd.AddCommand(runCmd)

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
	// configCmd := NewConfigCmd(baseCfg)
	// configCmd.GroupID = "management"
	// rootCmd.AddCommand(configCmd)

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

// IsNonInteractive returns true if the non-interactive flag is set
func IsNonInteractive() bool {
	return nonInteractive
}