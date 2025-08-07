package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/cli/pkg/interactive"
	"github.com/trebuchet-org/treb-cli/cli/pkg/resolvers"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// genCmd represents the gen command
var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate deployment scripts",
	Long: `Generate deployment scripts for contracts and libraries.

This command creates template scripts using treb-sol's base contracts.
The generated scripts handle both direct deployments and proxy patterns.`,
}

var genDeployCmd = &cobra.Command{
	Use:   "deploy <artifact>",
	Short: "Generate a deployment script for a contract or library",
	Long: `Generate a deployment script for a contract or library.

This command automatically detects whether the artifact is a library or contract
and generates the appropriate deployment script.

For contracts, you can optionally generate a proxy deployment pattern by using
the --proxy flag. If --proxy is specified without a value, an interactive
proxy selection will be shown.

Examples:
  # Library deployment
  treb gen deploy MathUtils
  treb gen deploy src/libs/StringUtils.sol:StringUtils
  
  # Contract deployment
  treb gen deploy Counter
  treb gen deploy src/Token.sol:Token
  
  # Proxy deployment (interactive proxy selection)
  treb gen deploy Counter --proxy
  
  # Proxy deployment with specific proxy
  treb gen deploy Counter --proxy --proxy-contract TransparentUpgradeableProxy
  treb gen deploy MyToken --proxy --proxy-contract src/proxies/CustomProxy.sol:CustomProxy
  
  # Custom script path
  treb gen deploy Counter --script-path script/deploy/CustomDeploy.s.sol`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		artifactPath := args[0]

		// Parse flags
		useProxy, _ := cmd.Flags().GetBool("proxy")
		proxyContract, _ := cmd.Flags().GetString("proxy-contract")
		customPath, _ := cmd.Flags().GetString("script-path")
		strategyStr, _ := cmd.Flags().GetString("strategy")

		// Default strategy
		strategy := contracts.StrategyCreate3
		if strategyStr != "" {
			var err error
			strategy, err = contracts.ValidateStrategy(strategyStr)
			if err != nil {
				return err
			}
		}

		// Initialize contract indexer
		isInteractive := !rootCmd.PersistentFlags().Changed("non-interactive")
		indexer, err := contracts.GetGlobalIndexer(".")
		if err != nil {
			return fmt.Errorf("failed to initialize contract indexer: %w", err)
		}
		ctx := resolvers.NewContractsResolver(indexer, isInteractive)

		// Resolve the main artifact
		contractInfo, err := ctx.ResolveContract(artifactPath, types.AllContractsFilter())
		if err != nil {
			return err
		}

		// Check if it's a library
		isLibrary := contractInfo.IsLibrary

		// Validate proxy usage
		if isLibrary && useProxy {
			return fmt.Errorf("libraries cannot be deployed with proxies")
		}

		// Use the resolved contract name for script naming
		contractName := contractInfo.Name

		// Build full artifact path if not already specified
		if !strings.Contains(artifactPath, ":") {
			artifactPath = fmt.Sprintf("%s:%s", contractInfo.Path, contractInfo.Name)
		}

		// Determine script path
		var scriptPath string
		if customPath != "" {
			scriptPath = customPath
		} else {
			if useProxy {
				scriptPath = filepath.Join("script", "deploy", fmt.Sprintf("Deploy%sProxy.s.sol", contractName))
			} else {
				scriptPath = filepath.Join("script", "deploy", fmt.Sprintf("Deploy%s.s.sol", contractName))
			}
		}

		// Ensure script directory exists
		scriptDir := filepath.Dir(scriptPath)
		if err := os.MkdirAll(scriptDir, 0755); err != nil {
			return fmt.Errorf("failed to create script directory: %w", err)
		}

		// Check if script already exists
		if _, err := os.Stat(scriptPath); err == nil {
			if customPath == "" {
				return fmt.Errorf("script already exists: %s\nUse --script-path flag to specify a different location", scriptPath)
			}
			// With custom path, we allow overwriting
			fmt.Printf("Warning: Overwriting existing script at %s\n", scriptPath)
		}

		// Generate appropriate script
		var content string
		if isLibrary {
			content = generateLibraryScript(contractName, artifactPath)
		} else if useProxy {
			// Handle proxy deployment
			var proxyInfo *contracts.ContractInfo
			if proxyContract != "" {
				// Specific proxy provided
				proxyInfo, err = ctx.ResolveContract(proxyContract, types.AllContractsFilter())
				if err != nil {
					return fmt.Errorf("failed to resolve proxy contract: %w", err)
				}
			} else {
				// Interactive proxy selection
				proxies := indexer.GetProxyContracts()
				if len(proxies) == 0 {
					return fmt.Errorf("no proxy contracts found in project")
				}
				proxyInfo, err = interactive.SelectContract(proxies, "Select a proxy contract:")
				if err != nil {
					return err
				}
			}

			// Build proxy artifact path
			proxyArtifactPath := fmt.Sprintf("%s:%s", proxyInfo.Path, proxyInfo.Name)

			content = generateProxyScript(contractName, artifactPath, proxyInfo.Name, proxyInfo.Path, proxyArtifactPath, strategy)
		} else {
			// Regular contract deployment
			content = generateContractScript(contractName, artifactPath, strategy)
		}

		// Write script file
		if err := os.WriteFile(scriptPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write script: %w", err)
		}

		// Print success message
		fmt.Printf("\n✅ Generated deployment script: %s\n", scriptPath)
		if isLibrary {
			fmt.Println("\nThis library will be deployed with CREATE2 for deterministic addresses.")
		} else if useProxy {
			fmt.Println("\nThis script will deploy both the implementation and proxy contracts.")
			fmt.Println("Make sure to update the initializer parameters if needed.")
		}
		fmt.Println("\nTo deploy, run:")
		fmt.Printf("  treb run %s --network <network>\n", scriptPath)

		return nil
	},
}



func init() {
	// Add flags to genDeployCmd
	genDeployCmd.Flags().Bool("proxy", false, "Generate proxy deployment script")
	genDeployCmd.Flags().String("proxy-contract", "", "Specific proxy contract to use (optional)")
	genDeployCmd.Flags().String("strategy", "", "Deployment strategy: CREATE2 or CREATE3 (default: CREATE3)")
	genDeployCmd.Flags().String("script-path", "", "Custom path for the generated script")

	genCmd.AddCommand(genDeployCmd)
}

// extractContractName extracts the contract/library name from an artifact path
// e.g., "src/libs/MathUtils.sol:MathUtils" -> "MathUtils"
// e.g., "MathUtils" -> "MathUtils"
func extractContractName(artifactPath string) string {
	// If it contains ":", split and take the second part
	if strings.Contains(artifactPath, ":") {
		parts := strings.Split(artifactPath, ":")
		if len(parts) == 2 {
			return parts[1]
		}
		return ""
	}
	// Otherwise, assume it's just the contract name
	return artifactPath
}


// generateLibraryScript creates a library deployment script
func generateLibraryScript(libraryName string, artifactPath string) string {
	return fmt.Sprintf(`// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/src/TrebScript.sol";
import {Senders} from "treb-sol/src/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/src/internal/sender/Deployer.sol";

contract Deploy%s is TrebScript {
    using Senders for Senders.Sender;
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    /**
     * @custom:env {sender:optional} deployer
     * @custom:senders anvil
     */
    function run() public broadcast {
        Senders.Sender storage deployer = sender(
            vm.envOr("deployer", string("anvil"))
        );

        deployer.create2("%s").deploy();
    }
}
`, libraryName, artifactPath)
}

// generateContractScript creates a contract deployment script
func generateContractScript(contractName string, artifactPath string, strategy contracts.DeployStrategy) string {
	strategyMethod := "create3"
	if strategy == contracts.StrategyCreate2 {
		strategyMethod = "create2"
	}

	return fmt.Sprintf(`// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/src/TrebScript.sol";
import {Senders} from "treb-sol/src/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/src/internal/sender/Deployer.sol";

contract Deploy%s is TrebScript {
    using Senders for Senders.Sender;
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    /**
     * @custom:env {sender:optional} deployer
     * @custom:senders anvil
     */
    function run() public broadcast {
        Senders.Sender storage deployer = sender(
            vm.envOr("deployer", string("anvil"))
        );

        // Deploy %s
        deployer.%s("%s")
            .setLabel(vm.envOr("LABEL", string("")))
            .deploy(_getConstructorArgs());
    }

    function _getConstructorArgs() internal pure returns (bytes memory) {
        // TODO: Update constructor arguments as needed
        return "";
    }
}
`, contractName, contractName, strategyMethod, artifactPath)
}

// generateProxyScript creates a proxy deployment script
func generateProxyScript(implName string, implArtifact string, proxyName string, proxyPath string, proxyArtifact string, strategy contracts.DeployStrategy) string {
	strategyMethod := "create3"
	if strategy == contracts.StrategyCreate2 {
		strategyMethod = "create2"
	}

	return fmt.Sprintf(`// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {TrebScript} from "treb-sol/src/TrebScript.sol";
import {Senders} from "treb-sol/src/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/src/internal/sender/Deployer.sol";
import {%s} from "%s";

contract Deploy%sProxy is TrebScript {
    using Senders for Senders.Sender;
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    /**
     * @custom:env {sender:optional} deployer
     * @custom:env {string:optional} implementationLabel
     * @custom:env {string:optional} proxyLabel
     * @custom:senders anvil
     */
    function run() public broadcast {
        Senders.Sender storage deployer = sender(
            vm.envOr("deployer", string("anvil"))
        );

        // Deploy implementation
        address implementation = deployer
            .%s("%s")
            .setLabel(vm.envOr("implementationLabel", string("")))
            .deploy();

        // Deploy proxy
        deployer
            .%s("%s")
            .setLabel(vm.envOr("proxyLabel", string("%s")))
            .deploy(_getProxyConstructorArgs(implementation));
    }

    function _getProxyConstructorArgs(address implementation) internal pure returns (bytes memory) {
        // TODO: Update based on proxy type
        // For TransparentUpgradeableProxy:
        // return abi.encode(implementation, proxyAdmin, initData);
        
        // For UUPS/ERC1967 proxy:
        // return abi.encode(implementation, initData);
        
        bytes memory initData = _getInitializerData();
        return abi.encode(implementation, initData);
    }

    function _getInitializerData() internal pure returns (bytes memory) {
        // TODO: Update with initializer parameters
        // Example: return abi.encodeWithSignature("initialize(address)", owner);
        return "";
    }
}
`, proxyName, proxyPath, implName, strategyMethod, implArtifact, strategyMethod, proxyArtifact, implName)
}
