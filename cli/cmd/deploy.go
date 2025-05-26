package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/deployment"
)

var (
	env                 string
	networkName         string
	label               string
	debug               bool
	implementationLabel string
	predict             bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy contracts and proxies",
	Long: `Deploy contracts using Foundry scripts. Supports singleton contracts,
proxies, and libraries with deterministic addresses via CreateX.

Examples:
  treb deploy contract Counter --network sepolia
  treb deploy proxy Counter --network mainnet
  treb deploy library MathLib --network sepolia`,
}

var deployContractCmd = &cobra.Command{
	Use:   "contract <name>",
	Short: "Deploy a singleton contract",
	Long: `Deploy a singleton contract using Foundry scripts.

Examples:
  treb deploy contract Counter
  treb deploy contract Token --env staging --label v2
  treb deploy contract src/Counter.sol:Counter --network sepolia`,
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"singleton"},
	Run: func(cmd *cobra.Command, args []string) {
		ctx, err := deployment.NewContext(deployment.DeploymentParams{
			DeploymentType: deployment.TypeSingleton,
			ContractQuery:  args[0],
			Env:            env,
			Label:          label,
			Predict:        predict,
			Debug:          debug,
			NetworkName:    networkName,
		})
		if err != nil {
			checkError(err)
		}

		if err = runDeployment(ctx); err != nil {
			checkError(err)
		}
	},
}

var deployProxyCmd = &cobra.Command{
	Use:   "proxy <contract>",
	Short: "Deploy a proxy for an implementation",
	Long: `Deploy proxy contracts using Foundry scripts. The implementation must be
deployed first. Supports UUPS, Transparent, and custom proxy patterns.

Examples:
  treb deploy proxy Counter
  treb deploy proxy Token --impl-label v1`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if ctx, err := deployment.NewContext(deployment.DeploymentParams{
			DeploymentType: deployment.TypeProxy,
			ContractQuery:  args[0],
			Env:            env,
			Label:          label,
			Predict:        predict,
			Debug:          debug,
			NetworkName:    networkName,
		}); err != nil {
			checkError(err)
		} else {
			if err := runDeployment(ctx); err != nil {
				checkError(err)
			}
		}
	},
}

var deployLibraryCmd = &cobra.Command{
	Use:   "library <name>",
	Short: "Deploy a library",
	Long: `Deploy a library globally (no environment) for cross-chain consistency.
Libraries are deployed using the default or "libraries" environment's deployer configuration.

Examples:
  treb deploy library MathLib --network sepolia
  treb deploy library StringUtils --network mainnet`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if ctx, err := deployment.NewContext(deployment.DeploymentParams{
			DeploymentType: deployment.TypeLibrary,
			ContractQuery:  args[0],
			Label:          label,
			Predict:        predict,
			Debug:          debug,
			NetworkName:    networkName,
		}); err != nil {
			checkError(err)
		} else {
			if err := runDeployment(ctx); err != nil {
				checkError(err)
			}
		}
	},
}

func init() {
	deployCmd.AddCommand(deployContractCmd)
	deployCmd.AddCommand(deployProxyCmd)
	deployCmd.AddCommand(deployLibraryCmd)

	// Make contract the default subcommand
	deployCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		// Treat as contract deployment
		deployContractCmd.Run(cmd, args)
		return nil
	}

	// Load config defaults
	var defaultNetwork string
	var defaultEnv string
	configManager := config.NewManager(".")
	if configManager.Exists() {
		if cfg, err := configManager.Load(); err == nil && cfg.Network != "" {
			defaultNetwork = cfg.Network
			if cfg.Environment == "" {
				defaultEnv = "default"
			} else {
				defaultEnv = cfg.Environment
			}
		}
	}

	// Global flags
	deployCmd.PersistentFlags().StringVar(&networkName, "network", defaultNetwork, "Network to deploy to")
	if defaultNetwork == "" {
		deployCmd.MarkPersistentFlagRequired("network")
	}
	deployCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Show detailed debug output")
	deployCmd.PersistentFlags().BoolVar(&predict, "predict", false, "Predict deployment address without deploying")

	// Contract/Proxy specific flags (also add to main deployCmd for bare usage)
	deployCmd.PersistentFlags().StringVar(&env, "env", defaultEnv, "Deployment environment")
	deployCmd.PersistentFlags().StringVar(&label, "label", "", "Deployment label (affects address)")

	deployContractCmd.Flags().StringVar(&env, "env", defaultEnv, "Deployment environment")
	deployContractCmd.Flags().StringVar(&label, "label", "", "Deployment label (affects address)")

	deployProxyCmd.Flags().StringVar(&env, "env", "default", "Deployment environment")
	deployProxyCmd.Flags().StringVar(&label, "label", "", "Deployment label (affects address)")
	deployProxyCmd.Flags().StringVar(&implementationLabel, "impl-label", "", "Implementation label to use")
}

func runDeployment(ctx *deployment.DeploymentContext) error {
	// Validate deployment config first (non-interactive)
	s := ctx.CreateSpinner("Validating deployment configuration...")
	if err := ctx.ValidateDeploymentConfig(); err != nil {
		s.Stop()
		return err
	}
	s.Stop()
	ctx.PrintStep("Validated deployment configuration")

	// Build contracts
	s = ctx.CreateSpinner("Building contracts...")
	if err := ctx.BuildContracts(); err != nil {
		s.Stop()
		return err
	}
	s.Stop()
	ctx.PrintStep("Built contracts")

	// Now do type-specific validation (may be interactive)
	switch ctx.Params.DeploymentType {
	case deployment.TypeSingleton:
		generated, err := ctx.PrepareContractDeployment()
		if err != nil {
			return err
		}
		if generated {
			fmt.Println("\nScript generated, update it if necessary and run again.")
			return nil
		}
	case deployment.TypeProxy:
		if err := ctx.PrepareProxyDeployment(); err != nil {
			return err
		}
	case deployment.TypeLibrary:
		if err := ctx.PrepareLibraryDeployment(); err != nil {
			return err
		}
	}

	// Show deployment summary after validation
	ctx.PrintSummary()

	// Handle prediction mode
	if ctx.Params.Predict {
		return predictDeployment(ctx)
	} else {
		return executeDeployment(ctx)
	}

}

func executeDeployment(ctx *deployment.DeploymentContext) error {
	s := ctx.CreateSpinner("Executing deployment script...")
	result, err := ctx.Execute()
	if err != nil {
		s.Stop()
		return err
	}
	s.Stop()

	// Show success
	ctx.ShowSuccess(result)

	return nil
}

func predictDeployment(ctx *deployment.DeploymentContext) error {
	// Show script path for debugging
	if ctx.Params.Debug {
		fmt.Printf("\n[%s]\n\n", ctx.ScriptPath)
	}

	s := ctx.CreateSpinner("Calculating deployment address...")
	result, err := ctx.Predict()
	if err != nil {
		s.Stop()
		return err
	}
	s.Stop()

	ctx.ShowPrediction(result)
	return nil
}
