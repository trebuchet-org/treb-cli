package cmd

import (
	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/pkg/project"
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
