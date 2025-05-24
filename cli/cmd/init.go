package cmd

import (
	"github.com/trebuchet-org/treb-cli/cli/pkg/project"
	"github.com/spf13/cobra"
)

var (
	createxFlag bool
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new treb project",
	Long: `Initialize a new treb project with enhanced registry and optional CreateX integration.

This command sets up:
- Project structure with lib submodule
- treb-sol as git submodule
- Initial registry configuration
- Foundry project setup`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]
		
		initializer := project.NewInitializer(projectName, createxFlag)
		if err := initializer.Initialize(); err != nil {
			checkError(err)
		}
	},
}

func init() {
	initCmd.Flags().BoolVar(&createxFlag, "createx", true, "Initialize with CreateX integration")
}

