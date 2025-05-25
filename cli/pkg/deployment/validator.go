package deployment

import (
	"fmt"
	"os"

	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	forgeExec "github.com/trebuchet-org/treb-cli/cli/pkg/forge"
	"github.com/trebuchet-org/treb-cli/cli/pkg/interactive"
	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
)

// Validator handles deployment validation
type Validator struct {
	projectRoot string
}

// NewValidator creates a new deployment validator
func NewValidator(projectRoot string) *Validator {
	return &Validator{
		projectRoot: projectRoot,
	}
}

// ValidateDeploymentConfig validates the deployment configuration
func (v *Validator) ValidateDeploymentConfig(ctx *Context) error {
	// Load deploy config
	deployConfig, err := config.LoadDeployConfig(v.projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load deploy config: %w", err)
	}

	// Validate deploy config for environment
	if err := deployConfig.Validate(ctx.Env); err != nil {
		return fmt.Errorf("invalid deploy config: %w", err)
	}

	// Resolve network
	networkResolver := network.NewResolver(v.projectRoot)
	networkInfo, err := networkResolver.ResolveNetwork(ctx.NetworkName)
	if err != nil {
		return fmt.Errorf("failed to resolve network: %w", err)
	}
	ctx.NetworkInfo = networkInfo

	// Generate environment variables
	envVars, err := deployConfig.GenerateEnvVars(ctx.Env)
	if err != nil {
		return fmt.Errorf("failed to generate environment variables: %w", err)
	}

	// Add network-specific environment variables
	if networkInfo.RpcUrl != "" {
		envVars["RPC_URL"] = networkInfo.RpcUrl
	}
	envVars["CHAIN_ID"] = fmt.Sprintf("%d", networkInfo.ChainID)

	// Add deployment label if specified
	if ctx.Label != "" {
		envVars["DEPLOYMENT_LABEL"] = ctx.Label
	}

	ctx.EnvVars = envVars

	return nil
}

// BuildContracts runs forge build
func (v *Validator) BuildContracts() error {
	forgeExecutor := forgeExec.NewExecutor(v.projectRoot)
	if err := forgeExecutor.CheckForgeInstallation(); err != nil {
		return fmt.Errorf("forge check failed: %w", err)
	}

	if err := forgeExecutor.Build(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	return nil
}

// ValidateContract validates a singleton contract deployment
func (v *Validator) ValidateContract(ctx *Context) error {
	generated, err := v.ValidateContractWithGeneration(ctx)
	if err != nil {
		return err
	}
	if generated {
		return fmt.Errorf("script generated, please run the deploy command again")
	}
	return nil
}

// ValidateContractWithGeneration validates a singleton contract deployment and returns whether a script was generated
func (v *Validator) ValidateContractWithGeneration(ctx *Context) (bool, error) {
	// Resolve the contract
	contractInfo, err := interactive.ResolveContract(ctx.ContractQuery)
	if err != nil {
		return false, fmt.Errorf("failed to resolve contract: %w", err)
	}

	// Update context with resolved contract name
	ctx.ContractInfo = contractInfo

	// Check if deploy script exists using the generator's path logic
	generator := contracts.NewGenerator(v.projectRoot)
	scriptPath := generator.GetDeployScriptPath(contractInfo)
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		if ctx.Predict {
			return false, fmt.Errorf("deploy script required but not found: %s", scriptPath)
		}

		fmt.Printf("\nDeploy script not found for %s (%s)\n", ctx.ContractInfo.Name, ctx.ContractInfo.Path)

		// Ask if user wants to generate the script
		selector := interactive.NewSelector()
		shouldGenerate, err := selector.PromptConfirm("Would you like to generate a deploy script?", true)
		if err != nil || !shouldGenerate {
			return false, fmt.Errorf("deploy script required but not found: %s", scriptPath)
		}

		// Generate the script interactively
		fmt.Printf("\nStarting interactive script generation...\n\n")
		interactiveGenerator := interactive.NewGenerator(v.projectRoot)
		if err := interactiveGenerator.GenerateDeployScriptForContract(contractInfo); err != nil {
			return false, fmt.Errorf("script generation failed: %w", err)
		}
		return true, nil
	}

	// Set script path
	ctx.ScriptPath = scriptPath

	return false, nil
}

// ValidateProxyDeployment validates a proxy deployment
func (v *Validator) ValidateProxyDeployment(ctx *Context) error {
	// Resolve the contract
	contractInfo, err := interactive.ResolveContract(ctx.ContractQuery)
	if err != nil {
		return fmt.Errorf("failed to resolve contract: %w", err)
	}

	// Update context with resolved contract name
	ctx.ContractInfo = contractInfo

	// Check if proxy deploy script exists
	generator := contracts.NewGenerator(v.projectRoot)
	scriptPath := generator.GetProxyScriptPath(contractInfo)
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("proxy deploy script not found: %s", scriptPath)
	}

	// Set script path
	ctx.ScriptPath = scriptPath

	// Get implementation info from registry
	if ctx.ImplementationLabel == "" {
		ctx.ImplementationLabel = "default"
	}

	// Determine implementation identifier
	implIdentifier := ctx.ImplementationName
	if ctx.Env != "" {
		implIdentifier = fmt.Sprintf("%s/%s", ctx.Env, ctx.ImplementationName)
	}
	if ctx.ImplementationLabel != "default" {
		implIdentifier = fmt.Sprintf("%s:%s", implIdentifier, ctx.ImplementationLabel)
	}

	// TODO: Validate implementation exists in registry
	// This would require access to the registry manager

	return nil
}

// ValidateLibrary validates a library deployment
func (v *Validator) ValidateLibrary(ctx *Context) error {
	// Resolve the library
	contractInfo, err := interactive.ResolveContract(ctx.ContractQuery)
	if err != nil {
		return fmt.Errorf("failed to resolve library: %w", err)
	}

	// Update context with resolved contract name
	ctx.ContractInfo = contractInfo

	return nil
}

// ValidateAll runs all necessary validations based on deployment type
func (v *Validator) ValidateAll(ctx *Context) error {
	// Always validate deployment config
	if err := v.ValidateDeploymentConfig(ctx); err != nil {
		return err
	}

	// Always build contracts
	if err := v.BuildContracts(); err != nil {
		return err
	}

	// Type-specific validation
	switch ctx.Type {
	case TypeSingleton:
		return v.ValidateContract(ctx)
	case TypeProxy:
		return v.ValidateProxyDeployment(ctx)
	case TypeLibrary:
		return v.ValidateLibrary(ctx)
	default:
		return fmt.Errorf("unknown deployment type: %s", ctx.Type)
	}
}
