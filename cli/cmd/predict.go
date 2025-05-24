package cmd

import (
	"fmt"

	"github.com/trebuchet-org/treb-cli/cli/internal/forge"
	"github.com/trebuchet-org/treb-cli/cli/internal/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
	forgeExec "github.com/trebuchet-org/treb-cli/cli/pkg/forge"
	"github.com/spf13/cobra"
)

var predictCmd = &cobra.Command{
	Use:   "predict [contract]",
	Short: "Predict deployment addresses",
	Long: `Predict deployment addresses using the generated deployment scripts.

This command uses the contract's deployment script predictAddress() function to calculate
deterministic addresses based on salt components before deployment.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		contract := args[0]
		
		if err := predictAddress(contract); err != nil {
			checkError(err)
		}
	},
}

func init() {
	// Get configured defaults (empty if no config file)
	_, defaultNetwork, _, _ := GetConfiguredDefaults()
	
	// Create flags with defaults - env always defaults to "default"
	predictCmd.Flags().StringVar(&env, "env", "default", "Deployment environment")
	predictCmd.Flags().StringVar(&networkName, "network", defaultNetwork, "Network to predict for (defined in foundry.toml)")
	predictCmd.Flags().StringVar(&label, "label", "", "Optional label for the deployment (included in salt)")
	predictCmd.Flags().BoolVar(&debug, "debug", false, "Show full Foundry script output")
	
	// Mark network flag as required if it doesn't have a default
	if defaultNetwork == "" {
		predictCmd.MarkFlagRequired("network")
	}
}

func predictAddress(contract string) error {
	// Step 0: Load and validate deploy configuration
	deployConfig, err := config.LoadDeployConfig(".")
	if err != nil {
		return fmt.Errorf("failed to load deploy configuration: %w", err)
	}

	if err := deployConfig.Validate(env); err != nil {
		return fmt.Errorf("invalid deploy configuration for environment '%s': %w", env, err)
	}

	fmt.Printf("✓ Validated deploy configuration for environment: %s\n", env)

	// Step 1: Initialize forge executor and check installation
	forgeExecutor := forgeExec.NewExecutor(".")
	if err := forgeExecutor.CheckForgeInstallation(); err != nil {
		return fmt.Errorf("forge check failed: %w", err)
	}

	// Step 2: Build contracts to ensure artifacts are up to date
	if err := forgeExecutor.Build(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// Step 3: Validate contract exists in src/
	validator := contracts.NewValidator(".")
	contractInfo, err := validator.ValidateContract(contract)
	if err != nil {
		return fmt.Errorf("contract validation failed: %w", err)
	}

	if !contractInfo.Exists {
		return fmt.Errorf("contract %s not found in src/ directory", contract)
	}

	fmt.Printf("Found contract: %s at %s\n", contract, contractInfo.SolidityFile)

	// Step 4: Check if deploy script exists
	if !validator.DeployScriptExists(contract) {
		return fmt.Errorf("deploy script required for prediction but not found: script/deploy/Deploy%s.s.sol", contract)
	}

	// Step 5: Generate environment variables for prediction
	envVars, err := deployConfig.GenerateEnvVars(env)
	if err != nil {
		return fmt.Errorf("failed to generate environment variables: %w", err)
	}

	// Step 6: Resolve network configuration
	resolver := network.NewResolver(".")
	networkInfo, err := resolver.ResolveNetwork(networkName)
	if err != nil {
		return fmt.Errorf("failed to resolve network: %w", err)
	}

	// Step 7: Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	// Step 8: Initialize forge executor
	executor := forge.NewScriptExecutor("", ".", registryManager)

	// Step 9: Predict address using the new method
	result, err := executor.PredictAddress(contract, env, forge.DeployArgs{
		RpcUrl:  networkInfo.RpcUrl,
		ChainID: networkInfo.ChainID,
		Label:   label,
		EnvVars: envVars,
		Debug:   debug,
	})
	if err != nil {
		return fmt.Errorf("address prediction failed: %w", err)
	}

	fmt.Printf("\n🔮 Address Prediction\n")
	fmt.Printf("📝 Contract: %s\n", contract)
	fmt.Printf("🏷️  Environment: %s\n", env)
	if label != "" {
		fmt.Printf("🏷️  Label: %s\n", label)
	}
	fmt.Printf("🌐 Network: %s (Chain ID: %d)\n", networkInfo.Name, networkInfo.ChainID)
	fmt.Printf("📍 Predicted Address: %s\n", result.Address.Hex())
	
	// Only show salt and init code hash if they're available
	if result.Salt != ([32]byte{}) {
		fmt.Printf("🧂 Salt: %x\n", result.Salt)
	}
	if result.InitCodeHash != ([32]byte{}) {
		fmt.Printf("🔧 Init Code Hash: %x\n", result.InitCodeHash)
	}

	return nil
}