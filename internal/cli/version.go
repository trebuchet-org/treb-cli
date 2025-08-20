package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set during build time
var Version = "dev"

// NewVersionCmd creates the version command
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number of treb",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("treb version %s\n", Version)
		},
	}
}
