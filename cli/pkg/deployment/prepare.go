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

// ValidateDeploymentConfig validates the deployment configuration
func (d *DeploymentContext) ValidateDeploymentConfig() error {
	// Load deploy config
	deployConfig, err := config.LoadDeployConfig(d.projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load deploy config: %w", err)
	}

	// Validate deploy config for environment
	if err := deployConfig.Validate(d.Params.Env); err != nil {
		return fmt.Errorf("invalid deploy config: %w", err)
	}

	// Resolve network
	networkResolver := network.NewResolver(d.projectRoot)
	networkInfo, err := networkResolver.ResolveNetwork(d.Params.NetworkName)
	if err != nil {
		return fmt.Errorf("failed to resolve network: %w", err)
	}
	d.networkInfo = networkInfo

	// Generate environment variables
	envVars, err := deployConfig.GenerateEnvVars(d.Params.Env)
	if err != nil {
		return fmt.Errorf("failed to generate environment variables: %w", err)
	}

	// Add network-specific environment variables
	if networkInfo.RpcUrl != "" {
		envVars["RPC_URL"] = networkInfo.RpcUrl
	}
	envVars["CHAIN_ID"] = fmt.Sprintf("%d", networkInfo.ChainID)

	// Add deployment label if specified
	if d.Params.Label != "" {
		envVars["DEPLOYMENT_LABEL"] = d.Params.Label
	}

	d.envVars = envVars

	return nil
}

// BuildContracts runs forge build
func (d *DeploymentContext) BuildContracts() error {
	forgeExecutor := forgeExec.NewForge(d.projectRoot)
	if err := forgeExecutor.CheckInstallation(); err != nil {
		return fmt.Errorf("forge check failed: %w", err)
	}

	if err := forgeExecutor.Build(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	return nil
}

// PrepareContractDeployment validates a singleton contract deployment and returns whether a script was generated
func (d *DeploymentContext) PrepareContractDeployment() (bool, error) {
	// Resolve the contract
	contractInfo, err := interactive.ResolveContract(d.Params.ContractQuery, contracts.ProjectFilter())
	if err != nil {
		return false, fmt.Errorf("failed to resolve contract: %w", err)
	}

	// Update context with resolved contract name
	d.contractInfo = contractInfo

	// Check if deploy script exists using the generator's path logic
	generator := contracts.NewGenerator(d.projectRoot)
	scriptPath := generator.GetDeployScriptPath(contractInfo)
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		if d.Params.Predict {
			return false, fmt.Errorf("deploy script required but not found: %s", scriptPath)
		}

		fmt.Printf("\nDeploy script not found for %s (%s)\n", d.contractInfo.Name, d.ScriptPath)

		// Ask if user wants to generate the script
		selector := interactive.NewSelector()
		shouldGenerate, err := selector.PromptConfirm("Would you like to generate a deploy script?", true)
		if err != nil || !shouldGenerate {
			return false, fmt.Errorf("deploy script required but not found: %s", scriptPath)
		}

		// Generate the script interactively
		fmt.Printf("\nStarting interactive script generation...\n\n")
		interactiveGenerator := interactive.NewGenerator(d.projectRoot)
		if err := interactiveGenerator.GenerateDeployScriptForContract(contractInfo); err != nil {
			return false, fmt.Errorf("script generation failed: %w", err)
		}
		return true, nil
	}

	// Set script path
	d.ScriptPath = scriptPath

	return false, nil
}

// PrepareProxyDeployment validates a proxy deployment
func (d *DeploymentContext) PrepareProxyDeployment() error {
	// Resolve the contract
	contractInfo, err := interactive.ResolveContract(d.Params.ContractQuery, contracts.DefaultFilter())
	if err != nil {
		return fmt.Errorf("failed to resolve contract: %w", err)
	}

	// Update context with resolved contract name
	d.contractInfo = contractInfo

	// Check if proxy deploy script exists
	generator := contracts.NewGenerator(d.projectRoot)
	scriptPath := generator.GetProxyScriptPath(contractInfo)
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("proxy deploy script not found: %s", scriptPath)
	}

	// Set script path
	d.ScriptPath = scriptPath

	// // Get implementation info from registry
	// if ctx.ImplementationLabel == "" {
	// 	ctx.ImplementationLabel = "default"
	// }

	// // Determine implementation identifier
	// implIdentifier := ctx.ImplementationName
	// if ctx.Env != "" {
	// 	implIdentifier = fmt.Sprintf("%s/%s", ctx.Env, ctx.ImplementationName)
	// }
	// if ctx.ImplementationLabel != "default" {
	// 	implIdentifier = fmt.Sprintf("%s:%s", implIdentifier, ctx.ImplementationLabel)
	// }

	// TODO: Validate implementation exists in registry
	// This would require access to the registry manager

	return nil
}

// PrepareLibraryDeployment validates a library deployment
func (d *DeploymentContext) PrepareLibraryDeployment() error {
	// Resolve the library
	contractInfo, err := interactive.ResolveContract(d.Params.ContractQuery, contracts.DefaultFilter())
	if err != nil {
		return fmt.Errorf("failed to resolve library: %w", err)
	}

	// Update context with resolved contract name
	d.contractInfo = contractInfo

	return nil
}
