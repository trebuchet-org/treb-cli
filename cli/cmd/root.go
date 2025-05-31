package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	nonInteractive bool
)

var rootCmd = &cobra.Command{
	Use:   "treb",
	Short: "Smart contract deployment orchestrator for Foundry",
	Long: `Trebuchet (treb) orchestrates Foundry script execution for deterministic 
smart contract deployments using CreateX factory contracts.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
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
	rootCmd.AddGroup(&cobra.Group{
		ID:    "legacy",
		Title: "Legacy Commands (v1 registry)", 
	})

	// Main workflow commands (init at the end)
	// deployCmd.GroupID = "main" // TODO: Fix deployment package
	runCmd.GroupID = "main"
	listCmd.GroupID = "main"
	showCmd.GroupID = "main"
	genCmd.GroupID = "main"
	initCmd.GroupID = "main"
	
	// Management commands
	contextCmd.GroupID = "management"
	
	// Legacy v1 commands
	verifyV1Cmd.GroupID = "legacy"
	syncV1Cmd.GroupID = "legacy"
	
	// Additional commands (merged with other utility commands)
	// debugCmd and versionCmd will appear in "Additional Commands" section
	// since they don't have a GroupID set

	// rootCmd.AddCommand(deployCmd) // TODO: Fix deployment package
	// Commands are registered in their respective init() functions
	// rootCmd.AddCommand(runCmd)
	// rootCmd.AddCommand(listCmd)
	// rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(genCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(contextCmd)
	rootCmd.AddCommand(versionCmd)
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// IsNonInteractive returns true if the non-interactive flag is set
func IsNonInteractive() bool {
	return nonInteractive
}