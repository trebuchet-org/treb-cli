package interactive

import (
	"fmt"
	"strings"

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
func (g *Generator) pickContract(contractNameOrPath string) (*contracts.ContractInfo, error) {
	// Use the global indexer
	indexer, err := contracts.GetGlobalIndexer(g.projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize contract indexer: %w", err)
	}

	var selected *contracts.ContractInfo

	// If a contract name/path was specified, try to resolve it
	if contractNameOrPath != "" {
		// Try direct lookup first
		if contract, err := indexer.GetContract(contractNameOrPath); err == nil {
			selected = contract
			fmt.Printf("Using specified contract: %s\n\n", contract.Name)
		} else {
			// Try by name
			contracts := indexer.GetContractsByName(contractNameOrPath)
			if len(contracts) == 0 {
				return nil, fmt.Errorf("contract not found: %s", contractNameOrPath)
			} else if len(contracts) == 1 {
				selected = contracts[0]
				fmt.Printf("Using specified contract: %s\n\n", contracts[0].Name)
			} else {
				// Multiple contracts with same name - let user pick
				var options []string
				for _, c := range contracts {
					options = append(options, fmt.Sprintf("%s (%s)", c.Name, c.Path))
				}
				_, selectedIndex, err := g.selector.SelectOption("Multiple contracts found. Select one:", options, 0)
				if err != nil {
					return nil, fmt.Errorf("contract selection failed: %w", err)
				}
				selected = contracts[selectedIndex]
			}
		}
	} else {
		// No contract specified - show picker with deployable contracts
		deployableContracts := indexer.GetDeployableContracts()
		if len(deployableContracts) == 0 {
			return nil, fmt.Errorf("no deployable contracts found")
		}

		var options []string
		for _, c := range deployableContracts {
			if c.Version != "" {
				options = append(options, fmt.Sprintf("%s (%s)", c.Name, c.Version))
			} else {
				options = append(options, c.Name)
			}
		}

		_, selectedIndex, err := g.selector.SelectOption("Select contract:", options, 0)
		if err != nil {
			return nil, fmt.Errorf("contract selection failed: %w", err)
		}

		selected = deployableContracts[selectedIndex]
	}

	// Convert to ContractInfo
	contractInfo := &contracts.ContractInfo{
		Name:      selected.Name,
		Path:      selected.Path,
		IsLibrary: selected.IsLibrary,
	}

	return contractInfo, nil
}

// pickProxyContract selects a proxy contract from available proxy contracts
func (g *Generator) pickProxyContract() (*contracts.ContractInfo, error) {
	// Use the global indexer (always includes libraries)
	indexer, err := contracts.GetGlobalIndexer(g.projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize contract indexer: %w", err)
	}

	// Get all proxy contracts (including from libraries)
	filter := contracts.DefaultFilter() // Include libraries for proxy contracts
	proxyContracts := indexer.GetProxyContractsFiltered(filter)
	if len(proxyContracts) == 0 {
		return nil, fmt.Errorf("no proxy contracts found. Make sure you have proxy contracts in your project")
	}

	// Build options
	var options []string
	for _, proxy := range proxyContracts {
		option := proxy.Name
		if proxy.Path != "" {
			option = fmt.Sprintf("%s (%s)", proxy.Name, proxy.Path)
		}
		options = append(options, option)
	}

	_, selectedIndex, err := g.selector.SelectOption("Select proxy contract:", options, 0)
	if err != nil {
		return nil, fmt.Errorf("proxy selection failed: %w", err)
	}

	selected := proxyContracts[selectedIndex]

	return selected, nil
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
func (g *Generator) GenerateDeployScript(contractNameOrPath string) error {
	// Step 1: Pick or resolve contract
	var contractInfo *contracts.ContractInfo
	var err error

	// Try to resolve the contract by name or path
	contractInfo, err = ResolveContract(contractNameOrPath, contracts.ProjectFilter())
	if err != nil {
		return fmt.Errorf("failed to resolve contract: %w", err)
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

// GenerateDeployScriptForContract generates a deploy script for a specific contract
func (g *Generator) GenerateDeployScriptForContract(contractInfo *contracts.ContractInfo) error {
	// Step 1: Pick deployment strategy
	strategy, err := g.pickDeploymentStrategy()
	if err != nil {
		return err
	}

	// Step 2: Generate the script
	generator := contracts.NewGenerator(g.projectRoot)
	if err := generator.GenerateDeployScript(contractInfo, strategy); err != nil {
		return fmt.Errorf("script generation failed: %w", err)
	}

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

// GenerateProxyDeployScript interactively generates a proxy deploy script
func (g *Generator) GenerateProxyDeployScript(contractName string) error {
	// Step 1: Pick implementation contract
	contractInfo, err := g.pickContract(contractName)
	if err != nil {
		return err
	}

	// Step 2: Pick proxy contract
	proxyInfo, err := g.pickProxyContract()
	if err != nil {
		return err
	}

	// Step 3: Determine proxy type based on selected proxy
	proxyType := g.determineProxyType(proxyInfo.Name)

	// Step 4: Pick deployment strategy
	strategy, err := g.pickDeploymentStrategy()
	if err != nil {
		return err
	}

	// Step 5: Generate the script
	generator := contracts.NewGenerator(g.projectRoot)
	if err := generator.GenerateProxyDeployScript(contractInfo, proxyInfo, strategy, proxyType); err != nil {
		return fmt.Errorf("proxy script generation failed: %w", err)
	}

	fmt.Printf("Implementation: %s\n", contractInfo.Name)
	fmt.Printf("Proxy: %s\n", proxyInfo.Name)
	fmt.Printf("Strategy: %s\n", strategy)

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

// determineProxyType determines the proxy type based on the proxy contract name
func (g *Generator) determineProxyType(proxyName string) contracts.ProxyType {
	if strings.Contains(proxyName, "TransparentUpgradeable") {
		return contracts.ProxyTypeOZTransparent
	} else if strings.Contains(proxyName, "ERC1967") || strings.Contains(proxyName, "UUPS") {
		return contracts.ProxyTypeOZUUPS
	}
	return contracts.ProxyTypeCustom
}
