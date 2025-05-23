package cmd

import (
	"fmt"

	"github.com/bogdan/fdeploy/cli/internal/forge"
	"github.com/bogdan/fdeploy/cli/internal/registry"
	"github.com/spf13/cobra"
)

var (
	env      string
	verify   bool
	networks []string
	rpcUrl   string
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
	deployCmd.Flags().StringVar(&env, "env", "staging", "Deployment environment (staging/prod)")
	deployCmd.Flags().BoolVar(&verify, "verify", false, "Verify contract after deployment")
	deployCmd.Flags().StringSliceVar(&networks, "networks", []string{}, "Deploy to multiple networks")
	deployCmd.Flags().StringVar(&rpcUrl, "rpc-url", "", "RPC URL (overrides foundry.toml)")
}

func deployContract(contract string) error {
	// Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	// Initialize forge executor
	executor := forge.NewScriptExecutor("", ".", registryManager)

	// Deploy contract
	result, err := executor.Deploy(contract, env, forge.DeployArgs{
		RpcUrl:    rpcUrl,
		Verify:    verify,
		Networks:  networks,
	})
	if err != nil {
		return fmt.Errorf("deployment failed: %w", err)
	}

	fmt.Printf("📝 Contract: %s\n", contract)
	fmt.Printf("🏷️  Environment: %s\n", env)
	fmt.Printf("📍 Address: %s\n", result.Address.Hex())
	fmt.Printf("🧂 Salt: %x\n", result.Salt)
	fmt.Printf("🔍 Tx Hash: %s\n", result.TxHash.Hex())
	fmt.Printf("📊 Block: %d\n", result.BlockNumber)

	return nil
}