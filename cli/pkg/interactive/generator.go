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
	GenerateTypeDeploy GenerateType = "deploy"
	GenerateTypeDeployProxy GenerateType = "deploy-proxy"
	GenerateTypeMigration GenerateType = "migration"
	GenerateTypeUpgrade GenerateType = "upgrade"
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

// promptString prompts for a string input with a default value
func (g *Generator) promptString(prompt, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	
	var input string
	fmt.Scanln(&input)
	input = strings.TrimSpace(input)
	
	if input == "" && defaultValue != "" {
		return defaultValue
	}
	return input
}

// GenerateDeployScript interactively generates a deploy script
func (g *Generator) GenerateDeployScript() error {
	return g.GenerateDeployScriptForContract("")
}

// GenerateDeployScriptForContract generates a deploy script for a specific contract
func (g *Generator) GenerateDeployScriptForContract(contractName string) error {
	fmt.Println("Interactive Deploy Script Generator")
	fmt.Println("===================================")
	
	var selectedContract contracts.ContractDiscovery
	var contractFound bool
	
	// Step 1: Discover all contracts in src directory
	discovery := contracts.NewDiscovery(g.projectRoot)
	discoveredContracts, err := discovery.DiscoverContracts()
	if err != nil {
		return fmt.Errorf("failed to discover contracts: %w", err)
	}
	
	if len(discoveredContracts) == 0 {
		return fmt.Errorf("no contracts found in src/ directory")
	}
	
	// Step 2: Select contract (or use specified contract)
	if contractName != "" {
		// Find the specific contract
		for _, contract := range discoveredContracts {
			if contract.Name == contractName {
				selectedContract = contract
				contractFound = true
				break
			}
		}
		
		if !contractFound {
			return fmt.Errorf("contract %s not found in src/ directory", contractName)
		}
		
		fmt.Printf("Generating deploy script for: %s\n\n", contractName)
	} else {
		// Interactive selection
		fmt.Printf("Found %d contract(s) in src/\n\n", len(discoveredContracts))
		
		contractOptions := discovery.GetFormattedOptions(discoveredContracts)
		_, selectedIndex, err := g.selector.SimpleSelect("Select contract to deploy:", contractOptions, 0)
		if err != nil {
			return fmt.Errorf("contract selection failed: %w", err)
		}
		
		selectedContract = discoveredContracts[selectedIndex]
		contractFound = true
	}
	
	// Step 3: Validate the selected contract
	validator := contracts.NewValidator(g.projectRoot)
	contractInfo, err := validator.ValidateContract(selectedContract.Name)
	if err != nil {
		return fmt.Errorf("contract validation failed: %w", err)
	}
	
	fmt.Printf("\nSelected: %s\n", discovery.FormatContractOption(selectedContract))
	
	// Step 4: Choose deployment strategy
	strategies := []string{"CREATE2", "CREATE3"}
	strategyStr, _, err := g.selector.SimpleSelect("Select deployment strategy:", strategies, 1) // Default to CREATE3
	if err != nil {
		return fmt.Errorf("strategy selection failed: %w", err)
	}
	
	strategy, err := contracts.ValidateStrategy(strategyStr)
	if err != nil {
		return fmt.Errorf("invalid strategy: %w", err)
	}
	
	// Step 5: Version settings removed - using tags instead
	
	// Step 6: Generate the script
	fmt.Printf("\nGenerating deploy script for %s...\n", contractInfo.Name)
	
	generator := contracts.NewGenerator(g.projectRoot)
	if err := generator.GenerateDeployScript(contractInfo, strategy); err != nil {
		return fmt.Errorf("script generation failed: %w", err)
	}
	
	fmt.Printf("Deploy script generated successfully!\n")
	fmt.Printf("Strategy: %s\n", strategy)
	
	// Show constructor info
	fmt.Printf("\nConstructor Arguments:\n")
	if contractInfo != nil {
		// Try to parse ABI to show constructor info
		abiParser := abi.NewParser(g.projectRoot)
		if contractABI, err := abiParser.ParseContractABI(contractInfo.Name); err == nil {
			if contractABI.HasConstructor {
				fmt.Printf("Constructor arguments automatically detected and configured\n")
				fmt.Printf("You can customize the values in getConstructorArgs() method\n")
			} else {
				fmt.Printf("No constructor arguments required\n")
			}
		}
	}
	
	return nil
}

// GenerateMigrationScript interactively generates a migration script
func (g *Generator) GenerateMigrationScript() error {
	fmt.Println("🔄 Interactive Migration Script Generator")
	fmt.Println("=========================================")
	
	fmt.Println("⚠️  Migration script generation is not yet implemented.")
	fmt.Println("    This feature will allow you to generate scripts for:")
	fmt.Println("    - State migrations between contract versions")
	fmt.Println("    - Data transfer scripts")
	fmt.Println("    - Multi-step deployment workflows")
	
	return fmt.Errorf("migration script generation not yet implemented")
}

// GenerateUpgradeScript interactively generates an upgrade script
func (g *Generator) GenerateUpgradeScript() error {
	fmt.Println("⬆️  Interactive Upgrade Script Generator") 
	fmt.Println("========================================")
	
	fmt.Println("⚠️  Upgrade script generation is not yet implemented.")
	fmt.Println("    This feature will allow you to generate scripts for:")
	fmt.Println("    - Proxy upgrade workflows")
	fmt.Println("    - Implementation deployment and upgrade")
	fmt.Println("    - Multi-signature upgrade processes")
	
	return fmt.Errorf("upgrade script generation not yet implemented")
}

// GenerateProxyDeployScript interactively generates a proxy deploy script
func (g *Generator) GenerateProxyDeployScript() error {
	fmt.Println("Interactive Proxy Deploy Script Generator")
	fmt.Println("=========================================")
	
	// Step 1: Discover all contracts in src directory
	discovery := contracts.NewDiscovery(g.projectRoot)
	discoveredContracts, err := discovery.DiscoverContracts()
	if err != nil {
		return fmt.Errorf("failed to discover contracts: %w", err)
	}
	
	if len(discoveredContracts) == 0 {
		return fmt.Errorf("no contracts found in src/ directory")
	}
	
	// Step 2: Select implementation contract
	fmt.Printf("Found %d contract(s) in src/\n\n", len(discoveredContracts))
	
	contractOptions := discovery.GetFormattedOptions(discoveredContracts)
	_, selectedIndex, err := g.selector.SimpleSelect("Select implementation contract:", contractOptions, 0)
	if err != nil {
		return fmt.Errorf("contract selection failed: %w", err)
	}
	
	selectedContract := discoveredContracts[selectedIndex]
	
	// Step 3: Validate the selected contract
	validator := contracts.NewValidator(g.projectRoot)
	contractInfo, err := validator.ValidateContract(selectedContract.Name)
	if err != nil {
		return fmt.Errorf("contract validation failed: %w", err)
	}
	
	fmt.Printf("\nSelected: %s\n", discovery.FormatContractOption(selectedContract))
	
	// Step 4: Select proxy type
	proxyTypes := []string{
		"OZ-TransparentUpgradeable - OpenZeppelin Transparent Upgradeable Proxy",
		"OZ-UUPSUpgradeable - OpenZeppelin UUPS Upgradeable Proxy", 
		"Custom - Custom proxy implementation",
	}
	
	proxyTypeStr, proxyTypeIndex, err := g.selector.SimpleSelect("Select proxy type:", proxyTypes, 0)
	if err != nil {
		return fmt.Errorf("proxy type selection failed: %w", err)
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
	
	// Step 5: Choose deployment strategy
	strategies := []string{"CREATE2", "CREATE3"}
	strategyStr, _, err := g.selector.SimpleSelect("Select deployment strategy:", strategies, 1) // Default to CREATE3
	if err != nil {
		return fmt.Errorf("strategy selection failed: %w", err)
	}
	
	strategy, err := contracts.ValidateStrategy(strategyStr)
	if err != nil {
		return fmt.Errorf("invalid strategy: %w", err)
	}
	
	// Step 6: Generate the script
	fmt.Printf("\nGenerating proxy deploy script for %s...\n", contractInfo.Name)
	
	generator := contracts.NewGenerator(g.projectRoot)
	if err := generator.GenerateProxyDeployScript(contractInfo, strategy, proxyType); err != nil {
		return fmt.Errorf("proxy script generation failed: %w", err)
	}
	
	fmt.Printf("Proxy deploy script generated successfully!\n")
	fmt.Printf("Strategy: %s\n", strategy)
	fmt.Printf("Proxy Type: %s\n", proxyType)
	
	// Show initializer info
	fmt.Printf("\nInitializer Information:\n")
	// Try to parse ABI to show initializer info
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

// GenerateProxyDeployScriptForContract generates a proxy deploy script for a specific contract
func (g *Generator) GenerateProxyDeployScriptForContract(contractName string) error {
	fmt.Println("Proxy Deploy Script Generator")
	fmt.Println("=============================")
	
	// Step 1: Validate the contract exists
	validator := contracts.NewValidator(g.projectRoot)
	contractInfo, err := validator.ValidateContract(contractName)
	if err != nil {
		return fmt.Errorf("contract validation failed: %w", err)
	}
	
	if !contractInfo.Exists {
		return fmt.Errorf("contract %s not found in src/ directory", contractName)
	}
	
	fmt.Printf("\nGenerating proxy deploy script for: %s\n", contractName)
	
	// Step 2: Select proxy type
	proxyTypes := []string{
		"OZ-TransparentUpgradeable - OpenZeppelin Transparent Upgradeable Proxy",
		"OZ-UUPSUpgradeable - OpenZeppelin UUPS Upgradeable Proxy", 
		"Custom - Custom proxy implementation",
	}
	
	proxyTypeStr, proxyTypeIndex, err := g.selector.SimpleSelect("Select proxy type:", proxyTypes, 0)
	if err != nil {
		return fmt.Errorf("proxy type selection failed: %w", err)
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
	
	// Step 3: Choose deployment strategy
	strategies := []string{"CREATE2", "CREATE3"}
	strategyStr, _, err := g.selector.SimpleSelect("Select deployment strategy:", strategies, 1) // Default to CREATE3
	if err != nil {
		return fmt.Errorf("strategy selection failed: %w", err)
	}
	
	strategy, err := contracts.ValidateStrategy(strategyStr)
	if err != nil {
		return fmt.Errorf("invalid strategy: %w", err)
	}
	
	// Step 4: Generate the script
	fmt.Printf("\nGenerating proxy deploy script for %s...\n", contractInfo.Name)
	
	generator := contracts.NewGenerator(g.projectRoot)
	if err := generator.GenerateProxyDeployScript(contractInfo, strategy, proxyType); err != nil {
		return fmt.Errorf("proxy script generation failed: %w", err)
	}
	
	fmt.Printf("Proxy deploy script generated successfully!\n")
	fmt.Printf("Strategy: %s\n", strategy)
	fmt.Printf("Proxy Type: %s\n", proxyType)
	
	// Show initializer info
	fmt.Printf("\nInitializer Information:\n")
	// Try to parse ABI to show initializer info
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