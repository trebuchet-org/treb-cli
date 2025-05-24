package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "treb",
	Short: "Trebuchet - Foundry Script Orchestration with CreateX",
	Long: `treb is a CLI tool that orchestrates Foundry script execution for 
deterministic smart contract deployments using CreateX.

Go handles configuration, planning, and registry management while all chain 
interactions happen through proven Foundry scripts.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(predictCmd)
	rootCmd.AddCommand(verifyCmd)
	rootCmd.AddCommand(deploymentsCmd)
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}