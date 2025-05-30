package generator

import (
	"fmt"

	"github.com/trebuchet-org/treb-cli/cli/pkg/abi"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
)

// Generator handles script generation with resolved contracts
type Generator struct {
	projectRoot  string
	contractsGen *contracts.Generator
}

// NewGenerator creates a new generator
func NewGenerator(projectRoot string) *Generator {
	return &Generator{
		projectRoot:  projectRoot,
		contractsGen: contracts.NewGenerator(projectRoot),
	}
}

// GenerateDeployScript generates a deploy script for a resolved contract
func (g *Generator) GenerateDeployScript(contractInfo *contracts.ContractInfo, strategy contracts.DeployStrategy) error {
	// Generate the script
	if err := g.contractsGen.GenerateDeployScript(contractInfo, strategy); err != nil {
		return fmt.Errorf("script generation failed: %w", err)
	}

	fmt.Printf("Generated deployment script for %s using %s strategy\n", contractInfo.Name, strategy)

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

	// Show next steps
	scriptPath := g.contractsGen.GetDeployScriptPath(contractInfo)
	fmt.Printf("\nGenerated deploy script:\n")
	fmt.Printf("  %s\n", scriptPath)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("1. Review and customize the script if needed\n")
	fmt.Printf("2. Deploy with: treb deploy %s --network <network>\n", contractInfo.Name)

	return nil
}

// GenerateProxyScript generates a proxy deploy script with resolved inputs
func (g *Generator) GenerateProxyScript(implementationInfo *contracts.ContractInfo, proxyInfo *contracts.ContractInfo, strategy contracts.DeployStrategy, proxyType contracts.ProxyType) error {
	// Generate the script
	if err := g.contractsGen.GenerateProxyDeployScript(implementationInfo, proxyInfo, strategy, proxyType); err != nil {
		return fmt.Errorf("proxy script generation failed: %w", err)
	}

	fmt.Printf("Generated proxy deployment script for %s using %s strategy\n", implementationInfo.Name, strategy)
	fmt.Printf("Implementation: %s\n", implementationInfo.Name)
	fmt.Printf("Proxy: %s (%s)\n", proxyInfo.Name, proxyType)

	// Show initializer info
	abiParser := abi.NewParser(g.projectRoot)
	if contractABI, err := abiParser.ParseContractABI(implementationInfo.Name); err == nil {
		if initMethod := abiParser.FindInitializeMethod(contractABI); initMethod != nil {
			fmt.Printf("Initialize method detected: %s\n", initMethod.Name)
			fmt.Printf("Arguments will be automatically configured in _getProxyInitializer()\n")
		} else {
			fmt.Printf("No initialize method found - proxy will be deployed without initialization\n")
		}
	}

	// Show next steps
	scriptPath := g.contractsGen.GetProxyScriptPath(implementationInfo)
	fmt.Printf("\nGenerated proxy deploy script:\n")
	fmt.Printf("  %s\n", scriptPath)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("1. Deploy the implementation: treb deploy %s --network <network>\n", implementationInfo.Name)
	fmt.Printf("2. Deploy the proxy: treb deploy %sProxy --network <network>\n", implementationInfo.Name)

	return nil
}
