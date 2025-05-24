package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show treb version",
	Long:  `Display the current version of treb CLI.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("treb version 0.1.0")
	},
}

func init() {
	// Add to root command
}