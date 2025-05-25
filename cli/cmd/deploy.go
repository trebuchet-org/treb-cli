package cmd

import (
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
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
	predict             bool
)

// DeploymentType represents the type of deployment
type DeploymentType string

const (
	DeploymentTypeSingleton DeploymentType = "singleton"
	DeploymentTypeProxy     DeploymentType = "proxy"
	DeploymentTypeLibrary   DeploymentType = "library"
)

// DeploymentContext holds all deployment configuration
type DeploymentContext struct {
	Type                DeploymentType
	ContractName        string
	ProxyName           string
	ImplementationName  string
	ImplementationLabel string
	Env                 string
	Label               string
	NetworkInfo         *network.NetworkInfo
	ScriptPath          string
	EnvVars             map[string]string
	Predict             bool
	Debug               bool
	Verify              bool
}

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy contracts and libraries",
	Long: `Deploy contracts and libraries using Foundry scripts with automatic registry tracking.
Supports EOA and Safe multisig deployments.

Available subcommands:
  contract - Deploy a regular contract (default when no subcommand specified)
  library  - Deploy a library globally
  proxy    - Deploy a proxy for an implementation

Examples:
  treb deploy Counter --network sepolia
  treb deploy library MathLib --network sepolia
  treb deploy contract Token --env production --verify
  treb deploy proxy Counter --impl-label v1`,
}

var deployContractCmd = &cobra.Command{
	Use:   "contract <name>",
	Short: "Deploy a contract",
	Long: `Deploy a contract using Foundry scripts with automatic registry tracking.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := &DeploymentContext{
			Type:         DeploymentTypeSingleton,
			ContractName: args[0],
			Env:          env,
			Label:        label,
			Predict:      predict,
			Debug:        debug,
			Verify:       verify,
		}
		
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
		ctx := &DeploymentContext{
			Type:                DeploymentTypeProxy,
			ImplementationName:  args[0],
			ProxyName:           args[0] + "Proxy",
			ImplementationLabel: implementationLabel,
			Env:                 env,
			Label:               label,
			Predict:             predict,
			Debug:               debug,
			Verify:              verify,
		}
		
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
		ctx := &DeploymentContext{
			Type:         DeploymentTypeLibrary,
			ContractName: args[0],
			Predict:      predict,
			Debug:        debug,
			Verify:       verify,
		}
		
		if err := runDeployment(ctx); err != nil {
			checkError(err)
		}
	},
}

func init() {
	// Get configured defaults (empty if no config file)
	_, defaultNetwork, defaultVerify, _ := GetConfiguredDefaults()
	
	// Add subcommands
	deployCmd.AddCommand(deployContractCmd)
	deployCmd.AddCommand(deployProxyCmd)
	deployCmd.AddCommand(deployLibraryCmd)
	
	// Make deploy command work without subcommand (defaults to contract deployment)
	deployCmd.Args = func(cmd *cobra.Command, args []string) error {
		// If no subcommand is provided and we have exactly 1 arg, treat it as contract deployment
		if cmd.Flags().NArg() == 1 && !cmd.Flags().Changed("help") {
			return nil
		}
		// Otherwise require a subcommand
		return cobra.NoArgs(cmd, args)
	}
	deployCmd.Run = func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			// Default to contract deployment
			ctx := &DeploymentContext{
				Type:         DeploymentTypeSingleton,
				ContractName: args[0],
				Env:          env,
				Label:        label,
				Predict:      predict,
				Debug:        debug,
				Verify:       verify,
			}
			
			if err := runDeployment(ctx); err != nil {
				checkError(err)
			}
		} else {
			// Show help if no args provided
			cmd.Help()
		}
	}
	
	// Create flags for main deploy command with defaults - env always defaults to "default"
	deployCmd.PersistentFlags().StringVar(&env, "env", "default", "Deployment environment")
	deployCmd.PersistentFlags().StringVar(&networkName, "network", defaultNetwork, "Network to deploy to (defined in foundry.toml)")
	deployCmd.PersistentFlags().BoolVar(&verify, "verify", defaultVerify, "Verify contract after deployment")
	deployCmd.PersistentFlags().StringVar(&label, "label", "", "Optional label for the deployment (included in salt)")
	deployCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Show full Foundry script output")
	deployCmd.PersistentFlags().BoolVar(&predict, "predict", false, "Only predict deployment address without deploying")
	
	// Proxy-specific flags
	deployProxyCmd.Flags().StringVar(&implementationLabel, "impl-label", "", "Implementation label to point to (optional)")
	
	// Mark network flag as required if it doesn't have a default
	if defaultNetwork == "" {
		deployCmd.MarkPersistentFlagRequired("network")
	}
}

// runDeployment executes the deployment based on the context
func runDeployment(ctx *DeploymentContext) error {
	// Resolve network information
	resolver := network.NewResolver(".")
	networkInfo, err := resolver.ResolveNetwork(networkName)
	if err != nil {
		return fmt.Errorf("failed to resolve network: %w", err)
	}
	ctx.NetworkInfo = networkInfo
	
	// Handle library environment setup
	if ctx.Type == DeploymentTypeLibrary {
		deployConfig, err := config.LoadDeployConfig(".")
		if err != nil {
			return fmt.Errorf("failed to load deploy configuration: %w", err)
		}
		
		// Try "libraries" env first, then "default"
		librariesDeployer := deployConfig.GetDeployer("libraries")
		if librariesDeployer != nil {
			ctx.Env = "libraries"
			if librariesDeployer.Type != "private_key" && librariesDeployer.Type != "ledger" {
				return fmt.Errorf("library deployment requires a private key or ledger deployer in 'libraries' environment")
			}
		} else {
			ctx.Env = "default"
			defaultDeployer := deployConfig.GetDeployer("default")
			if defaultDeployer == nil || (defaultDeployer.Type != "private_key" && defaultDeployer.Type != "ledger") {
				return fmt.Errorf("library deployment requires a private key or ledger deployer.\nPlease configure a 'libraries' environment in foundry.toml:\n\n[profile.libraries.deployer]\ntype = \"private_key\"\nprivate_key = \"${DEPLOYER_PRIVATE_KEY}\"")
			}
		}
	}
	
	// Print deployment summary
	printDeploymentSummary(ctx)
	
	// Common pre-deployment steps
	if err := validateDeploymentConfig(ctx); err != nil {
		return err
	}
	
	if err := buildContracts(); err != nil {
		return err
	}
	
	// Type-specific validation and setup
	switch ctx.Type {
	case DeploymentTypeSingleton:
		if err := validateContract(ctx); err != nil {
			return err
		}
	case DeploymentTypeProxy:
		if err := validateProxyDeployment(ctx); err != nil {
			return err
		}
	case DeploymentTypeLibrary:
		if err := validateLibrary(ctx); err != nil {
			return err
		}
	}
	
	// If predict-only mode, run prediction and exit
	if ctx.Predict {
		return runPrediction(ctx)
	}
	
	// Execute deployment
	result, err := executeDeployment(ctx)
	if err != nil {
		return err
	}
	
	// Show deployment summary if not already deployed
	if result != nil && !result.AlreadyDeployed {
		showDeploymentSuccess(ctx, result)
	}
	
	return nil
}

// printDeploymentSummary prints the initial deployment summary
func printDeploymentSummary(ctx *DeploymentContext) {
	fmt.Println()
	
	switch ctx.Type {
	case DeploymentTypeSingleton:
		identifier := fmt.Sprintf("%s/%s", ctx.Env, ctx.ContractName)
		if ctx.Label != "" {
			identifier += fmt.Sprintf(":%s", ctx.Label)
		}
		
		if ctx.Predict {
			color.New(color.FgWhite).Printf("Predicting address for ")
			color.New(color.FgWhite, color.Bold).Printf("%s", identifier)
		} else {
			color.New(color.FgWhite).Printf("Deploying ")
			color.New(color.FgWhite, color.Bold).Printf("%s", identifier)
		}
		color.New(color.FgWhite).Printf(" to ")
		color.New(color.FgMagenta, color.Bold).Printf("%s\n", ctx.NetworkInfo.Name)
		
	case DeploymentTypeProxy:
		identifier := fmt.Sprintf("%s/%s", ctx.Env, ctx.ProxyName)
		if ctx.Label != "" {
			identifier += fmt.Sprintf(":%s", ctx.Label)
		}
		
		if ctx.Predict {
			color.New(color.FgWhite).Printf("Predicting proxy address ")
			color.New(color.FgCyan, color.Bold).Printf("%s", identifier)
		} else {
			color.New(color.FgWhite).Printf("Deploying proxy ")
			color.New(color.FgCyan, color.Bold).Printf("%s", identifier)
		}
		color.New(color.FgWhite).Printf(" to ")
		color.New(color.FgMagenta, color.Bold).Printf("%s\n", ctx.NetworkInfo.Name)
		
		color.New(color.FgWhite).Printf("Implementation: ")
		color.New(color.FgCyan).Printf("%s/%s", ctx.Env, ctx.ImplementationName)
		if ctx.ImplementationLabel != "" {
			color.New(color.FgCyan).Printf(":%s", ctx.ImplementationLabel)
		}
		fmt.Println()
		
	case DeploymentTypeLibrary:
		if ctx.Predict {
			color.New(color.FgWhite).Printf("Predicting library address ")
			color.New(color.FgWhite, color.Bold).Printf("%s", ctx.ContractName)
		} else {
			color.New(color.FgWhite).Printf("Deploying library ")
			color.New(color.FgWhite, color.Bold).Printf("%s", ctx.ContractName)
		}
		color.New(color.FgWhite).Printf(" to ")
		color.New(color.FgMagenta, color.Bold).Printf("%s\n", ctx.NetworkInfo.Name)
	}
	
	fmt.Println()
}

// validateDeploymentConfig validates the deployment configuration
func validateDeploymentConfig(ctx *DeploymentContext) error {
	s := createSpinner("Validating deployment configuration...")
	
	deployConfig, err := config.LoadDeployConfig(".")
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to load deploy configuration: %w", err)
	}
	
	if err := deployConfig.Validate(ctx.Env); err != nil {
		s.Stop()
		return fmt.Errorf("invalid deploy configuration for environment '%s': %w", ctx.Env, err)
	}
	
	// Generate environment variables
	envVars, err := deployConfig.GenerateEnvVars(ctx.Env)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to generate environment variables: %w", err)
	}
	ctx.EnvVars = envVars
	
	s.Stop()
	color.New(color.FgGreen).Printf("✓ ")
	fmt.Printf("Validated deployment configuration\n")
	
	return nil
}

// buildContracts runs forge build
func buildContracts() error {
	forgeExecutor := forgeExec.NewExecutor(".")
	if err := forgeExecutor.CheckForgeInstallation(); err != nil {
		return fmt.Errorf("forge check failed: %w", err)
	}
	
	s := createSpinner("Building contracts...")
	if err := forgeExecutor.Build(); err != nil {
		s.Stop()
		return fmt.Errorf("build failed: %w", err)
	}
	s.Stop()
	color.New(color.FgGreen).Printf("✓ ")
	fmt.Printf("Built contracts\n")
	
	return nil
}

// validateContract validates a singleton contract deployment
func validateContract(ctx *DeploymentContext) error {
	// Resolve the contract
	contractDiscovery, err := interactive.ResolveContract(ctx.ContractName)
	if err != nil {
		return fmt.Errorf("failed to resolve contract: %w", err)
	}
	
	// Update context with resolved contract name
	ctx.ContractName = contractDiscovery.Name
	
	// Check if deploy script exists
	validator := contracts.NewValidator(".")
	if !validator.DeployScriptExists(ctx.ContractName) {
		if ctx.Predict {
			return fmt.Errorf("deploy script required but not found: script/deploy/Deploy%s.s.sol", ctx.ContractName)
		}
		
		fmt.Printf("\nDeploy script not found for %s\n", ctx.ContractName)
		
		// Ask if user wants to generate the script
		selector := interactive.NewSelector()
		shouldGenerate, err := selector.PromptConfirm("Would you like to generate a deploy script?", true)
		if err != nil || !shouldGenerate {
			return fmt.Errorf("deploy script required but not found: script/deploy/Deploy%s.s.sol", ctx.ContractName)
		}
		
		// Generate the script interactively
		fmt.Printf("\nStarting interactive script generation...\n\n")
		generator := interactive.NewGenerator(".")
		if err := generator.GenerateDeployScript(ctx.ContractName); err != nil {
			return fmt.Errorf("script generation failed: %w", err)
		}
		return fmt.Errorf("script generated, please run the deploy command again")
	}
	
	// Set script path
	discovery := contracts.NewDiscovery(".")
	ctx.ScriptPath = discovery.GetDeployScriptPath(*contractDiscovery)
	
	return nil
}

// validateProxyDeployment validates a proxy deployment
func validateProxyDeployment(ctx *DeploymentContext) error {
	// Check if proxy deploy script exists
	validator := contracts.NewValidator(".")
	if !validator.DeployScriptExists(ctx.ProxyName) {
		return fmt.Errorf("proxy deploy script not found: script/deploy/Deploy%s.s.sol\nPlease run 'treb gen proxy %s' first", ctx.ProxyName, ctx.ImplementationName)
	}
	
	// Set script path
	ctx.ScriptPath = fmt.Sprintf("script/deploy/Deploy%s.s.sol", ctx.ProxyName)
	
	// Check if implementation is deployed (unless we're just predicting)
	if !ctx.Predict {
		registryManager, err := registry.NewManager("deployments.json")
		if err != nil {
			return fmt.Errorf("failed to initialize registry: %w", err)
		}
		
		s := createSpinner("Checking implementation deployment...")
		var implDeployment *types.DeploymentEntry
		if ctx.ImplementationLabel != "" {
			implDeployment = registryManager.GetDeploymentWithLabel(ctx.ImplementationName, ctx.Env, ctx.ImplementationLabel, ctx.NetworkInfo.ChainID)
			if implDeployment == nil {
				s.Stop()
				return fmt.Errorf("implementation %s with label '%s' not found in %s environment on %s", 
					ctx.ImplementationName, ctx.ImplementationLabel, ctx.Env, ctx.NetworkInfo.Name)
			}
		} else {
			implDeployment = registryManager.GetDeployment(ctx.ImplementationName, ctx.Env, ctx.NetworkInfo.ChainID)
			if implDeployment == nil {
				s.Stop()
				return fmt.Errorf("implementation %s not found in %s environment on %s\nPlease deploy the implementation first with 'treb deploy %s'", 
					ctx.ImplementationName, ctx.Env, ctx.NetworkInfo.Name, ctx.ImplementationName)
			}
		}
		s.Stop()
		color.New(color.FgGreen).Printf("✓ ")
		fmt.Printf("Found implementation at ")
		color.New(color.FgCyan).Printf("%s\n", implDeployment.Address.Hex())
		
		// Add implementation info to env vars
		if ctx.ImplementationLabel != "" {
			ctx.EnvVars["IMPLEMENTATION_LABEL"] = ctx.ImplementationLabel
		}
		ctx.EnvVars["IMPLEMENTATION_IDENTIFIER"] = fmt.Sprintf("%s/%s", ctx.Env, ctx.ImplementationName)
		if ctx.ImplementationLabel != "" {
			ctx.EnvVars["IMPLEMENTATION_IDENTIFIER"] = fmt.Sprintf("%s:%s", ctx.EnvVars["IMPLEMENTATION_IDENTIFIER"], ctx.ImplementationLabel)
		}
	}
	
	return nil
}

// validateLibrary validates a library deployment
func validateLibrary(ctx *DeploymentContext) error {
	// Validate it's actually a library
	validator := contracts.NewValidator(".")
	if !validator.IsLibrary(ctx.ContractName) {
		return fmt.Errorf("%s is not a library", ctx.ContractName)
	}
	
	// Check if deploy script exists
	scriptPath := fmt.Sprintf("script/deploy/Deploy%s.s.sol", ctx.ContractName)
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("library deploy script required but not found: %s\n\nGenerate it with: treb gen library %s", scriptPath, ctx.ContractName)
	}
	
	// Set script path
	ctx.ScriptPath = scriptPath
	
	// Add library-specific env vars
	ctx.EnvVars["LIBRARY_NAME"] = ctx.ContractName
	ctx.EnvVars["LIBRARY_ARTIFACT_PATH"] = fmt.Sprintf("src/%s.sol:%s", ctx.ContractName, ctx.ContractName)
	
	return nil
}

// runPrediction runs address prediction without deployment
func runPrediction(ctx *DeploymentContext) error {
	switch ctx.Type {
	case DeploymentTypeLibrary:
		// For libraries, run the script without broadcast and parse the address
		return runLibraryPrediction(ctx)
	default:
		// For contracts/proxies, use the predictAddress() function
		return runScriptPrediction(ctx)
	}
}

// runScriptPrediction runs predictAddress() on deployment scripts
func runScriptPrediction(ctx *DeploymentContext) error {
	color.New(color.FgCyan).Printf("\n[%s]\n\n", ctx.ScriptPath)
	
	s := createSpinner("Calculating deployment address...")
	
	// Set up environment variables
	envVars := os.Environ()
	for key, value := range ctx.EnvVars {
		envVars = append(envVars, fmt.Sprintf("%s=%s", key, value))
	}
	
	// Add deployment-specific env vars
	envVars = append(envVars, fmt.Sprintf("DEPLOYMENT_ENV=%s", ctx.Env))
	if ctx.Label != "" {
		envVars = append(envVars, fmt.Sprintf("DEPLOYMENT_LABEL=%s", ctx.Label))
	}
	
	// Execute predictAddress function
	cmdArgs := []string{"script", ctx.ScriptPath, "--sig", "predictAddress()", "-vvvv"}
	if ctx.NetworkInfo.RpcUrl != "" {
		cmdArgs = append(cmdArgs, "--rpc-url", ctx.NetworkInfo.RpcUrl)
	}
	
	cmd := exec.Command("forge", cmdArgs...)
	cmd.Dir = "."
	cmd.Env = envVars
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.Stop()
		if ctx.Debug {
			fmt.Printf("Full output:\n%s\n", string(output))
		}
		return handleForgeError(err, output)
	}
	s.Stop()
	
	if ctx.Debug {
		fmt.Printf("\n=== Full Foundry Script Output (Prediction) ===\n")
		fmt.Printf("%s\n", string(output))
		fmt.Printf("=== End of Output ===\n\n")
	}
	
	// Parse prediction output
	predictResult, err := parsePredictionOutput(string(output))
	if err != nil {
		return fmt.Errorf("failed to parse prediction output: %w", err)
	}
	
	// Display results
	fmt.Println()
	color.New(color.FgGreen, color.Bold).Printf("🎯 Predicted Deployment Address\n")
	fmt.Println()
	
	switch ctx.Type {
	case DeploymentTypeSingleton:
		color.New(color.FgWhite, color.Bold).Printf("Contract:     ")
		fmt.Printf("%s/%s", ctx.Env, ctx.ContractName)
		if ctx.Label != "" {
			color.New(color.FgCyan).Printf(":%s", ctx.Label)
		}
	case DeploymentTypeProxy:
		color.New(color.FgWhite, color.Bold).Printf("Proxy:        ")
		fmt.Printf("%s/%s", ctx.Env, ctx.ProxyName)
		if ctx.Label != "" {
			color.New(color.FgCyan).Printf(":%s", ctx.Label)
		}
	}
	fmt.Println()
	
	color.New(color.FgWhite, color.Bold).Printf("Network:      ")
	color.New(color.FgMagenta).Printf("%s\n", ctx.NetworkInfo.Name)
	
	color.New(color.FgWhite, color.Bold).Printf("Address:      ")
	color.New(color.FgGreen, color.Bold).Printf("%s\n", predictResult.Address.Hex())
	
	if predictResult.Salt != [32]byte{} {
		color.New(color.FgWhite, color.Bold).Printf("Salt:         ")
		fmt.Printf("%x\n", predictResult.Salt)
	}
	
	fmt.Println()
	
	return nil
}

// runLibraryPrediction runs library deployment script without broadcast to get predicted address
func runLibraryPrediction(ctx *DeploymentContext) error {
	color.New(color.FgCyan).Printf("\n[%s]\n\n", ctx.ScriptPath)
	
	s := createSpinner("Calculating library deployment address...")
	
	// Set up environment variables
	envVars := os.Environ()
	for key, value := range ctx.EnvVars {
		envVars = append(envVars, fmt.Sprintf("%s=%s", key, value))
	}
	
	// Execute script without broadcast
	cmdArgs := []string{"script", ctx.ScriptPath, "-vvvv"}
	if ctx.NetworkInfo.RpcUrl != "" {
		cmdArgs = append(cmdArgs, "--rpc-url", ctx.NetworkInfo.RpcUrl)
	}
	
	cmd := exec.Command("forge", cmdArgs...)
	cmd.Dir = "."
	cmd.Env = envVars
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.Stop()
		if ctx.Debug {
			fmt.Printf("Full output:\n%s\n", string(output))
		}
		return handleForgeError(err, output)
	}
	s.Stop()
	
	if ctx.Debug {
		fmt.Printf("\n=== Full Foundry Script Output (Prediction) ===\n")
		fmt.Printf("%s\n", string(output))
		fmt.Printf("=== End of Output ===\n\n")
	}
	
	// Parse library address from output
	address, err := parseLibraryAddress(string(output))
	if err != nil {
		return fmt.Errorf("failed to parse library address: %w", err)
	}
	
	// Display results
	fmt.Println()
	color.New(color.FgGreen, color.Bold).Printf("🎯 Predicted Library Address\n")
	fmt.Println()
	
	color.New(color.FgWhite, color.Bold).Printf("Library:      ")
	fmt.Printf("%s\n", ctx.ContractName)
	
	color.New(color.FgWhite, color.Bold).Printf("Network:      ")
	color.New(color.FgMagenta).Printf("%s\n", ctx.NetworkInfo.Name)
	
	color.New(color.FgWhite, color.Bold).Printf("Address:      ")
	color.New(color.FgGreen, color.Bold).Printf("%s\n", address.Hex())
	
	fmt.Println()
	
	return nil
}

// executeDeployment executes the actual deployment
func executeDeployment(ctx *DeploymentContext) (*types.DeploymentResult, error) {
	// Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize registry: %w", err)
	}
	
	// Check for existing deployment
	if result, exists := checkExistingDeployment(ctx, registryManager); exists {
		return result, nil
	}
	
	// Execute deployment script
	color.New(color.FgCyan).Printf("\n[%s]\n\n", ctx.ScriptPath)
	
	s := createSpinner("Deploying...")
	
	// Set up environment variables
	envVars := os.Environ()
	for key, value := range ctx.EnvVars {
		envVars = append(envVars, fmt.Sprintf("%s=%s", key, value))
	}
	
	// Add deployment-specific env vars
	envVars = append(envVars, fmt.Sprintf("DEPLOYMENT_ENV=%s", ctx.Env))
	if ctx.Label != "" {
		envVars = append(envVars, fmt.Sprintf("DEPLOYMENT_LABEL=%s", ctx.Label))
	}
	
	// Execute deployment
	cmdArgs := []string{"script", ctx.ScriptPath, "-vvvv", "--broadcast"}
	if ctx.NetworkInfo.RpcUrl != "" {
		cmdArgs = append(cmdArgs, "--rpc-url", ctx.NetworkInfo.RpcUrl)
	}
	if ctx.Verify {
		cmdArgs = append(cmdArgs, "--verify")
	}
	
	cmd := exec.Command("forge", cmdArgs...)
	cmd.Dir = "."
	cmd.Env = envVars
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.Stop()
		if ctx.Debug {
			fmt.Printf("Full output:\n%s\n", string(output))
		}
		return nil, handleForgeError(err, output)
	}
	s.Stop()
	
	if ctx.Debug {
		fmt.Printf("\n=== Full Foundry Script Output ===\n")
		fmt.Printf("%s\n", string(output))
		fmt.Printf("=== End of Output ===\n\n")
	}
	
	color.New(color.FgGreen).Printf("✓ ")
	fmt.Printf("Deployed successfully\n")
	
	// Parse deployment result
	deploymentData, err := parseDeploymentResult(string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse deployment result: %w", err)
	}
	
	// Create DeploymentResult from parsed data
	result := &types.DeploymentResult{
		Address:      common.HexToAddress(deploymentData["ADDRESS"]),
		Type:         strings.ToLower(deploymentData["DEPLOYMENT_TYPE"]),
		Metadata:     &types.ContractMetadata{},
	}
	
	// Parse additional fields
	if salt := deploymentData["SALT"]; salt != "" {
		if saltBytes := common.FromHex(salt); len(saltBytes) == 32 {
			copy(result.Salt[:], saltBytes)
		}
	}
	
	if initCodeHash := deploymentData["INIT_CODE_HASH"]; initCodeHash != "" {
		if hashBytes := common.FromHex(initCodeHash); len(hashBytes) == 32 {
			copy(result.InitCodeHash[:], hashBytes)
		}
	}
	
	if blockNum := deploymentData["BLOCK_NUMBER"]; blockNum != "" {
		fmt.Sscanf(blockNum, "%d", &result.BlockNumber)
	}
	
	if txHash := deploymentData["TX_HASH"]; txHash != "" {
		result.TxHash = common.HexToHash(txHash)
	}
	
	if safeTxHash := deploymentData["SAFE_TX_HASH"]; safeTxHash != "" {
		result.SafeTxHash = common.HexToHash(safeTxHash)
	}
	
	// Handle proxy-specific metadata
	if ctx.Type == DeploymentTypeProxy {
		if result.Metadata.Extra == nil {
			result.Metadata.Extra = make(map[string]interface{})
		}
		if implAddr := deploymentData["IMPLEMENTATION_ADDRESS"]; implAddr != "" {
			result.Metadata.Extra["implementationAddress"] = implAddr
		}
		result.TargetContract = ctx.ImplementationName
	}
	
	// Record deployment based on type
	switch ctx.Type {
	case DeploymentTypeSingleton, DeploymentTypeProxy:
		contractName := ctx.ContractName
		if ctx.Type == DeploymentTypeProxy {
			contractName = ctx.ProxyName
		}
		err = registryManager.RecordDeployment(contractName, ctx.Env, result, ctx.NetworkInfo.ChainID)
	case DeploymentTypeLibrary:
		err = registryManager.RecordLibraryDeployment(ctx.ContractName, result, ctx.NetworkInfo.ChainID)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to record deployment: %w", err)
	}
	
	return result, nil
}

// checkExistingDeployment checks if deployment already exists
func checkExistingDeployment(ctx *DeploymentContext, registry forge.RegistryManager) (*types.DeploymentResult, bool) {
	var existing *types.DeploymentEntry
	
	switch ctx.Type {
	case DeploymentTypeSingleton:
		existing = registry.GetDeploymentWithLabel(ctx.ContractName, ctx.Env, ctx.Label, ctx.NetworkInfo.ChainID)
	case DeploymentTypeProxy:
		existing = registry.GetDeploymentWithLabel(ctx.ProxyName, ctx.Env, ctx.Label, ctx.NetworkInfo.ChainID)
	case DeploymentTypeLibrary:
		existing = registry.GetLibrary(ctx.ContractName, ctx.NetworkInfo.ChainID)
	}
	
	if existing != nil {
		// Check deployment status
		if existing.Deployment.Status == "pending_safe" {
			fmt.Println()
			color.New(color.FgYellow).Printf("⚠️  Deployment is pending Safe execution\n")
			fmt.Printf("Address: %s\n", existing.Address.Hex())
			fmt.Printf("Safe: %s\n", existing.Deployment.SafeAddress)
			if existing.Deployment.SafeTxHash != nil {
				fmt.Printf("Safe Tx Hash: %s\n", existing.Deployment.SafeTxHash.Hex())
			}
			fmt.Printf("\nPlease execute the pending Safe transaction before attempting to redeploy\n")
			return nil, true
		}
		
		fmt.Println()
		color.New(color.FgYellow).Printf("⚠️  Already deployed at ")
		color.New(color.FgCyan).Printf("%s\n", existing.Address.Hex())
		
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
		}, true
	}
	
	return nil, false
}

// showDeploymentSuccess shows the deployment success message
func showDeploymentSuccess(ctx *DeploymentContext, result *types.DeploymentResult) {
	fmt.Println()
	
	// Title based on deployment type and status
	isSafe := result.SafeTxHash != (common.Hash{})
	
	switch ctx.Type {
	case DeploymentTypeProxy:
		color.New(color.FgCyan, color.Bold).Printf("🚀 Proxy Deployment Successful!\n")
	case DeploymentTypeLibrary:
		color.New(color.FgGreen, color.Bold).Printf("✅ Library Deployment Successful!\n")
	default:
		if isSafe {
			color.New(color.FgYellow, color.Bold).Printf("🔐 Safe Deployment Initiated!\n")
		} else {
			color.New(color.FgGreen, color.Bold).Printf("✅ Deployment Successful!\n")
		}
	}
	
	fmt.Println()
	
	// Show deployment details
	switch ctx.Type {
	case DeploymentTypeSingleton:
		color.New(color.FgWhite, color.Bold).Printf("Contract:     ")
		fmt.Printf("%s/%s", ctx.Env, ctx.ContractName)
		if ctx.Label != "" {
			color.New(color.FgCyan).Printf(":%s", ctx.Label)
		}
		fmt.Println()
		
	case DeploymentTypeProxy:
		color.New(color.FgWhite, color.Bold).Printf("Proxy:        ")
		fmt.Printf("%s/%s", ctx.Env, ctx.ProxyName)
		if ctx.Label != "" {
			color.New(color.FgCyan).Printf(":%s", ctx.Label)
		}
		fmt.Println()
		
		if result.Metadata != nil && result.Metadata.Extra != nil {
			if implAddr, ok := result.Metadata.Extra["implementationAddress"].(string); ok {
				color.New(color.FgWhite, color.Bold).Printf("Implementation: ")
				color.New(color.FgCyan).Printf("%s\n", implAddr)
			}
		}
		
	case DeploymentTypeLibrary:
		color.New(color.FgWhite, color.Bold).Printf("Library:      ")
		fmt.Printf("%s\n", ctx.ContractName)
		
		// Show library format for foundry.toml
		fmt.Printf("\n")
		color.New(color.FgWhite, color.Bold).Printf("Add to foundry.toml libraries array:\n")
		color.New(color.FgCyan).Printf("  \"src/%s.sol:%s:%s\"\n", 
			ctx.ContractName, ctx.ContractName, result.Address.Hex())
		
		// Try to auto-update foundry.toml
		foundryManager := config.NewFoundryManager(".")
		if err := foundryManager.AddLibraryAuto("default", ctx.ContractName, result.Address.Hex()); err == nil {
			fmt.Printf("\n✅ Updated foundry.toml with library mapping\n")
		}
	}
	
	// Common details
	color.New(color.FgWhite, color.Bold).Printf("Network:      ")
	color.New(color.FgMagenta).Printf("%s\n", ctx.NetworkInfo.Name)
	
	color.New(color.FgWhite, color.Bold).Printf("Address:      ")
	color.New(color.FgGreen).Printf("%s\n", result.Address.Hex())
	
	// Transaction details
	if result.TxHash.Hex() != "0x0000000000000000000000000000000000000000000000000000000000000000" {
		color.New(color.FgWhite, color.Bold).Printf("Tx Hash:      ")
		fmt.Printf("%s\n", result.TxHash.Hex())
		
		if result.BlockNumber > 0 {
			color.New(color.FgWhite, color.Bold).Printf("Block:        ")
			fmt.Printf("%d\n", result.BlockNumber)
		}
	}
	
	// Safe-specific info
	if isSafe && result.SafeTxHash != (common.Hash{}) {
		fmt.Println()
		color.New(color.FgYellow).Printf("⚠️  Safe Transaction Details:\n")
		if result.SafeAddress != (common.Address{}) {
			color.New(color.FgWhite, color.Bold).Printf("Safe Address: ")
			fmt.Printf("%s\n", result.SafeAddress.Hex())
		}
		color.New(color.FgWhite, color.Bold).Printf("Safe Tx Hash: ")
		fmt.Printf("%s\n", result.SafeTxHash.Hex())
		fmt.Println()
		color.New(color.FgYellow).Printf("Please execute the Safe transaction to complete deployment.\n")
	}
	
	fmt.Println()
}

// parseDeploymentResult parses the DEPLOYMENT_RESULT section from script output
func parseDeploymentResult(output string) (map[string]string, error) {
	lines := strings.Split(output, "\n")
	result := make(map[string]string)
	inDeploymentSection := false
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if line == "=== DEPLOYMENT_RESULT ===" {
			inDeploymentSection = true
			continue
		}
		
		if line == "=== END_DEPLOYMENT ===" {
			inDeploymentSection = false
			break
		}
		
		if !inDeploymentSection {
			continue
		}
		
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		result[key] = value
	}
	
	// Check for required fields
	if result["ADDRESS"] == "" {
		return nil, fmt.Errorf("deployment result missing required ADDRESS field")
	}
	
	return result, nil
}

// parsePredictionOutput parses the prediction output
func parsePredictionOutput(output string) (*types.PredictResult, error) {
	// Look for the line containing "Predicted Address:"
	predictedAddressRegex := regexp.MustCompile(`Predicted Address:\s*(0x[a-fA-F0-9]{40})`)
	
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if matches := predictedAddressRegex.FindStringSubmatch(line); len(matches) > 1 {
			address := common.HexToAddress(matches[1])
			return &types.PredictResult{
				Address: address,
			}, nil
		}
	}
	
	return nil, fmt.Errorf("'Predicted Address:' not found in output")
}

// parseLibraryAddress parses library address from deployment output
func parseLibraryAddress(output string) (common.Address, error) {
	// Look for LIBRARY_ADDRESS in the output
	libraryAddressRegex := regexp.MustCompile(`LIBRARY_ADDRESS:\s*(0x[a-fA-F0-9]{40})`)
	
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if matches := libraryAddressRegex.FindStringSubmatch(line); len(matches) > 1 {
			return common.HexToAddress(matches[1]), nil
		}
	}
	
	return common.Address{}, fmt.Errorf("library address not found in output")
}

// handleForgeError handles forge script errors and extracts known revert reasons
func handleForgeError(err error, output []byte) error {
	outputStr := string(output)
	
	// Known error patterns from the Solidity contracts
	knownErrors := map[string]string{
		"DeploymentPendingSafe()": "Deployment is pending Safe execution. Please execute the Safe transaction before redeploying.",
		"DeploymentAlreadyExists()": "Contract already deployed at the predicted address.",
		"DeploymentFailed()": "Deployment transaction failed.",
		"DeploymentAddressMismatch()": "Deployed address does not match predicted address.",
		"UnlinkedLibraries()": "Contract has unlinked libraries. Please link libraries before deployment.",
		"CompilationArtifactsNotFound()": "Contract compilation artifacts not found. Please run 'forge build'.",
		"IMPLEMENTATION_IDENTIFIER is not set": "Implementation identifier not set. Please specify the implementation contract.",
		"LIBRARY_NAME is not set": "Library name not set in environment.",
		"LIBRARY_ARTIFACT_PATH is not set": "Library artifact path not set in environment.",
	}
	
	// Check for known errors in output
	for pattern, message := range knownErrors {
		if strings.Contains(outputStr, pattern) {
			return fmt.Errorf(message)
		}
	}
	
	// Check for generic revert
	if strings.Contains(outputStr, "revert") || strings.Contains(outputStr, "Revert") {
		// Try to extract revert reason
		revertRegex := regexp.MustCompile(`(?:revert|Revert)(?:ed)?\s*(?:with\s*)?(?:reason\s*)?(?:string\s*)?[:\s]*["']?([^"'\n]+)["']?`)
		if matches := revertRegex.FindStringSubmatch(outputStr); len(matches) > 1 {
			return fmt.Errorf("deployment reverted: %s", strings.TrimSpace(matches[1]))
		}
		return fmt.Errorf("deployment reverted without reason")
	}
	
	// Default error
	return fmt.Errorf("forge script failed: %w", err)
}

// createSpinner creates and starts a spinner with the given message
func createSpinner(message string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Suffix = " " + message
	s.Color("cyan")
	s.Start()
	return s
}