package cmd

import (
	"fmt"

	"github.com/bogdan/fdeploy/cli/internal/forge"
	"github.com/bogdan/fdeploy/cli/internal/registry"
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

func predictAddress(contract string) error {
	// Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	// Initialize forge executor
	executor := forge.NewScriptExecutor("", ".", registryManager)

	// Predict address
	result, err := executor.PredictAddress(contract, env, forge.DeployArgs{})
	if err != nil {
		return fmt.Errorf("address prediction failed: %w", err)
	}

	fmt.Printf("🔮 Address Prediction\n")
	fmt.Printf("📝 Contract: %s\n", contract)
	fmt.Printf("🏷️  Environment: %s\n", env)
	fmt.Printf("📍 Predicted Address: %s\n", result.Address.Hex())
	fmt.Printf("🧂 Salt: %x\n", result.Salt)
	fmt.Printf("🔧 Init Code Hash: %x\n", result.InitCodeHash)

	return nil
}