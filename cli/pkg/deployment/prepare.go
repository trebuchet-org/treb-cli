package deployment

import (
	"fmt"
	"os"

	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	forgeExec "github.com/trebuchet-org/treb-cli/cli/pkg/forge"
	"github.com/trebuchet-org/treb-cli/cli/pkg/interactive"
	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
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
	envVars["CHAIN_ID"] = fmt.Sprintf("%d", networkInfo.ChainID())

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
	implementationInfo, err := interactive.ResolveContract(d.Params.ImplementationQuery, contracts.DefaultFilter())
	if err != nil {
		return fmt.Errorf("failed to resolve contract: %w", err)
	}

	// Update context with resolved contract name
	d.implementationInfo = implementationInfo

	// Check if proxy deploy script exists
	generator := contracts.NewGenerator(d.projectRoot)
	scriptPath := generator.GetProxyScriptPath(implementationInfo)
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("proxy deploy script not found: %s", scriptPath)
	}

	// Set script path
	d.ScriptPath = scriptPath

	// Parse the proxy artifact path from the deployment script
	proxyArtifactPath, err := parseProxyArtifactPath(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to parse proxy artifact from script: %w", err)
	}

	// Get proxy contract info using the artifact path
	indexer, err := contracts.GetGlobalIndexer(d.projectRoot)
	if err != nil {
		return fmt.Errorf("failed to initialize contract indexer: %w", err)
	}

	proxyContractInfo, err := indexer.GetContract(proxyArtifactPath)
	if err != nil {
		return fmt.Errorf("failed to get proxy contract info for %s: %w", proxyArtifactPath, err)
	}

	// Set the proxy contract as the contract being deployed
	d.contractInfo = proxyContractInfo

	// Pass current network and environment context to narrow down results
	// Use ResolveImplementationDeployment to filter out proxy deployments
	deployment, err := interactive.ResolveImplementationDeployment(d.Params.TargetQuery, d.registryManager, d.networkInfo.ChainID(), d.Params.Env)
	if err != nil {
		return fmt.Errorf("failed to resolve target deployment: %w", err)
	}

	// Verify the deployment is on the same network (double-check since query should handle this)
	deploymentChainID := deployment.ChainID
	currentChainID := fmt.Sprintf("%d", d.networkInfo.ChainID())
	if deploymentChainID != currentChainID {
		return fmt.Errorf("target deployment is on network %s (chain %s) but deploying to network %s (chain %s)",
			deployment.NetworkName, deploymentChainID, d.networkInfo.Name, currentChainID)
	}

	// Filter to only deployed contracts
	if deployment.Entry.Deployment.Status != types.StatusExecuted {
		return fmt.Errorf("target deployment %s is not yet deployed (status: %s)", deployment.Entry.FQID, deployment.Entry.Deployment.Status)
	}

	// Set IMPLEMENTATION_ADDRESS environment variable
	d.envVars["IMPLEMENTATION_ADDRESS"] = deployment.Entry.Address.Hex()
	d.targetDeploymentFQID = deployment.Entry.FQID

	fmt.Printf("Using implementation: %s at %s\n", deployment.Entry.ShortID, deployment.Entry.Address.Hex())

	return nil
}

// PrepareLibraryDeployment validates a library deployment
func (d *DeploymentContext) PrepareLibraryDeployment() error {
	// Resolve the library - use AllFilter to include libraries
	libraryFilter := contracts.QueryFilter{
		IncludeLibraries: true,
		IncludeAbstract:  false,
		IncludeInterface: false,
	}
	contractInfo, err := interactive.ResolveContract(d.Params.ContractQuery, libraryFilter)
	if err != nil {
		return fmt.Errorf("failed to resolve library: %w", err)
	}

	// Verify it's actually a library
	if !contractInfo.IsLibrary {
		return fmt.Errorf("contract '%s' is not a library", contractInfo.Name)
	}

	// Update context with resolved contract name
	d.contractInfo = contractInfo

	// Set the script path to the standard library deployment script
	d.ScriptPath = "lib/treb-sol/src/LibraryDeployment.s.sol:LibraryDeployment"

	// Set required environment variables for library deployment
	d.envVars["LIBRARY_NAME"] = contractInfo.Name
	// For vm.getCode(), we need the format "filename.sol:ContractName"
	// d.envVars["LIBRARY_ARTIFACT_PATH"] = fmt.Sprintf("%s:%s", contractInfo.Path, contractInfo.Name)
	d.envVars["LIBRARY_ARTIFACT_PATH"] = contractInfo.ArtifactPath

	return nil
}
