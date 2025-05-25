package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/internal/registry"
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
		ctx := deployment.NewContext(deployment.TypeSingleton)
		ctx.ContractName = args[0]
		ctx.Env = env
		ctx.Label = label
		ctx.Predict = predict
		ctx.Debug = debug
		ctx.NetworkName = networkName

		if err := runDeployment(ctx); err != nil {
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
		ctx := deployment.NewContext(deployment.TypeProxy)
		ctx.ImplementationName = args[0]
		ctx.ProxyName = args[0] + "Proxy"
		ctx.ImplementationLabel = implementationLabel
		ctx.Env = env
		ctx.Label = label
		ctx.Predict = predict
		ctx.Debug = debug
		ctx.NetworkName = networkName

		if err := runDeployment(ctx); err != nil {
			checkError(err)
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
		ctx := deployment.NewContext(deployment.TypeLibrary)
		ctx.ContractName = args[0]
		ctx.Predict = predict
		ctx.Debug = debug
		ctx.NetworkName = networkName

		if err := runDeployment(ctx); err != nil {
			checkError(err)
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

	// Contract/Proxy specific flags
	deployContractCmd.Flags().StringVar(&env, "env", defaultEnv, "Deployment environment")
	deployContractCmd.Flags().StringVar(&label, "label", "", "Deployment label (affects address)")

	deployProxyCmd.Flags().StringVar(&env, "env", "default", "Deployment environment")
	deployProxyCmd.Flags().StringVar(&label, "label", "", "Deployment label (affects address)")
	deployProxyCmd.Flags().StringVar(&implementationLabel, "impl-label", "", "Implementation label to use")
}

func runDeployment(ctx *deployment.Context) error {
	display := deployment.NewDisplay()
	validator := deployment.NewValidator(".")

	// Validate deployment config first (non-interactive)
	s := display.CreateSpinner("Validating deployment configuration...")
	if err := validator.ValidateDeploymentConfig(ctx); err != nil {
		s.Stop()
		return err
	}
	s.Stop()
	display.PrintStep("Validated deployment configuration")
	
	// Build contracts
	s = display.CreateSpinner("Building contracts...")
	if err := validator.BuildContracts(); err != nil {
		s.Stop()
		return err
	}
	s.Stop()
	display.PrintStep("Built contracts")
	
	// Now do type-specific validation (may be interactive)
	var scriptGenerated bool
	switch ctx.Type {
	case deployment.TypeSingleton:
		generated, err := validator.ValidateContractWithGeneration(ctx)
		if err != nil {
			return err
		}
		scriptGenerated = generated
	case deployment.TypeProxy:
		if err := validator.ValidateProxyDeployment(ctx); err != nil {
			return err
		}
	case deployment.TypeLibrary:
		if err := validator.ValidateLibrary(ctx); err != nil {
			return err
		}
	}
	
	// If script was generated, restart the deployment
	if scriptGenerated {
		fmt.Println("\nRestarting deployment with generated script...")
		return runDeployment(ctx)
	}
	
	// Show deployment summary after validation
	display.PrintSummary(ctx)

	// Handle prediction mode
	if ctx.Predict {
		predictor := deployment.NewPredictor(".")

		// Show script path for debugging
		if ctx.Debug {
			fmt.Printf("\n[%s]\n\n", ctx.ScriptPath)
		}

		s := display.CreateSpinner("Calculating deployment address...")
		result, err := predictor.Predict(ctx)
		if err != nil {
			s.Stop()
			return err
		}
		s.Stop()

		// Check if already deployed
		registryPath := filepath.Join(".", "deployments.json")
		registryManager, err := registry.NewManager(registryPath)
		if err == nil {
			if existing := predictor.GetExistingAddress(ctx, registryManager); existing != (common.Address{}) {
				return fmt.Errorf("contract already deployed at %s", existing.Hex())
			}
		}

		display.ShowPrediction(ctx, result)
		return nil
	}

	// Execute deployment
	executor := deployment.NewExecutor(".")

	s = display.CreateSpinner("Executing deployment script...")
	result, err := executor.Execute(ctx)
	if err != nil {
		s.Stop()
		return err
	}
	s.Stop()

	// Show success
	display.ShowSuccess(ctx, result)

	return nil
}
