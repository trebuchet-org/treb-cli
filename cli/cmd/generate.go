package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/cli/pkg/generator"
	"github.com/trebuchet-org/treb-cli/cli/pkg/interactive"
	"github.com/trebuchet-org/treb-cli/cli/pkg/resolvers"
)

var (
	genStrategyFlag      string
	genProxyContractFlag string
	genProxyTypeFlag     string
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

		var contractName string
		if len(args) > 0 {
			contractName = args[0]
		}

		// Validate required fields in non-interactive mode
		if IsNonInteractive() {
			if contractName == "" {
				return fmt.Errorf("contract name is required in non-interactive mode")
			}
			if genStrategyFlag == "" {
				return fmt.Errorf("--strategy flag is required in non-interactive mode (CREATE2 or CREATE3)")
			}
		}

		// Create resolver and generator
		resolver := resolvers.NewContext(".", !IsNonInteractive())
		gen := generator.NewGenerator(".")

		// Resolve contract
		var contractInfo *contracts.ContractInfo
		var err error
		if contractName != "" {
			contractInfo, err = resolver.ResolveContractForImplementation(contractName)
			if err != nil {
				return fmt.Errorf("failed to resolve contract: %w", err)
			}
		} else if !IsNonInteractive() {
			// Interactive mode: let user pick a contract
			indexer, err := contracts.GetGlobalIndexer(".")
			if err != nil {
				return fmt.Errorf("failed to initialize contract indexer: %w", err)
			}
			contracts := indexer.GetDeployableContracts()
			if len(contracts) == 0 {
				return fmt.Errorf("no deployable contracts found")
			}
			contractInfo, err = interactive.SelectContract(contracts, "Select contract to generate deploy script:")
			if err != nil {
				return err
			}
		}

		// Handle strategy selection
		var strategy contracts.DeployStrategy
		if genStrategyFlag != "" {
			strategy, err = contracts.ValidateStrategy(genStrategyFlag)
			if err != nil {
				return fmt.Errorf("invalid strategy '%s': %w", genStrategyFlag, err)
			}
		} else if !IsNonInteractive() {
			// Interactive mode: let user pick strategy
			strategies := []string{"CREATE2", "CREATE3"}
			selector := interactive.NewSelector()
			strategyStr, _, err := selector.SelectOption("Select deployment strategy:", strategies, 1) // Default to CREATE3
			if err != nil {
				return fmt.Errorf("strategy selection failed: %w", err)
			}
			strategy, err = contracts.ValidateStrategy(strategyStr)
			if err != nil {
				return err
			}
		}

		// Generate the script
		return gen.GenerateDeployScript(contractInfo, strategy)
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

		var contractName string
		if len(args) > 0 {
			contractName = args[0]
		}

		// Validate required fields in non-interactive mode
		if IsNonInteractive() {
			if contractName == "" {
				return fmt.Errorf("contract name is required in non-interactive mode")
			}
			if genStrategyFlag == "" {
				return fmt.Errorf("--strategy flag is required in non-interactive mode (CREATE2 or CREATE3)")
			}
			if genProxyContractFlag == "" {
				return fmt.Errorf("--proxy-contract flag is required in non-interactive mode")
			}
		}

		// Create resolver and generator
		resolver := resolvers.NewContext(".", !IsNonInteractive())
		gen := generator.NewGenerator(".")

		// Resolve implementation contract
		var implementationInfo *contracts.ContractInfo
		var err error
		if contractName != "" {
			implementationInfo, err = resolver.ResolveContractForImplementation(contractName)
			if err != nil {
				return fmt.Errorf("failed to resolve implementation contract: %w", err)
			}
		} else if !IsNonInteractive() {
			// Interactive mode: let user pick a contract
			indexer, err := contracts.GetGlobalIndexer(".")
			if err != nil {
				return fmt.Errorf("failed to initialize contract indexer: %w", err)
			}
			contracts := indexer.GetDeployableContracts()
			if len(contracts) == 0 {
				return fmt.Errorf("no deployable contracts found")
			}
			implementationInfo, err = interactive.SelectContract(contracts, "Select implementation contract:")
			if err != nil {
				return err
			}
		}

		// Handle proxy contract selection
		var proxyInfo *contracts.ContractInfo
		if genProxyContractFlag != "" {
			proxyInfo, err = resolver.ResolveContractForProxy(genProxyContractFlag)
			if err != nil {
				return fmt.Errorf("failed to resolve proxy contract '%s': %w", genProxyContractFlag, err)
			}
		} else if !IsNonInteractive() {
			// Interactive mode: let user pick proxy contract
			proxyInfo, err = resolver.SelectProxyContract()
			if err != nil {
				return err
			}
		}

		// Handle strategy selection
		var strategy contracts.DeployStrategy
		if genStrategyFlag != "" {
			strategy, err = contracts.ValidateStrategy(genStrategyFlag)
			if err != nil {
				return fmt.Errorf("invalid strategy '%s': %w", genStrategyFlag, err)
			}
		} else if !IsNonInteractive() {
			// Interactive mode: let user pick strategy
			strategies := []string{"CREATE2", "CREATE3"}
			selector := interactive.NewSelector()
			strategyStr, _, err := selector.SelectOption("Select deployment strategy:", strategies, 1) // Default to CREATE3
			if err != nil {
				return fmt.Errorf("strategy selection failed: %w", err)
			}
			strategy, err = contracts.ValidateStrategy(strategyStr)
			if err != nil {
				return err
			}
		}

		// Handle proxy type
		var proxyType contracts.ProxyType
		if genProxyTypeFlag != "" {
			// Validate provided proxy type
			proxyType = contracts.ProxyType(genProxyTypeFlag)
			switch proxyType {
			case contracts.ProxyTypeOZTransparent, contracts.ProxyTypeOZUUPS, contracts.ProxyTypeCustom:
				// Valid
			default:
				return fmt.Errorf("invalid proxy type: %s (must be TransparentUpgradeable, UUPSUpgradeable, or Custom)", genProxyTypeFlag)
			}
		} else {
			// Auto-determine proxy type based on proxy contract name
			proxyType = determineProxyType(proxyInfo.Name)
			// If we can't determine it, just use Custom
		}

		// Generate the script
		return gen.GenerateProxyScript(implementationInfo, proxyInfo, strategy, proxyType)
	},
}

func init() {
	genCmd.AddCommand(genDeployCmd)
	genCmd.AddCommand(genProxyCmd)
	
	// Add flags to subcommands
	genDeployCmd.Flags().StringVar(&genStrategyFlag, "strategy", "", "Deployment strategy (CREATE2 or CREATE3)")
	genProxyCmd.Flags().StringVar(&genStrategyFlag, "strategy", "", "Deployment strategy (CREATE2 or CREATE3)")
	genProxyCmd.Flags().StringVar(&genProxyContractFlag, "proxy-contract", "", "Proxy contract (name or path:contract format)")
	genProxyCmd.Flags().StringVar(&genProxyTypeFlag, "proxy-type", "", "Proxy type (TransparentUpgradeable, UUPSUpgradeable, or Custom)")
}

// determineProxyType determines the proxy type based on the proxy contract name
func determineProxyType(proxyName string) contracts.ProxyType {
	if strings.Contains(proxyName, "TransparentUpgradeable") {
		return contracts.ProxyTypeOZTransparent
	} else if strings.Contains(proxyName, "ERC1967") || strings.Contains(proxyName, "UUPS") {
		return contracts.ProxyTypeOZUUPS
	}
	return contracts.ProxyTypeCustom
}
