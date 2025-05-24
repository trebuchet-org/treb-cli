package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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
	// Add command groups
	rootCmd.AddGroup(&cobra.Group{
		ID:    "main",
		Title: "Main Commands",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "management",
		Title: "Management Commands", 
	})

	// Main workflow commands (init at the end)
	deployCmd.GroupID = "main"
	listCmd.GroupID = "main"
	showCmd.GroupID = "main"
	verifyCmd.GroupID = "main"
	genCmd.GroupID = "main"
	initCmd.GroupID = "main"
	
	// Management commands
	tagCmd.GroupID = "management"
	syncCmd.GroupID = "management"
	configCmd.GroupID = "management"
	
	// Additional commands (merged with other utility commands)
	// debugCmd and versionCmd will appear in "Additional Commands" section
	// since they don't have a GroupID set

	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(verifyCmd)
	rootCmd.AddCommand(genCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(tagCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(debugCmd)
	rootCmd.AddCommand(versionCmd)
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}