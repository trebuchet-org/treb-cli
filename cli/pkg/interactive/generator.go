package interactive

import (
	"fmt"

	"github.com/trebuchet-org/treb-cli/cli/pkg/abi"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
)

// Generator handles interactive generation workflows
type Generator struct {
	projectRoot string
	selector    *Selector
}

// NewGenerator creates a new interactive generator
func NewGenerator(projectRoot string) *Generator {
	return &Generator{
		projectRoot: projectRoot,
		selector:    NewSelector(),
	}
}

// GenerateType represents different generation types
type GenerateType string

const (
	GenerateTypeDeploy      GenerateType = "deploy"
	GenerateTypeDeployProxy GenerateType = "deploy-proxy"
	GenerateTypeMigration   GenerateType = "migration"
	GenerateTypeUpgrade     GenerateType = "upgrade"
)

// GetAvailableTypes returns all available generation types
func GetAvailableTypes() []GenerateType {
	return []GenerateType{
		GenerateTypeDeploy,
		GenerateTypeDeployProxy,
		GenerateTypeMigration,
		GenerateTypeUpgrade,
	}
}

// pickContract selects a contract from available contracts, or validates a specific contract
func (g *Generator) pickContract(contractName string) (*contracts.ContractInfo, error) {
	// Step 1: Discover all contracts in src directory
	discovery := contracts.NewDiscovery(g.projectRoot)
	discoveredContracts, err := discovery.DiscoverContracts()
	if err != nil {
		return nil, fmt.Errorf("failed to discover contracts: %w", err)
	}

	if len(discoveredContracts) == 0 {
		return nil, fmt.Errorf("no contracts found in src/ directory")
	}

	var selectedContract contracts.ContractDiscovery

	// Step 2: Select contract (or use specified contract)
	if contractName != "" {
		// Find the specific contract
		found := false
		for _, contract := range discoveredContracts {
			if contract.Name == contractName {
				selectedContract = contract
				found = true
				break
			}
		}

		if !found {
			return nil, fmt.Errorf("contract %s not found in src/ directory", contractName)
		}

		fmt.Printf("Using specified contract: %s\n\n", contractName)
	} else {
		contractOptions := discovery.GetFormattedOptions(discoveredContracts)
		_, selectedIndex, err := g.selector.SelectOption("Select contract:", contractOptions, 0)
		if err != nil {
			return nil, fmt.Errorf("contract selection failed: %w", err)
		}

		selectedContract = discoveredContracts[selectedIndex]
	}

	// Step 3: Validate the selected contract
	validator := contracts.NewValidator(g.projectRoot)
	contractInfo, err := validator.ValidateContract(selectedContract.Name)
	if err != nil {
		return nil, fmt.Errorf("contract validation failed: %w", err)
	}

	fmt.Printf("Selected: %s\n", discovery.FormatContractOption(selectedContract))

	return contractInfo, nil
}

// pickProxyType selects a proxy type for proxy deployments
func (g *Generator) pickProxyType() (contracts.ProxyType, error) {
	proxyTypes := []string{
		"OZ-TransparentUpgradeable - OpenZeppelin Transparent Upgradeable Proxy",
		"OZ-UUPSUpgradeable - OpenZeppelin UUPS Upgradeable Proxy",
		"Custom - Custom proxy implementation",
	}

	proxyTypeStr, proxyTypeIndex, err := g.selector.SelectOption("Select proxy type:", proxyTypes, 0)
	if err != nil {
		return "", fmt.Errorf("proxy type selection failed: %w", err)
	}

	var proxyType contracts.ProxyType
	switch proxyTypeIndex {
	case 0:
		proxyType = contracts.ProxyTypeTransparent
	case 1:
		proxyType = contracts.ProxyTypeUUPS
	case 2:
		proxyType = contracts.ProxyTypeCustom
	}

	fmt.Printf("Selected proxy type: %s\n", proxyTypeStr)
	return proxyType, nil
}

// pickDeploymentStrategy selects a deployment strategy (CREATE2 or CREATE3)
func (g *Generator) pickDeploymentStrategy() (contracts.DeployStrategy, error) {
	strategies := []string{"CREATE2", "CREATE3"}
	strategyStr, _, err := g.selector.SelectOption("Select deployment strategy:", strategies, 1) // Default to CREATE3
	if err != nil {
		return "", fmt.Errorf("strategy selection failed: %w", err)
	}

	strategy, err := contracts.ValidateStrategy(strategyStr)
	if err != nil {
		return "", fmt.Errorf("invalid strategy: %w", err)
	}

	return strategy, nil
}

// GenerateDeployScript interactively generates a deploy script
func (g *Generator) GenerateDeployScript(contractName string) error {
	// Step 1: Pick contract
	contractInfo, err := g.pickContract(contractName)
	if err != nil {
		return err
	}

	// Step 2: Pick deployment strategy
	strategy, err := g.pickDeploymentStrategy()
	if err != nil {
		return err
	}

	// Step 3: Generate the script
	generator := contracts.NewGenerator(g.projectRoot)
	if err := generator.GenerateDeployScript(contractInfo, strategy); err != nil {
		return fmt.Errorf("script generation failed: %w", err)
	}

	fmt.Printf("Deploy script generated successfully!\n")
	fmt.Printf("Strategy: %s\n", strategy)

	// Show constructor info
	abiParser := abi.NewParser(g.projectRoot)
	if contractABI, err := abiParser.ParseContractABI(contractInfo.Name); err == nil {
		if contractABI.HasConstructor {
			fmt.Printf("Constructor arguments automatically detected and configured\n")
			fmt.Printf("You can customize the values in getConstructorArgs() method\n")
		} else {
			fmt.Printf("No constructor arguments required\n")
		}
	}

	return nil
}

// GenerateLibraryScript interactively generates a library deploy script
func (g *Generator) GenerateLibraryScript(libraryName string) error {
	// Step 1: Pick library
	libraryInfo, err := g.pickContract(libraryName)
	if err != nil {
		return err
	}

	// Step 2: Validate it's a library
	validator := contracts.NewValidator(g.projectRoot)
	if !validator.IsLibrary(libraryInfo.Name) {
		return fmt.Errorf("%s is not a library", libraryInfo.Name)
	}

	// Step 3: Generate the script (libraries always use CREATE2 for determinism)
	generator := contracts.NewGenerator(g.projectRoot)
	if err := generator.GenerateLibraryScript(libraryInfo); err != nil {
		return fmt.Errorf("library script generation failed: %w", err)
	}

	fmt.Printf("Library deploy script generated successfully!\n")
	fmt.Printf("Library: %s\n", libraryInfo.Name)
	fmt.Printf("Strategy: CREATE2 (deterministic)\n")
	fmt.Printf("\nLibraries are deployed globally (no environment) for cross-chain consistency.\n")

	return nil
}

// GenerateProxyDeployScript interactively generates a proxy deploy script
func (g *Generator) GenerateProxyDeployScript(contractName string) error {
	// Step 1: Pick implementation contract
	contractInfo, err := g.pickContract(contractName)
	if err != nil {
		return err
	}

	// Step 2: Pick proxy type
	proxyType, err := g.pickProxyType()
	if err != nil {
		return err
	}

	// Step 3: Pick deployment strategy
	strategy, err := g.pickDeploymentStrategy()
	if err != nil {
		return err
	}

	// Step 4: Generate the script
	generator := contracts.NewGenerator(g.projectRoot)
	if err := generator.GenerateProxyDeployScript(contractInfo, strategy, proxyType); err != nil {
		return fmt.Errorf("proxy script generation failed: %w", err)
	}

	fmt.Printf("Proxy deploy script generated successfully!\n")
	fmt.Printf("Strategy: %s\n", strategy)
	fmt.Printf("Proxy Type: %s\n", proxyType)

	// Show initializer info
	abiParser := abi.NewParser(g.projectRoot)
	if contractABI, err := abiParser.ParseContractABI(contractInfo.Name); err == nil {
		if initMethod := abiParser.FindInitializeMethod(contractABI); initMethod != nil {
			fmt.Printf("Initialize method detected: %s\n", initMethod.Name)
			fmt.Printf("Arguments will be automatically configured in _getProxyInitializer()\n")
		} else {
			fmt.Printf("No initialize method found - proxy will be deployed without initialization\n")
		}
	}

	return nil
}
