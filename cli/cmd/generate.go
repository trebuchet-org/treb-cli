package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/pkg/interactive"
)

var (
	genStrategyFlag      string
	genProxyContractFlag string
)

// genCmd represents the gen command
var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate deployment scripts",
	Long: `Interactive generator for creating deployment scripts.

Available types:
  deploy  - Standard contract deployment
  proxy   - Proxy deployment scripts`,
}

var genDeployCmd = &cobra.Command{
	Use:   "deploy [contract]",
	Short: "Generate deploy script for a contract",
	Long: `Generate a deployment script for a specific contract.

Examples:
  treb gen deploy Counter                        # Interactive mode
  treb gen deploy Counter --strategy CREATE3    # Non-interactive with strategy
  treb gen deploy --non-interactive Counter --strategy CREATE2  # Fully non-interactive`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !IsNonInteractive() {
			fmt.Println("🧙 Interactive Contract Deploy Script Generator")
		}
		
		// Create generator
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
	Long: `Generate a proxy deployment script for a specific contract.

Examples:
  treb gen proxy Counter                                    # Interactive mode
  treb gen proxy Counter --strategy CREATE3 --proxy-contract ERC1967Proxy  # Non-interactive
  treb gen proxy --non-interactive Counter --strategy CREATE2 --proxy-contract lib/openzeppelin-contracts/contracts/proxy/ERC1967/ERC1967Proxy.sol:ERC1967Proxy  # Fully non-interactive`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !IsNonInteractive() {
			fmt.Println("🧙 Interactive Proxy Deploy Script Generator")
		}
		
		// Create generator
		generator := interactive.NewGenerator(".")

		var contractName string
		if len(args) > 0 {
			contractName = args[0]
		}

		return generator.GenerateProxyScript(contractName)
	},
}

func init() {
	genCmd.AddCommand(genDeployCmd)
	genCmd.AddCommand(genProxyCmd)
	
	// Add flags to subcommands
	genDeployCmd.Flags().StringVar(&genStrategyFlag, "strategy", "", "Deployment strategy (CREATE2 or CREATE3)")
	genProxyCmd.Flags().StringVar(&genStrategyFlag, "strategy", "", "Deployment strategy (CREATE2 or CREATE3)")
	genProxyCmd.Flags().StringVar(&genProxyContractFlag, "proxy-contract", "", "Proxy contract (name or path:contract format)")
}
