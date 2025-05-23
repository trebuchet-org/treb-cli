package cmd

import (
	"fmt"
	"strings"

	"github.com/bogdan/fdeploy/cli/internal/forge"
	"github.com/bogdan/fdeploy/cli/internal/registry"
	"github.com/bogdan/fdeploy/cli/pkg/config"
	"github.com/bogdan/fdeploy/cli/pkg/contracts"
	"github.com/bogdan/fdeploy/cli/pkg/interactive"
	"github.com/bogdan/fdeploy/cli/pkg/network"
	"github.com/bogdan/fdeploy/cli/pkg/types"
	forgeExec "github.com/bogdan/fdeploy/cli/pkg/forge"
	"github.com/spf13/cobra"
)

var (
	env         string
	verify      bool
	networkName string
	label       string
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
		
		result, err := deployContract(contract)
		if err != nil {
			checkError(err)
		}
		
		// Show final success message
		if result.AlreadyDeployed {
			fmt.Printf("\nContract %s was already deployed to %s environment\n", contract, env)
		} else {
			fmt.Printf("\nSuccessfully deployed %s to %s environment\n", contract, env)
		}
	},
}

func init() {
	// Get configured defaults (empty if no config file)
	_, defaultNetwork, defaultVerify, _ := GetConfiguredDefaults()
	
	// Create flags with defaults - env always defaults to "default"
	deployCmd.Flags().StringVar(&env, "env", "default", "Deployment environment")
	deployCmd.Flags().StringVar(&networkName, "network", defaultNetwork, "Network to deploy to (defined in foundry.toml)")
	deployCmd.Flags().BoolVar(&verify, "verify", defaultVerify, "Verify contract after deployment")
	deployCmd.Flags().StringVar(&label, "label", "", "Optional label for the deployment (included in salt)")
	
	// Mark network flag as required if it doesn't have a default
	if defaultNetwork == "" {
		deployCmd.MarkFlagRequired("network")
	}
}

func deployContract(contract string) (*types.DeploymentResult, error) {
	// Step 0: Load and validate deploy configuration
	deployConfig, err := config.LoadDeployConfig(".")
	if err != nil {
		return nil, fmt.Errorf("failed to load deploy configuration: %w", err)
	}

	if err := deployConfig.Validate(env); err != nil {
		return nil, fmt.Errorf("invalid deploy configuration for environment '%s': %w", env, err)
	}

	fmt.Printf("✓ Validated deploy configuration for environment: %s\n", env)

	// Step 1: Initialize forge executor and check installation
	forgeExecutor := forgeExec.NewExecutor(".")
	if err := forgeExecutor.CheckForgeInstallation(); err != nil {
		return nil, fmt.Errorf("forge check failed: %w", err)
	}

	// Step 2: Build contracts to ensure artifacts are up to date
	if err := forgeExecutor.Build(); err != nil {
		return nil, fmt.Errorf("build failed: %w", err)
	}

	// Step 3: Validate contract exists in src/
	validator := contracts.NewValidator(".")
	contractInfo, err := validator.ValidateContract(contract)
	if err != nil {
		return nil, fmt.Errorf("contract validation failed: %w", err)
	}

	if !contractInfo.Exists {
		return nil, fmt.Errorf("contract %s not found in src/ directory", contract)
	}

	fmt.Printf("Found contract: %s at %s\n", contract, contractInfo.SolidityFile)

	// Step 4: Check if deploy script exists, generate if needed
	if !validator.DeployScriptExists(contract) {
		fmt.Printf("Deploy script not found for %s\n", contract)
		
		// Ask if user wants to generate the script
		fmt.Printf("Would you like to generate a deploy script? (Y/n): ")
		var response string
		fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))
		
		if response == "n" || response == "no" {
			return nil, fmt.Errorf("deploy script required but not found: script/deploy/Deploy%s.s.sol", contract)
		}
		
		// Generate the script interactively
		fmt.Printf("Starting interactive script generation...\n\n")
		generator := interactive.NewGenerator(".")
		if err := generator.GenerateDeployScriptForContract(contract); err != nil {
			return nil, fmt.Errorf("script generation failed: %w", err)
		}
		return nil, nil
	}

	// Step 5: Resolve network configuration
	resolver := network.NewResolver(".")
	networkInfo, err := resolver.ResolveNetwork(networkName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve network: %w", err)
	}

	// Step 6: Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize registry: %w", err)
	}

	// Step 7: Generate environment variables for deployment
	envVars, err := deployConfig.GenerateEnvVars(env)
	if err != nil {
		return nil, fmt.Errorf("failed to generate environment variables: %w", err)
	}

	// Step 8: Initialize forge script executor
	executor := forge.NewScriptExecutor("", ".", registryManager)

	// Step 9: Deploy contract
	result, err := executor.Deploy(contract, env, forge.DeployArgs{
		RpcUrl:  networkInfo.RpcUrl,
		Verify:  verify,
		ChainID: networkInfo.ChainID,
		Label:   label,
		EnvVars: envVars,
	})
	if err != nil {
		return nil, fmt.Errorf("deployment failed: %w", err)
	}

	// Step 10: Display results
	fmt.Printf("\nDeployment Summary:\n")
	fmt.Printf("  Contract: %s\n", contract)
	fmt.Printf("  Environment: %s\n", env)
	fmt.Printf("  Network: %s (Chain ID: %d)\n", networkInfo.Name, networkInfo.ChainID)
	fmt.Printf("  Address: %s\n", result.Address.Hex())
	fmt.Printf("  Salt: %x\n", result.Salt)
	if result.TxHash.Hex() != "0x0000000000000000000000000000000000000000000000000000000000000000" {
		fmt.Printf("  Tx Hash: %s\n", result.TxHash.Hex())
		fmt.Printf("  Block: %d\n", result.BlockNumber)
	}

	return result, nil
}