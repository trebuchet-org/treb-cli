package interactive

import (
	"fmt"

	"github.com/trebuchet-org/treb-cli/cli/pkg/abi"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
)

// GenerateType represents the type of script to generate
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

// Generator handles interactive script generation
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

// GenerateType returns available generate types for a generator
func (g *Generator) GenerateType() []GenerateType {
	return []GenerateType{
		GenerateTypeDeploy,
		GenerateTypeDeployProxy,
		GenerateTypeMigration,
		GenerateTypeUpgrade,
	}
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
		}
	}

	// Show next steps
	scriptPath := generator.GetDeployScriptPath(contractInfo)
	fmt.Printf("\nGenerated deploy script:\n")
	fmt.Printf("  %s\n", scriptPath)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("1. Review and customize the script if needed\n")
	fmt.Printf("2. Deploy with: treb deploy %s --network <network>\n", contractInfo.Name)

	return nil
}

// GenerateProxyScript interactively generates a proxy deploy script
func (g *Generator) GenerateProxyScript(contractNameOrPath string) error {
	// Step 1: Pick or resolve implementation contract
	contractInfo, err := ResolveContract(contractNameOrPath, contracts.ProjectFilter())
	if err != nil {
		return fmt.Errorf("failed to resolve contract: %w", err)
	}

	// Step 2: Pick proxy contract
	proxyContract, err := g.pickProxyContract()
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
	if err := generator.GenerateProxyDeployScript(contractInfo, proxyContract, strategy, contracts.ProxyTypeOZTransparent); err != nil {
		return fmt.Errorf("script generation failed: %w", err)
	}

	fmt.Printf("Strategy: %s\n", strategy)
	fmt.Printf("Proxy contract: %s\n", proxyContract.Name)

	// Show next steps
	scriptPath := generator.GetProxyScriptPath(contractInfo)
	fmt.Printf("\nGenerated proxy deploy script:\n")
	fmt.Printf("  %s\n", scriptPath)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("1. Deploy the implementation: treb deploy %s --network <network>\n", contractInfo.Name)
	fmt.Printf("2. Deploy the proxy: treb deploy proxy %s --network <network>\n", contractInfo.Name)

	return nil
}