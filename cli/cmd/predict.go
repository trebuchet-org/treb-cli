package cmd

import (
	"fmt"

	"github.com/bogdan/fdeploy/cli/internal/forge"
	"github.com/bogdan/fdeploy/cli/internal/registry"
	"github.com/bogdan/fdeploy/cli/pkg/network"
	"github.com/spf13/cobra"
)

var predictCmd = &cobra.Command{
	Use:   "predict [contract]",
	Short: "Predict deployment addresses",
	Long: `Predict deployment addresses using Solidity scripts.

This command calls the PredictAddress.s.sol script to calculate
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
	defaultEnv, defaultNetwork, _, _ := GetConfiguredDefaults()
	
	// Create flags with defaults (empty if no config)
	predictCmd.Flags().StringVar(&env, "env", defaultEnv, "Deployment environment (staging/prod)")
	predictCmd.Flags().StringVar(&networkName, "network", defaultNetwork, "Network to predict for (defined in foundry.toml)")
	
	// Mark flags as required if they don't have defaults
	if defaultEnv == "" {
		predictCmd.MarkFlagRequired("env")
	}
	if defaultNetwork == "" {
		predictCmd.MarkFlagRequired("network")
	}
}

func predictAddress(contract string) error {
	// Resolve network configuration
	resolver := network.NewResolver(".")
	networkInfo, err := resolver.ResolveNetwork(networkName)
	if err != nil {
		return fmt.Errorf("failed to resolve network: %w", err)
	}

	// Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	// Initialize forge executor
	executor := forge.NewScriptExecutor("", ".", registryManager)

	// Predict address
	result, err := executor.PredictAddress(contract, env, forge.DeployArgs{
		RpcUrl:  networkInfo.RpcUrl,
		ChainID: networkInfo.ChainID,
	})
	if err != nil {
		return fmt.Errorf("address prediction failed: %w", err)
	}

	fmt.Printf("🔮 Address Prediction\n")
	fmt.Printf("📝 Contract: %s\n", contract)
	fmt.Printf("🏷️  Environment: %s\n", env)
	fmt.Printf("🌐 Network: %s (Chain ID: %d)\n", networkInfo.Name, networkInfo.ChainID)
	fmt.Printf("📍 Predicted Address: %s\n", result.Address.Hex())
	fmt.Printf("🧂 Salt: %x\n", result.Salt)
	
	// For CREATE3, init code hash is not used in address calculation
	if result.InitCodeHash == ([32]byte{}) {
		fmt.Printf("🔧 Init Code Hash: N/A (CREATE3)\n")
	} else {
		fmt.Printf("🔧 Init Code Hash: %x\n", result.InitCodeHash)
	}

	return nil
}