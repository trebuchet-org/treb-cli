package deployment

import (
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/abi"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	forgeExec "github.com/trebuchet-org/treb-cli/cli/pkg/forge"
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

	// Validate deploy config for namespace
	if err := deployConfig.Validate(d.Params.Namespace); err != nil {
		return fmt.Errorf("invalid deploy config: %w", err)
	}

	// Resolve network
	networkResolver := network.NewResolver(d.projectRoot)
	networkInfo, err := networkResolver.ResolveNetwork(d.Params.NetworkName)
	if err != nil {
		return fmt.Errorf("failed to resolve network: %w", err)
	}
	d.networkInfo = networkInfo

	// Build deployment config directly
	d.deploymentConfig = &abi.DeploymentConfig{
		Namespace: d.Params.Namespace,
		Label:     d.Params.Label,
	}

	// Build executor config if sender is specified
	if d.Params.Sender != "" {
		// Validate sender configuration
		if err := deployConfig.ValidateSender("treb", d.Params.Sender); err != nil {
			return fmt.Errorf("invalid sender configuration: %w", err)
		}

		sender, err := deployConfig.GetSender("treb", d.Params.Sender)
		if err != nil {
			return fmt.Errorf("failed to get sender config: %w", err)
		}

		// Build executor config based on sender type
		executorConfig := abi.ExecutorConfig{
			// Initialize big.Int fields to avoid nil pointer issues
			SenderPrivateKey:   big.NewInt(0),
			ProposerPrivateKey: big.NewInt(0),
		}

		switch sender.Type {
		case "private_key":
			executorConfig.SenderType = abi.SenderTypePrivateKey

			// Derive address from private key
			privKey := new(big.Int)
			privKey.SetString(sender.PrivateKey, 0) // Handle 0x prefix automatically
			executorConfig.SenderPrivateKey = privKey

			// Use helper to get address from private key
			address, err := config.GetAddressFromPrivateKey(sender.PrivateKey)
			if err != nil {
				return fmt.Errorf("failed to derive address from private key: %w", err)
			}
			executorConfig.Sender = common.HexToAddress(address)

		case "safe":
			executorConfig.SenderType = abi.SenderTypeSafe
			executorConfig.Sender = common.HexToAddress(sender.Safe)
			// TODO: Handle proposer configuration for Safe

		case "ledger":
			executorConfig.SenderType = abi.SenderTypeLedger
			executorConfig.SenderDerivationPath = sender.DerivationPath

			// Resolve address dynamically using cast
			address, err := config.GetLedgerAddress(sender.DerivationPath)
			if err != nil {
				return fmt.Errorf("failed to resolve ledger address: %w", err)
			}
			executorConfig.Sender = common.HexToAddress(address)
		}

		d.executorConfig = &executorConfig
		d.deploymentConfig.ExecutorConfig = executorConfig
	}

	// No env vars needed - we'll pass config directly to scripts

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
	// Resolve the contract using the resolver
	contractInfo, err := d.resolver.ResolveContractForImplementation(d.Params.ContractQuery)
	if err != nil {
		return false, fmt.Errorf("failed to resolve contract: %w", err)
	}

	// Update context with resolved contract name
	d.contractInfo = contractInfo

	// Check if deploy script exists using the generator's path logic
	contractsGen := contracts.NewGenerator(d.projectRoot)
	scriptPath := contractsGen.GetDeployScriptPath(contractInfo)
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return false, fmt.Errorf("deploy script not found at %s\n\nTo generate a deploy script, run:\n  treb gen deploy %s", scriptPath, contractInfo.Name)
	}

	// Set script path
	d.ScriptPath = scriptPath

	// Check and resolve required libraries
	resolvedLibs, err := d.checkAndResolveLibraries(contractInfo)
	if err != nil {
		return false, fmt.Errorf("failed to resolve libraries: %w", err)
	}
	d.resolvedLibraries = resolvedLibs
	d.deploymentConfig.DeploymentType = abi.DeploymentTypeSingleton

	return false, nil
}

// PrepareProxyDeployment validates a proxy deployment
func (d *DeploymentContext) PrepareProxyDeployment() error {
	// Resolve the contract using the resolver
	implementationInfo, err := d.resolver.ResolveContractForImplementation(d.Params.ImplementationQuery)
	if err != nil {
		return fmt.Errorf("failed to resolve contract: %w", err)
	}

	// Update context with resolved contract name
	d.implementationInfo = implementationInfo

	// Check if proxy deploy script exists
	contractsGen := contracts.NewGenerator(d.projectRoot)
	scriptPath := contractsGen.GetProxyScriptPath(implementationInfo)
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("proxy deploy script not found at %s\n\nTo generate a proxy deploy script, run:\n  treb gen proxy %s", scriptPath, implementationInfo.Name)
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

	// Check and resolve required libraries for the proxy contract
	resolvedLibs, err := d.checkAndResolveLibraries(proxyContractInfo)
	if err != nil {
		return fmt.Errorf("failed to resolve libraries: %w", err)
	}
	d.resolvedLibraries = resolvedLibs

	// Pass current network and environment context to narrow down results
	// Use ResolveDeploymentForProxy to filter out proxy deployments
	deployment, err := d.resolver.ResolveDeploymentForProxy(d.Params.TargetQuery, d.registryManager, d.networkInfo.ChainID(), d.Params.Namespace)
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

	// Build proxy deployment config
	if d.deploymentConfig == nil {
		return fmt.Errorf("deployment config not initialized")
	}

	// Set the deployment type to proxy BEFORE creating the proxy deployment config
	d.deploymentConfig.DeploymentType = abi.DeploymentTypeProxy

	// Create proxy deployment config with implementation address
	d.proxyDeploymentConfig = &abi.ProxyDeploymentConfig{
		ImplementationAddress: deployment.Entry.Address,
		DeploymentConfig:      abi.ConvertDeploymentConfigToProxy(*d.deploymentConfig),
	}

	d.targetDeploymentFQID = deployment.Entry.FQID
	fmt.Printf("Using implementation: %s at %s\n", deployment.Entry.ShortID, deployment.Entry.Address.Hex())

	return nil
}

// PrepareLibraryDeployment validates a library deployment
func (d *DeploymentContext) PrepareLibraryDeployment() error {
	// Resolve the library using the resolver
	contractInfo, err := d.resolver.ResolveContractForLibrary(d.Params.ContractQuery)
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
	d.ScriptPath = "script/deploy/DeployLibrary.s.sol"

	// Build library deployment config
	if d.executorConfig == nil {
		return fmt.Errorf("executor config not initialized for library deployment")
	}

	d.libraryDeploymentConfig = &abi.LibraryDeploymentConfig{
		ExecutorConfig:      abi.ConvertExecutorConfigToLibrary(*d.executorConfig),
		LibraryArtifactPath: contractInfo.ArtifactPath,
	}

	return nil
}
