package cmd

import (
	"github.com/trebuchet-org/treb-cli/cli/pkg/project"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize treb in a Foundry project",
	Long: `Initialize treb in an existing Foundry project by installing dependencies
and creating the deployment registry.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		initializer := project.NewInitializer()
		if err := initializer.Initialize(); err != nil {
			checkError(err)
		}
	},
}

func init() {
	// No flags needed
}

