package cmd

import (
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/trebuchet-org/treb-cli/cli/internal/forge"
	"github.com/trebuchet-org/treb-cli/cli/internal/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/cli/pkg/interactive"
	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
	forgeExec "github.com/trebuchet-org/treb-cli/cli/pkg/forge"
	"github.com/briandowns/spinner"
	"github.com/ethereum/go-ethereum/common"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	env                 string
	verify              bool
	networkName         string
	label               string
	debug               bool
	implementationLabel string
)

var deployCmd = &cobra.Command{
	Use:   "deploy <contract>",
	Short: "Deploy a contract",
	Long: `Deploy contracts using Foundry scripts with automatic registry tracking.
Supports EOA and Safe multisig deployments.

Examples:
  treb deploy Counter --network sepolia
  treb deploy Token --env production --verify`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		contract := args[0]
		
		result, err := deployContract(contract)
		if err != nil {
			checkError(err)
		}
		
		// Show final success message only if we have a result
		if result != nil && !result.AlreadyDeployed {
			// Resolve network info for display
			resolver := network.NewResolver(".")
			netInfo, _ := resolver.ResolveNetwork(networkName)
			isSafe := result.SafeTxHash != (common.Hash{})
			showDeploymentSummary(contract, env, netInfo.Name, result, false, isSafe)
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
		implementationContract := args[0]
		proxyContract := implementationContract + "Proxy"
		
		result, err := deployProxyContract(implementationContract, proxyContract)
		if err != nil {
			checkError(err)
		}
		
		// Show final success message only if we have a result
		if result != nil && !result.AlreadyDeployed {
			// Resolve network info for display
			resolver := network.NewResolver(".")
			netInfo, _ := resolver.ResolveNetwork(networkName)
			isSafe := result.SafeTxHash != (common.Hash{})
			showDeploymentSummary(proxyContract, env, netInfo.Name, result, true, isSafe)
		}
	},
}

var deployPredictCmd = &cobra.Command{
	Use:   "predict <contract>",
	Short: "Predict deployment address",
	Long: `Calculate deterministic deployment addresses before deploying.
Useful for pre-funding accounts or configuration.

Examples:
  treb deploy predict Counter
  treb deploy predict Token --label v2`,
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
	_, defaultNetwork, defaultVerify, _ := GetConfiguredDefaults()
	
	// Add subcommands
	deployCmd.AddCommand(deployProxyCmd)
	deployCmd.AddCommand(deployPredictCmd)
	
	// Create flags for main deploy command with defaults - env always defaults to "default"
	deployCmd.PersistentFlags().StringVar(&env, "env", "default", "Deployment environment")
	deployCmd.PersistentFlags().StringVar(&networkName, "network", defaultNetwork, "Network to deploy to (defined in foundry.toml)")
	deployCmd.PersistentFlags().BoolVar(&verify, "verify", defaultVerify, "Verify contract after deployment")
	deployCmd.PersistentFlags().StringVar(&label, "label", "", "Optional label for the deployment (included in salt)")
	deployCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Show full Foundry script output")
	
	// Proxy-specific flags
	deployProxyCmd.Flags().StringVar(&implementationLabel, "impl-label", "", "Implementation label to point to (optional)")
	
	// Mark network flag as required if it doesn't have a default
	if defaultNetwork == "" {
		deployCmd.MarkPersistentFlagRequired("network")
	}
}

func deployContract(contract string) (*types.DeploymentResult, error) {
	// Print initial deployment summary
	identifier := fmt.Sprintf("%s/%s", env, contract)
	if label != "" {
		identifier += fmt.Sprintf(":%s", label)
	}
	
	// Resolve network first for the summary
	resolver := network.NewResolver(".")
	networkInfo, err := resolver.ResolveNetwork(networkName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve network: %w", err)
	}
	
	fmt.Println()
	color.New(color.FgWhite).Printf("Deploying ")
	color.New(color.FgWhite, color.Bold).Printf("%s", identifier)
	color.New(color.FgWhite).Printf(" to ")
	color.New(color.FgMagenta, color.Bold).Printf("%s\n", networkInfo.Name)
	fmt.Println()
	
	// Step 0: Load and validate deploy configuration
	s := createSpinner("Validating deployment configuration...")
	deployConfig, err := config.LoadDeployConfig(".")
	if err != nil {
		s.Stop()
		return nil, fmt.Errorf("failed to load deploy configuration: %w", err)
	}

	if err := deployConfig.Validate(env); err != nil {
		s.Stop()
		return nil, fmt.Errorf("invalid deploy configuration for environment '%s': %w", env, err)
	}
	s.Stop()
	color.New(color.FgGreen).Printf("✓ ")
	fmt.Printf("Validated deployment configuration\n")

	// Step 1: Initialize forge executor and check installation
	forgeExecutor := forgeExec.NewExecutor(".")
	if err := forgeExecutor.CheckForgeInstallation(); err != nil {
		return nil, fmt.Errorf("forge check failed: %w", err)
	}

	// Step 2: Build contracts to ensure artifacts are up to date
	s = createSpinner("Building contracts...")
	if err := forgeExecutor.Build(); err != nil {
		s.Stop()
		return nil, fmt.Errorf("build failed: %w", err)
	}
	s.Stop()
	color.New(color.FgGreen).Printf("✓ ")
	fmt.Printf("Built contracts\n")

	// Step 3: Validate contract exists in src/
	validator := contracts.NewValidator(".")
	contractInfo, err := validator.ValidateContract(contract)
	if err != nil {
		return nil, fmt.Errorf("contract validation failed: %w", err)
	}

	if !contractInfo.Exists {
		return nil, fmt.Errorf("contract %s not found in src/ directory", contract)
	}

	// Step 4: Check if deploy script exists, generate if needed
	if !validator.DeployScriptExists(contract) {
		fmt.Printf("\nDeploy script not found for %s\n", contract)
		
		// Ask if user wants to generate the script
		selector := interactive.NewSelector()
		shouldGenerate, err := selector.PromptConfirm("Would you like to generate a deploy script?", true)
		if err != nil || !shouldGenerate {
			return nil, fmt.Errorf("deploy script required but not found: script/deploy/Deploy%s.s.sol", contract)
		}
		
		// Generate the script interactively
		fmt.Printf("\nStarting interactive script generation...\n\n")
		generator := interactive.NewGenerator(".")
		if err := generator.GenerateDeployScript(contract); err != nil {
			return nil, fmt.Errorf("script generation failed: %w", err)
		}
		return nil, nil
	}

	// Step 5: Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize registry: %w", err)
	}

	// Step 6: Generate environment variables for deployment
	envVars, err := deployConfig.GenerateEnvVars(env)
	if err != nil {
		return nil, fmt.Errorf("failed to generate environment variables: %w", err)
	}

	// Step 7: Initialize forge script executor
	executor := forge.NewScriptExecutor("", ".", registryManager)

	// Step 8: Deploy contract with spinner updates
	scriptPath := fmt.Sprintf("script/deploy/Deploy%s.s.sol", contract)
	color.New(color.FgCyan).Printf("\n[%s]\n\n", scriptPath)
	
	s = createSpinner("Predicting deployment address...")
	predictResult, err := executor.PredictAddress(contract, env, forge.DeployArgs{
		RpcUrl:  networkInfo.RpcUrl,
		Label:   label,
		EnvVars: envVars,
	})
	if err != nil {
		s.Stop()
		return nil, fmt.Errorf("address prediction failed: %w", err)
	}
	s.Stop()
	color.New(color.FgGreen).Printf("✓ ")
	fmt.Printf("Predicted address: ")
	color.New(color.FgCyan).Printf("%s\n", predictResult.Address.Hex())
	
	// Check if already deployed
	existing := registryManager.GetDeploymentWithLabel(contract, env, label, networkInfo.ChainID)
	if existing != nil && existing.Address == predictResult.Address {
		fmt.Println()
		color.New(color.FgYellow).Printf("⚠️  Contract already deployed at predicted address\n")
		
		// Convert hex strings back to byte arrays
		var salt [32]byte
		var initCodeHash [32]byte
		
		if saltBytes, err := hex.DecodeString(existing.Salt); err == nil && len(saltBytes) == 32 {
			copy(salt[:], saltBytes)
		}
		
		if hashBytes, err := hex.DecodeString(existing.InitCodeHash); err == nil && len(hashBytes) == 32 {
			copy(initCodeHash[:], hashBytes)
		}
		
		return &types.DeploymentResult{
			Address:         existing.Address,
			Salt:            salt,
			InitCodeHash:    initCodeHash,
			AlreadyDeployed: true,
		}, nil
	}
	
	s = createSpinner(fmt.Sprintf("Deploying to %s...", predictResult.Address.Hex()))
	result, err := executor.Deploy(contract, env, forge.DeployArgs{
		RpcUrl:  networkInfo.RpcUrl,
		Verify:  verify,
		ChainID: networkInfo.ChainID,
		Label:   label,
		EnvVars: envVars,
		Debug:   debug,
	})
	if err != nil {
		s.Stop()
		return nil, fmt.Errorf("deployment failed: %w", err)
	}
	s.Stop()
	color.New(color.FgGreen).Printf("✓ ")
	fmt.Printf("Deployed successfully\n")

	return result, nil
}

func deployProxyContract(implementationContract, proxyContract string) (*types.DeploymentResult, error) {
	// Print initial deployment summary
	identifier := fmt.Sprintf("%s/%s", env, proxyContract)
	if label != "" {
		identifier += fmt.Sprintf(":%s", label)
	}
	
	// Resolve network first for the summary
	resolver := network.NewResolver(".")
	networkInfo, err := resolver.ResolveNetwork(networkName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve network: %w", err)
	}
	
	fmt.Println()
	color.New(color.FgWhite).Printf("Deploying proxy ")
	color.New(color.FgCyan, color.Bold).Printf("%s", identifier)
	color.New(color.FgWhite).Printf(" to ")
	color.New(color.FgMagenta, color.Bold).Printf("%s\n", networkInfo.Name)
	color.New(color.FgWhite).Printf("Implementation: ")
	color.New(color.FgCyan).Printf("%s/%s", env, implementationContract)
	if implementationLabel != "" {
		color.New(color.FgCyan).Printf(":%s", implementationLabel)
	}
	fmt.Println("\n")
	
	// Step 1: Load deploy configuration
	s := createSpinner("Validating deployment configuration...")
	deployConfig, err := config.LoadDeployConfig(".")
	if err != nil {
		s.Stop()
		return nil, fmt.Errorf("failed to load deploy config: %w", err)
	}
	
	// Step 2: Validate environment configuration
	if err := deployConfig.Validate(env); err != nil {
		s.Stop()
		return nil, fmt.Errorf("invalid deploy configuration for environment '%s': %w", env, err)
	}
	s.Stop()
	color.New(color.FgGreen).Printf("✓ ")
	fmt.Printf("Validated deployment configuration\n")
	
	// Step 3: Check if proxy deploy script exists
	validator := contracts.NewValidator(".")
	if !validator.DeployScriptExists(proxyContract) {
		return nil, fmt.Errorf("proxy deploy script not found: script/deploy/Deploy%s.s.sol\nPlease run 'treb gen proxy %s' first", proxyContract, implementationContract)
	}
	
	// Step 4: Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize registry: %w", err)
	}
	
	// Step 5: Check if implementation is deployed
	s = createSpinner("Checking implementation deployment...")
	var implDeployment *types.DeploymentEntry
	if implementationLabel != "" {
		implDeployment = registryManager.GetDeploymentWithLabel(implementationContract, env, implementationLabel, networkInfo.ChainID)
		if implDeployment == nil {
			s.Stop()
			return nil, fmt.Errorf("implementation %s with label '%s' not found in %s environment on %s", 
				implementationContract, implementationLabel, env, networkInfo.Name)
		}
	} else {
		// Check for default implementation
		implDeployment = registryManager.GetDeployment(implementationContract, env, networkInfo.ChainID)
		if implDeployment == nil {
			s.Stop()
			return nil, fmt.Errorf("implementation %s not found in %s environment on %s\nPlease deploy the implementation first with 'treb deploy %s'", 
				implementationContract, env, networkInfo.Name, implementationContract)
		}
	}
	s.Stop()
	color.New(color.FgGreen).Printf("✓ ")
	fmt.Printf("Found implementation at ")
	color.New(color.FgCyan).Printf("%s\n", implDeployment.Address.Hex())
	
	// Step 6: Check for existing proxy deployment
	existingProxy := registryManager.GetDeploymentWithLabel(proxyContract, env, label, networkInfo.ChainID)
	if existingProxy != nil {
		fmt.Println()
		color.New(color.FgYellow).Printf("⚠️  Proxy already deployed at ")
		color.New(color.FgCyan).Printf("%s\n", existingProxy.Address.Hex())
		
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
	
	// Step 7: Generate environment variables for deployment
	envVars, err := deployConfig.GenerateEnvVars(env)
	if err != nil {
		return nil, fmt.Errorf("failed to generate environment variables: %w", err)
	}
	
	// Add implementation label if specified
	if implementationLabel != "" {
		envVars["IMPLEMENTATION_LABEL"] = implementationLabel
	}
	
	// Step 8: Initialize forge script executor with proxy support
	executor := forge.NewScriptExecutor("", ".", registryManager)
	
	// Step 9: Deploy proxy with spinner
	scriptPath := fmt.Sprintf("script/deploy/Deploy%s.s.sol", proxyContract)
	color.New(color.FgCyan).Printf("\n[%s]\n\n", scriptPath)
	
	s = createSpinner("Deploying proxy...")
	result, err := executor.DeployProxy(proxyContract, implementationContract, env, forge.DeployArgs{
		RpcUrl:  networkInfo.RpcUrl,
		Verify:  verify,
		ChainID: networkInfo.ChainID,
		Label:   label,
		EnvVars: envVars,
		Debug:   debug,
	})
	if err != nil {
		s.Stop()
		return nil, fmt.Errorf("proxy deployment failed: %w", err)
	}
	s.Stop()
	color.New(color.FgGreen).Printf("✓ ")
	fmt.Printf("Proxy deployed successfully\n")
	
	// Store implementation address for summary
	if result.Metadata == nil {
		result.Metadata = &types.ContractMetadata{}
	}
	if result.Metadata.Extra == nil {
		result.Metadata.Extra = make(map[string]interface{})
	}
	result.Metadata.Extra["implementationAddress"] = implDeployment.Address.Hex()
	
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

func predictAddress(contract string) error {
	// Print initial summary
	identifier := fmt.Sprintf("%s/%s", env, contract)
	if label != "" {
		identifier += fmt.Sprintf(":%s", label)
	}
	
	// Resolve network first for the summary
	resolver := network.NewResolver(".")
	networkInfo, err := resolver.ResolveNetwork(networkName)
	if err != nil {
		return fmt.Errorf("failed to resolve network: %w", err)
	}
	
	fmt.Println()
	color.New(color.FgWhite).Printf("Predicting address for ")
	color.New(color.FgWhite, color.Bold).Printf("%s", identifier)
	color.New(color.FgWhite).Printf(" on ")
	color.New(color.FgMagenta, color.Bold).Printf("%s\n", networkInfo.Name)
	fmt.Println()
	
	// Initialize forge executor
	forgeExecutor := forgeExec.NewExecutor(".")
	if err := forgeExecutor.CheckForgeInstallation(); err != nil {
		return fmt.Errorf("forge check failed: %w", err)
	}

	// Build contracts to ensure artifacts are up to date
	s := createSpinner("Building contracts...")
	if err := forgeExecutor.Build(); err != nil {
		s.Stop()
		return fmt.Errorf("build failed: %w", err)
	}
	s.Stop()
	color.New(color.FgGreen).Printf("✓ ")
	fmt.Printf("Built contracts\n")

	// Validate contract exists in src/
	validator := contracts.NewValidator(".")
	contractInfo, err := validator.ValidateContract(contract)
	if err != nil {
		return fmt.Errorf("contract validation failed: %w", err)
	}

	if !contractInfo.Exists {
		return fmt.Errorf("contract %s not found in src/ directory", contract)
	}

	// Load deploy config for environment variables
	deployConfig, err := config.LoadDeployConfig(".")
	if err != nil {
		return fmt.Errorf("failed to load deploy config: %w", err)
	}

	envVars, err := deployConfig.GenerateEnvVars(env)
	if err != nil {
		return fmt.Errorf("failed to generate environment variables: %w", err)
	}

	// Initialize script executor
	executor := forge.NewScriptExecutor("", ".", nil)

	// Predict address with spinner
	s = createSpinner("Calculating deployment address...")
	result, err := executor.PredictAddress(contract, env, forge.DeployArgs{
		RpcUrl:  networkInfo.RpcUrl,
		Label:   label,
		EnvVars: envVars,
	})
	if err != nil {
		s.Stop()
		return fmt.Errorf("address prediction failed: %w", err)
	}
	s.Stop()
	
	// Display results
	fmt.Println()
	color.New(color.FgGreen, color.Bold).Printf("🎯 Predicted Deployment Address\n")
	fmt.Println()
	
	color.New(color.FgWhite, color.Bold).Printf("Contract:     ")
	fmt.Printf("%s/%s", env, contract)
	if label != "" {
		color.New(color.FgCyan).Printf(":%s", label)
	}
	fmt.Println()
	
	color.New(color.FgWhite, color.Bold).Printf("Network:      ")
	color.New(color.FgMagenta).Printf("%s\n", networkInfo.Name)
	
	color.New(color.FgWhite, color.Bold).Printf("Address:      ")
	color.New(color.FgGreen, color.Bold).Printf("%s\n", result.Address.Hex())
	
	color.New(color.FgWhite, color.Bold).Printf("Salt:         ")
	fmt.Printf("%x\n", result.Salt)
	
	fmt.Println()

	return nil
}

// showDeploymentSummary displays a beautiful deployment summary
func showDeploymentSummary(contract, env, network string, result *types.DeploymentResult, isProxy, isSafe bool) {
	fmt.Println()
	
	// Title
	if isProxy {
		color.New(color.FgCyan, color.Bold).Printf("🚀 Proxy Deployment Successful!\n")
	} else if isSafe {
		color.New(color.FgYellow, color.Bold).Printf("🔐 Safe Deployment Initiated!\n")
	} else {
		color.New(color.FgGreen, color.Bold).Printf("✅ Deployment Successful!\n")
	}
	
	fmt.Println()
	
	// Contract info
	color.New(color.FgWhite, color.Bold).Printf("Contract:     ")
	fmt.Printf("%s/%s", env, contract)
	if label != "" {
		color.New(color.FgCyan).Printf(":%s", label)
	}
	fmt.Println()
	
	// Network
	color.New(color.FgWhite, color.Bold).Printf("Network:      ")
	color.New(color.FgMagenta).Printf("%s\n", network)
	
	// Address
	color.New(color.FgWhite, color.Bold).Printf("Address:      ")
	color.New(color.FgGreen).Printf("%s\n", result.Address.Hex())
	
	// Proxy-specific info
	if isProxy && result.Metadata != nil && result.Metadata.Extra != nil {
		if implAddr, ok := result.Metadata.Extra["implementationAddress"].(string); ok {
			color.New(color.FgWhite, color.Bold).Printf("Implementation: ")
			color.New(color.FgCyan).Printf("%s\n", implAddr)
		}
	}
	
	// Transaction details
	if result.TxHash.Hex() != "0x0000000000000000000000000000000000000000000000000000000000000000" {
		color.New(color.FgWhite, color.Bold).Printf("Tx Hash:      ")
		fmt.Printf("%s\n", result.TxHash.Hex())
		
		color.New(color.FgWhite, color.Bold).Printf("Block:        ")
		fmt.Printf("%d\n", result.BlockNumber)
	}
	
	// Safe-specific info
	if isSafe && result.SafeTxHash != (common.Hash{}) {
		fmt.Println()
		color.New(color.FgYellow).Printf("⚠️  Safe Transaction Details:\n")
		color.New(color.FgWhite, color.Bold).Printf("Safe Address: ")
		fmt.Printf("%s\n", result.SafeAddress.Hex())
		color.New(color.FgWhite, color.Bold).Printf("Safe Tx Hash: ")
		fmt.Printf("%s\n", result.SafeTxHash.Hex())
		fmt.Println()
		color.New(color.FgYellow).Printf("Please execute the Safe transaction to complete deployment.\n")
	}
	
	fmt.Println()
}

// createSpinner creates and starts a spinner with the given message
func createSpinner(message string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Suffix = " " + message
	s.Color("cyan")
	s.Start()
	return s
}