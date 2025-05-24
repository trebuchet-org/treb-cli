package cmd

import (
	"github.com/trebuchet-org/treb-cli/cli/pkg/interactive"
	"github.com/spf13/cobra"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate deployment scripts",
	Long: `Interactive generator for creating deployment scripts.

Available types:
  deploy - Standard contract deployment
  proxy  - Proxy deployment scripts`,
}

var generateDeployCmd = &cobra.Command{
	Use:   "deploy <contract>",
	Short: "Generate deploy script for a contract",
	Long:  `Generate a deployment script for a specific contract.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		generator := interactive.NewGenerator(".")
		if len(args) > 0 {
			return generator.GenerateDeployScriptForContract(args[0])
		}
		return generator.GenerateDeployScript()
	},
}

var generateProxyCmd = &cobra.Command{
	Use:   "proxy <contract>",
	Short: "Generate proxy deploy script for a contract",
	Long:  `Generate a proxy deployment script for a specific contract.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		generator := interactive.NewGenerator(".")
		if len(args) > 0 {
			return generator.GenerateProxyDeployScriptForContract(args[0])
		}
		return generator.GenerateProxyDeployScript()
	},
}

func init() {
	generateCmd.AddCommand(generateDeployCmd)
	generateCmd.AddCommand(generateProxyCmd)
}


