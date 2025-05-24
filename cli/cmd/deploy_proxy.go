package cmd

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/trebuchet-org/treb-cli/cli/internal/forge"
	"github.com/trebuchet-org/treb-cli/cli/internal/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
	"github.com/spf13/cobra"
)

var (
	implementationLabel string
)

var deployProxyCmd = &cobra.Command{
	Use:   "deploy-proxy [implementation-contract]",
	Short: "Deploy proxy contracts via Foundry scripts",
	Long: `Deploy proxy contracts using Foundry scripts with enhanced tracking.

This command:
- Validates proxy deployment script exists
- Sets implementation label if specified
- Executes forge script for proxy deployment
- Records both proxy and implementation addresses in registry
- Handles verification if requested`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		implementationContract := args[0]
		proxyContract := implementationContract + "Proxy"
		
		result, err := deployProxyContract(implementationContract, proxyContract)
		if err != nil {
			checkError(err)
		}
		
		// Show final success message only if we have a result
		if result != nil {
			if result.AlreadyDeployed {
				fmt.Printf("\nProxy %s was already deployed to %s environment\n", proxyContract, env)
			} else {
				fmt.Printf("\n✅ Proxy deployment successful!\n")
				fmt.Printf("   Implementation: %s\n", implementationContract)
				if implementationLabel != "" {
					fmt.Printf("   Implementation Label: %s\n", implementationLabel)
				}
			}
		}
	},
}

func init() {
	deployProxyCmd.Flags().StringVarP(&env, "env", "e", "default", "Environment to deploy to")
	deployProxyCmd.Flags().BoolVarP(&verify, "verify", "v", true, "Verify on Etherscan after deployment")
	deployProxyCmd.Flags().StringVarP(&networkName, "network", "n", "", "Network name (if not set, will use --rpc-url or prompt)")
	deployProxyCmd.Flags().StringVarP(&label, "label", "l", "", "Deployment label for the proxy (optional)")
	deployProxyCmd.Flags().StringVar(&implementationLabel, "impl-label", "", "Implementation label to point to (optional)")
	deployProxyCmd.Flags().BoolVar(&debug, "debug", false, "Show debug output including full forge output")
	
	rootCmd.AddCommand(deployProxyCmd)
}

func deployProxyContract(implementationContract, proxyContract string) (*types.DeploymentResult, error) {
	fmt.Printf("🚀 Deploying proxy for %s\n", implementationContract)
	fmt.Println("================================")
	
	// Step 1: Load deploy configuration
	deployConfig, err := config.LoadDeployConfig(".")
	if err != nil {
		return nil, fmt.Errorf("failed to load deploy config: %w", err)
	}
	
	// Step 2: Validate environment configuration
	if err := deployConfig.Validate(env); err != nil {
		return nil, fmt.Errorf("invalid deploy configuration for environment '%s': %w", env, err)
	}
	
	fmt.Printf("✓ Validated deploy configuration for environment: %s\n", env)
	
	// Step 3: Check if proxy deploy script exists
	validator := contracts.NewValidator(".")
	if !validator.DeployScriptExists(proxyContract) {
		return nil, fmt.Errorf("proxy deploy script not found: script/deploy/Deploy%s.s.sol\nPlease run 'treb generate deploy-proxy' first", proxyContract)
	}
	
	fmt.Printf("Found proxy deploy script: script/deploy/Deploy%s.s.sol\n", proxyContract)
	
	// Step 4: Resolve network configuration
	resolver := network.NewResolver(".")
	networkInfo, err := resolver.ResolveNetwork(networkName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve network: %w", err)
	}
	
	// Step 5: Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize registry: %w", err)
	}
	
	// Step 6: Check if implementation is deployed
	var implDeployment *types.DeploymentEntry
	if implementationLabel != "" {
		implDeployment = registryManager.GetDeploymentWithLabel(implementationContract, env, implementationLabel, networkInfo.ChainID)
		if implDeployment == nil {
			return nil, fmt.Errorf("implementation %s with label '%s' not found in %s environment on %s", 
				implementationContract, implementationLabel, env, networkInfo.Name)
		}
		fmt.Printf("Found implementation at: %s (label: %s)\n", implDeployment.Address.Hex(), implementationLabel)
	} else {
		// Check for default implementation
		implDeployment = registryManager.GetDeployment(implementationContract, env, networkInfo.ChainID)
		if implDeployment == nil {
			return nil, fmt.Errorf("implementation %s not found in %s environment on %s\nPlease deploy the implementation first with 'treb deploy %s'", 
				implementationContract, env, networkInfo.Name, implementationContract)
		}
		fmt.Printf("Found implementation at: %s\n", implDeployment.Address.Hex())
	}
	
	// Step 7: Check for existing proxy deployment
	existingProxy := registryManager.GetDeploymentWithLabel(proxyContract, env, label, networkInfo.ChainID)
	if existingProxy != nil {
		fmt.Printf("\n⚠️  Proxy already deployed!\n")
		fmt.Printf("   Contract: %s\n", proxyContract)
		fmt.Printf("   Environment: %s\n", env)
		fmt.Printf("   Network: %s\n", networkInfo.Name)
		fmt.Printf("   Address: %s\n", existingProxy.Address.Hex())
		if label != "" {
			fmt.Printf("   Label: %s\n", label)
		}
		
		// Convert hex string salt to [32]byte
		var salt [32]byte
		if saltBytes, err := hex.DecodeString(existingProxy.Salt); err == nil && len(saltBytes) == 32 {
			copy(salt[:], saltBytes)
		}
		
		return &types.DeploymentResult{
			AlreadyDeployed: true,
			Address:         existingProxy.Address,
			Salt:            salt,
		}, nil
	}
	
	// Step 8: Generate environment variables for deployment
	envVars, err := deployConfig.GenerateEnvVars(env)
	if err != nil {
		return nil, fmt.Errorf("failed to generate environment variables: %w", err)
	}
	
	// Add implementation label if specified
	if implementationLabel != "" {
		envVars["IMPLEMENTATION_LABEL"] = implementationLabel
	}
	
	// Step 9: Initialize forge script executor with proxy support
	executor := forge.NewScriptExecutor("", ".", registryManager)
	
	// Step 10: Deploy proxy
	result, err := executor.DeployProxy(proxyContract, implementationContract, env, forge.DeployArgs{
		RpcUrl:  networkInfo.RpcUrl,
		Verify:  verify,
		ChainID: networkInfo.ChainID,
		Label:   label,
		EnvVars: envVars,
		Debug:   debug,
	})
	if err != nil {
		return nil, fmt.Errorf("proxy deployment failed: %w", err)
	}
	
	// Step 11: Display results
	fmt.Printf("\nProxy Deployment Summary:\n")
	fmt.Printf("  Proxy Contract: %s\n", proxyContract)
	fmt.Printf("  Implementation: %s\n", implementationContract)
	if implementationLabel != "" {
		fmt.Printf("  Implementation Label: %s\n", implementationLabel)
	}
	fmt.Printf("  Environment: %s\n", env)
	fmt.Printf("  Network: %s (Chain ID: %d)\n", networkInfo.Name, networkInfo.ChainID)
	fmt.Printf("  Proxy Address: %s\n", result.Address.Hex())
	fmt.Printf("  Implementation Address: %s\n", implDeployment.Address.Hex())
	fmt.Printf("  Salt: %x\n", result.Salt)
	if result.TxHash.Hex() != "0x0000000000000000000000000000000000000000000000000000000000000000" {
		fmt.Printf("  Tx Hash: %s\n", result.TxHash.Hex())
		fmt.Printf("  Block: %d\n", result.BlockNumber)
	}
	
	return result, nil
}

// validateProxyDeployment performs proxy-specific validation
func validateProxyDeployment(implementationContract, proxyContract string) error {
	// Check if proxy deploy script exists
	scriptPath := fmt.Sprintf("script/deploy/Deploy%s.s.sol", proxyContract)
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("proxy deploy script not found: %s", scriptPath)
	}
	
	return nil
}