package cmd

import (
	"fmt"
	"strings"

	"github.com/bogdan/fdeploy/cli/internal/forge"
	"github.com/bogdan/fdeploy/cli/internal/registry"
	"github.com/bogdan/fdeploy/cli/pkg/contracts"
	"github.com/bogdan/fdeploy/cli/pkg/network"
	forgeExec "github.com/bogdan/fdeploy/cli/pkg/forge"
	"github.com/spf13/cobra"
)

var (
	env         string
	verify      bool
	networkName string
)

var deployCmd = &cobra.Command{
	Use:   "deploy [contract]",
	Short: "Deploy contracts via Foundry scripts",
	Long: `Deploy contracts using Foundry scripts with enhanced tracking.

This command:
- Predicts deployment addresses
- Checks for existing deployments
- Executes forge script for deployment
- Records deployment in registry
- Handles verification if requested`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		contract := args[0]
		
		if err := deployContract(contract); err != nil {
			checkError(err)
		}
		
		fmt.Printf("✅ Deployed %s to %s environment\n", contract, env)
	},
}

func init() {
	// Get configured defaults (empty if no config file)
	defaultEnv, defaultNetwork, defaultVerify, _ := GetConfiguredDefaults()
	
	// Create flags with defaults (empty if no config)
	deployCmd.Flags().StringVar(&env, "env", defaultEnv, "Deployment environment (staging/prod)")
	deployCmd.Flags().StringVar(&networkName, "network", defaultNetwork, "Network to deploy to (defined in foundry.toml)")
	deployCmd.Flags().BoolVar(&verify, "verify", defaultVerify, "Verify contract after deployment")
	
	// Mark flags as required if they don't have defaults
	if defaultEnv == "" {
		deployCmd.MarkFlagRequired("env")
	}
	if defaultNetwork == "" {
		deployCmd.MarkFlagRequired("network")
	}
}

func deployContract(contract string) error {
	// Step 0: Initialize forge executor and check installation
	forgeExecutor := forgeExec.NewExecutor(".")
	if err := forgeExecutor.CheckForgeInstallation(); err != nil {
		return fmt.Errorf("forge check failed: %w", err)
	}

	// Step 1: Build contracts to ensure artifacts are up to date
	if err := forgeExecutor.Build(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// Step 2: Contract strategy will be determined by existing script or generation prompt

	// Step 3: Validate contract exists in src/
	validator := contracts.NewValidator(".")
	contractInfo, err := validator.ValidateContract(contract)
	if err != nil {
		return fmt.Errorf("contract validation failed: %w", err)
	}

	if !contractInfo.Exists {
		return fmt.Errorf("contract %s not found in src/ directory", contract)
	}

	fmt.Printf("✅ Found contract: %s at %s\n", contract, contractInfo.SolidityFile)

	// Step 4: Check if deploy script exists, generate if needed
	if !validator.DeployScriptExists(contract) {
		fmt.Printf("📋 Deploy script not found for %s\n", contract)
		
		// Ask if user wants to generate the script
		fmt.Printf("❓ Would you like to generate a deploy script? (Y/n): ")
		var response string
		fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))
		
		if response == "n" || response == "no" {
			return fmt.Errorf("deploy script required but not found: script/Deploy%s.s.sol", contract)
		}
		
		fmt.Printf("🚀 Use 'fdeploy generate deploy' for interactive script generation with strategy selection\n")
		return fmt.Errorf("deploy script required but not found. Please generate one first using: fdeploy generate deploy")
	} else {
		fmt.Printf("📋 Using existing deploy script: Deploy%s.s.sol\n", contract)
	}

	// Step 5: Resolve network configuration
	resolver := network.NewResolver(".")
	networkInfo, err := resolver.ResolveNetwork(networkName)
	if err != nil {
		return fmt.Errorf("failed to resolve network: %w", err)
	}

	// Step 6: Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	// Step 7: Initialize forge script executor
	executor := forge.NewScriptExecutor("", ".", registryManager)

	// Step 8: Deploy contract
	result, err := executor.Deploy(contract, env, forge.DeployArgs{
		RpcUrl:  networkInfo.RpcUrl,
		Verify:  verify,
		ChainID: networkInfo.ChainID,
	})
	if err != nil {
		return fmt.Errorf("deployment failed: %w", err)
	}

	// Step 9: Display results
	fmt.Printf("📝 Contract: %s\n", contract)
	fmt.Printf("🏷️  Environment: %s\n", env)
	fmt.Printf("🌐 Network: %s (Chain ID: %d)\n", networkInfo.Name, networkInfo.ChainID)
	fmt.Printf("📍 Address: %s\n", result.Address.Hex())
	fmt.Printf("🧂 Salt: %x\n", result.Salt)
	fmt.Printf("🔍 Tx Hash: %s\n", result.TxHash.Hex())
	fmt.Printf("📊 Block: %d\n", result.BlockNumber)

	return nil
}