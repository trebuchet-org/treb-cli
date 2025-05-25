package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/pkg/interactive"
)

// genCmd represents the gen command
var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate deployment scripts",
	Long: `Interactive generator for creating deployment scripts.

Available types:
  deploy  - Standard contract deployment
  proxy   - Proxy deployment scripts
  library - Library deployment scripts`,
}

var genDeployCmd = &cobra.Command{
	Use:   "deploy [contract]",
	Short: "Generate deploy script for a contract",
	Long:  `Generate a deployment script for a specific contract.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🧙 Interactive Contract Deploy Script Generator")
		generator := interactive.NewGenerator(".")

		var contractName string
		if len(args) > 0 {
			contractName = args[0]
		}

		return generator.GenerateDeployScript(contractName)
	},
}

var genProxyCmd = &cobra.Command{
	Use:   "proxy [contract]",
	Short: "Generate proxy deploy script for a contract",
	Long:  `Generate a proxy deployment script for a specific contract.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🧙 Interactive Proxy Deploy Script Generator")
		generator := interactive.NewGenerator(".")

		var contractName string
		if len(args) > 0 {
			contractName = args[0]
		}

		return generator.GenerateProxyDeployScript(contractName)
	},
}

var genLibraryCmd = &cobra.Command{
	Use:   "library [name]",
	Short: "Generate library deploy script",
	Long:  `Generate a deployment script for a library.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🧙 Interactive Library Deploy Script Generator")
		generator := interactive.NewGenerator(".")

		var libraryName string
		if len(args) > 0 {
			libraryName = args[0]
		}

		return generator.GenerateLibraryScript(libraryName)
	},
}

func init() {
	genCmd.AddCommand(genDeployCmd)
	genCmd.AddCommand(genProxyCmd)
	genCmd.AddCommand(genLibraryCmd)
}
