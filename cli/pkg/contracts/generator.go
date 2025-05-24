package contracts

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/trebuchet-org/treb-cli/cli/pkg/abi"
)

// DeployStrategy represents the deployment strategy
type DeployStrategy string

const (
	StrategyCreate2 DeployStrategy = "CREATE2"
	StrategyCreate3 DeployStrategy = "CREATE3"
)

// ProxyType represents the type of proxy pattern
type ProxyType string

const (
	ProxyTypeTransparent ProxyType = "TransparentUpgradeable"
	ProxyTypeUUPS        ProxyType = "UUPSUpgradeable"
	ProxyTypeCustom      ProxyType = "Custom"
)

// ScriptTemplate contains data for generating deploy scripts
type ScriptTemplate struct {
	ContractName        string
	SolidityFile        string
	Strategy            DeployStrategy
	Version             string
	ImportPath          string
	TargetVersion       string // Solidity version of the target contract
	VersionMismatch     bool   // True if target version differs from 0.8
	UseTypeCreationCode bool   // True if we should use type().creationCode
	HasConstructor      bool   // True if contract has constructor
	ConstructorVars     string // Variable declarations for constructor args
	ConstructorEncode   string // abi.encode call for constructor args
}

// ProxyScriptTemplate contains data for generating proxy deploy scripts
type ProxyScriptTemplate struct {
	ImplementationName  string
	SolidityFile        string
	Strategy            DeployStrategy
	ProxyType           ProxyType
	ImportPath          string
	ProxyImportPath     string // Import path for the proxy contract
	HasInitializer      bool   // True if contract has initialize method
	InitializerVars     string // Variable declarations for initializer args
	InitializerEncode   string // abi.encodeWithSelector call for initializer
}

// Generator handles deploy script generation
type Generator struct {
	projectRoot string
}

// NewGenerator creates a new script generator
func NewGenerator(projectRoot string) *Generator {
	return &Generator{
		projectRoot: projectRoot,
	}
}

// GenerateDeployScript creates a new deploy script from template
func (g *Generator) GenerateDeployScript(contractInfo *ContractInfo, strategy DeployStrategy) error {
	// Ensure script/deploy directory exists
	scriptDir := filepath.Join(g.projectRoot, "script", "deploy")
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		return fmt.Errorf("failed to create script/deploy directory: %w", err)
	}

	// Parse target contract version
	targetVersion, err := g.parseContractVersion(contractInfo.FilePath)
	if err != nil {
		// If we can't parse version, assume version mismatch for safety
		targetVersion = "unknown"
	}

	// Check for version compatibility
	versionMismatch := !strings.HasPrefix(targetVersion, "0.8")
	useTypeCreationCode := strings.HasPrefix(targetVersion, "0.8") && !versionMismatch

	// Parse ABI for constructor information
	abiParser := abi.NewParser(g.projectRoot)
	contractABI, err := abiParser.ParseContractABI(contractInfo.Name)
	if err != nil {
		// If ABI parsing fails, assume no constructor for safety
		contractABI = &abi.ContractABI{HasConstructor: false}
	}

	// Generate constructor argument code
	constructorVars, constructorEncode := abiParser.GenerateConstructorArgs(contractABI)

	// Prepare template data
	templateData := ScriptTemplate{
		ContractName: contractInfo.Name,
		SolidityFile: contractInfo.SolidityFile,
		Strategy:     strategy,
		// Version removed - using tags instead
		ImportPath:          fmt.Sprintf("../../src/%s", contractInfo.SolidityFile),
		TargetVersion:       targetVersion,
		VersionMismatch:     versionMismatch,
		UseTypeCreationCode: useTypeCreationCode,
		HasConstructor:      contractABI.HasConstructor,
		ConstructorVars:     constructorVars,
		ConstructorEncode:   constructorEncode,
	}

	// Get template content
	templateContent := g.getDeployScriptTemplate(versionMismatch)

	// Parse and execute template
	tmpl, err := template.New("deploy").Parse(templateContent)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Check if deploy script already exists
	outputPath := filepath.Join(scriptDir, fmt.Sprintf("Deploy%s.s.sol", contractInfo.Name))
	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("deploy script already exists: %s\nUse a different contract name or remove the existing script", outputPath)
	}

	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create deploy script: %w", err)
	}
	defer file.Close()

	// Execute template
	if err := tmpl.Execute(file, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	fmt.Printf("Generated deploy script: %s\n", outputPath)
	return nil
}

// getDeployScriptTemplate returns the appropriate template based on version compatibility
func (g *Generator) getDeployScriptTemplate(versionMismatch bool) string {
	if versionMismatch {
		return g.getCrossVersionTemplate()
	}
	return g.getSameVersionTemplate()
}

// getSameVersionTemplate returns template for same-version deployments
func (g *Generator) getSameVersionTemplate() string {
	return `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {ContractDeployment, DeployStrategy} from "treb-sol/ContractDeployment.sol";
import { {{.ContractName}} } from "{{.ImportPath}}";

/**
 * @title Deploy{{.ContractName}}
 * @notice Deployment script for {{.ContractName}} contract
 * @dev Generated automatically by treb
 */
contract Deploy{{.ContractName}} is ContractDeployment {
    constructor() ContractDeployment(
        "{{.ContractName}}",
        DeployStrategy.{{.Strategy}}
    ) {}

{{if .UseTypeCreationCode}}    /// @notice Get contract bytecode using type().creationCode
    function _getContractBytecode() internal pure override returns (bytes memory) {
        return type({{.ContractName}}).creationCode;
    }
{{end}}
{{if .HasConstructor}}    /// @notice Get constructor arguments
    function _getConstructorArgs() internal pure override returns (bytes memory) {
        // Constructor arguments detected from ABI
{{.ConstructorVars}}
        {{.ConstructorEncode}}
    }
{{end}}
}`
}

// getCrossVersionTemplate returns template for cross-version deployments
func (g *Generator) getCrossVersionTemplate() string {
	return `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {ContractDeployment, DeployStrategy} from "treb-sol/ContractDeployment.sol";
// Target contract uses Solidity {{.TargetVersion}}, which is incompatible with this deployment script (0.8)
// Import commented out to avoid version conflicts. Using artifact-based deployment instead.
// import "{{.ImportPath}}";

/**
 * @title Deploy{{.ContractName}}
 * @notice Deployment script for {{.ContractName}} contract
 * @dev Generated automatically by treb
 * @dev Target contract version: {{.TargetVersion}} (cross-version deployment)
 */
contract Deploy{{.ContractName}} is ContractDeployment {
    constructor() ContractDeployment(
        "{{.ContractName}}",
        DeployStrategy.{{.Strategy}}
    ) {}

{{if .HasConstructor}}    /// @notice Get constructor arguments
    function _getConstructorArgs() internal pure override returns (bytes memory) {
        // Constructor arguments detected from ABI
{{.ConstructorVars}}
        {{.ConstructorEncode}}
    }
{{end}}
}`
}

// parseContractVersion extracts the Solidity version from a contract file
func (g *Generator) parseContractVersion(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read contract file: %w", err)
	}

	// Look for pragma solidity version
	// Matches patterns like: pragma solidity ^0.8.0; or pragma solidity 0.7.6;
	re := regexp.MustCompile(`pragma\s+solidity\s+[\^~>=<]*([0-9]+\.[0-9]+(?:\.[0-9]+)?)`)
	matches := re.FindSubmatch(content)
	if len(matches) > 1 {
		return string(matches[1]), nil
	}

	return "", fmt.Errorf("could not find pragma solidity version")
}

// ValidateStrategy checks if the provided strategy is valid
func ValidateStrategy(strategy string) (DeployStrategy, error) {
	upper := strings.ToUpper(strategy)
	switch upper {
	case "CREATE2":
		return StrategyCreate2, nil
	case "CREATE3":
		return StrategyCreate3, nil
	default:
		return "", fmt.Errorf("invalid strategy: %s (must be CREATE2 or CREATE3)", strategy)
	}
}

// GenerateProxyDeployScript creates a new proxy deploy script from template
func (g *Generator) GenerateProxyDeployScript(contractInfo *ContractInfo, strategy DeployStrategy, proxyType ProxyType) error {
	// Ensure script/deploy directory exists
	scriptDir := filepath.Join(g.projectRoot, "script", "deploy")
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		return fmt.Errorf("failed to create script/deploy directory: %w", err)
	}

	// Parse ABI for initializer information
	abiParser := abi.NewParser(g.projectRoot)
	contractABI, err := abiParser.ParseContractABI(contractInfo.Name)
	if err != nil {
		// If ABI parsing fails, assume no initializer for safety
		contractABI = &abi.ContractABI{Methods: []abi.Method{}}
	}

	// Find initialize method
	initMethod := abiParser.FindInitializeMethod(contractABI)
	hasInitializer := initMethod != nil

	// Generate initializer argument code
	var initializerVars, initializerEncode string
	if hasInitializer {
		initializerVars, initializerEncode = abiParser.GenerateInitializerArgs(initMethod)
	}

	// Determine proxy import path based on type
	var proxyImportPath string
	switch proxyType {
	case ProxyTypeTransparent:
		proxyImportPath = "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol"
	case ProxyTypeUUPS:
		proxyImportPath = "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol"
	case ProxyTypeCustom:
		// For custom proxy, we'll use a placeholder that users need to update
		proxyImportPath = "../../src/YourCustomProxy.sol"
	}

	// Prepare template data
	templateData := ProxyScriptTemplate{
		ImplementationName: contractInfo.Name,
		SolidityFile:       contractInfo.SolidityFile,
		Strategy:           strategy,
		ProxyType:          proxyType,
		ImportPath:         fmt.Sprintf("../../src/%s", contractInfo.SolidityFile),
		ProxyImportPath:    proxyImportPath,
		HasInitializer:     hasInitializer,
		InitializerVars:    initializerVars,
		InitializerEncode:  initializerEncode,
	}

	// Get template content
	templateContent := g.getProxyDeployScriptTemplate(proxyType)

	// Parse and execute template
	tmpl, err := template.New("proxy-deploy").Parse(templateContent)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Check if deploy script already exists
	outputPath := filepath.Join(scriptDir, fmt.Sprintf("Deploy%sProxy.s.sol", contractInfo.Name))
	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("proxy deploy script already exists: %s\nUse a different contract name or remove the existing script", outputPath)
	}

	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create proxy deploy script: %w", err)
	}
	defer file.Close()

	// Execute template
	if err := tmpl.Execute(file, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	fmt.Printf("Generated proxy deploy script: %s\n", outputPath)
	return nil
}

// getProxyDeployScriptTemplate returns the appropriate proxy template
func (g *Generator) getProxyDeployScriptTemplate(proxyType ProxyType) string {
	switch proxyType {
	case ProxyTypeTransparent:
		return g.getTransparentProxyTemplate()
	case ProxyTypeUUPS:
		return g.getUUPSProxyTemplate()
	case ProxyTypeCustom:
		return g.getCustomProxyTemplate()
	default:
		return g.getTransparentProxyTemplate() // Default to transparent
	}
}

// getTransparentProxyTemplate returns template for OpenZeppelin Transparent proxy
func (g *Generator) getTransparentProxyTemplate() string {
	return `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {ProxyDeployment, DeployStrategy} from "treb-sol/ProxyDeployment.sol";
import {TransparentUpgradeableProxy} from "{{.ProxyImportPath}}";
import { {{.ImplementationName}} } from "{{.ImportPath}}";

/**
 * @title Deploy{{.ImplementationName}}Proxy
 * @notice Deployment script for {{.ImplementationName}} with Transparent Upgradeable Proxy
 * @dev Generated automatically by treb
 */
contract Deploy{{.ImplementationName}}Proxy is ProxyDeployment {
    constructor() ProxyDeployment(
        "{{.ImplementationName}}",
        DeployStrategy.{{.Strategy}}
    ) {}

    /// @notice Get contract bytecode for the proxy
    function _getContractBytecode() internal pure override returns (bytes memory) {
        return type(TransparentUpgradeableProxy).creationCode;
    }

    /// @notice Get constructor arguments - override to include admin parameter
    function _getConstructorArgs() internal view override returns (bytes memory) {
        address implementation = getDeployment(_getImplementationIdentifier());
        address admin = executor; // Use executor as the ProxyAdmin owner
        bytes memory initData = _getProxyInitializer();
        return abi.encode(implementation, admin, initData);
    }

{{if .HasInitializer}}    /// @notice Get proxy initializer data
    function _getProxyInitializer() internal view override returns (bytes memory) {
        // Initialize method arguments detected from ABI
{{.InitializerVars}}
        {{.InitializerEncode}}
    }
{{else}}    /// @notice Get proxy initializer data
    function _getProxyInitializer() internal pure override returns (bytes memory) {
        // No initialize method detected - proxy will be deployed without initialization
        return "";
    }
{{end}}
}`
}

// getUUPSProxyTemplate returns template for OpenZeppelin UUPS proxy
func (g *Generator) getUUPSProxyTemplate() string {
	return `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {ProxyDeployment, DeployStrategy} from "treb-sol/ProxyDeployment.sol";
import {ERC1967Proxy} from "{{.ProxyImportPath}}";
import { {{.ImplementationName}} } from "{{.ImportPath}}";

/**
 * @title Deploy{{.ImplementationName}}Proxy
 * @notice Deployment script for {{.ImplementationName}} with UUPS Upgradeable Proxy
 * @dev Generated automatically by treb
 */
contract Deploy{{.ImplementationName}}Proxy is ProxyDeployment {
    constructor() ProxyDeployment(
        "{{.ImplementationName}}",
        DeployStrategy.{{.Strategy}}
    ) {}

    /// @notice Get contract bytecode for the proxy
    function _getContractBytecode() internal pure override returns (bytes memory) {
        return type(ERC1967Proxy).creationCode;
    }

{{if .HasInitializer}}    /// @notice Get proxy initializer data
    function _getProxyInitializer() internal view override returns (bytes memory) {
        // Initialize method arguments detected from ABI
{{.InitializerVars}}
        {{.InitializerEncode}}
    }
{{else}}    /// @notice Get proxy initializer data
    function _getProxyInitializer() internal pure override returns (bytes memory) {
        // No initialize method detected - proxy will be deployed without initialization
        return "";
    }
{{end}}
}`
}

// getCustomProxyTemplate returns template for custom proxy implementations
func (g *Generator) getCustomProxyTemplate() string {
	return `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {ProxyDeployment, DeployStrategy} from "treb-sol/ProxyDeployment.sol";
// TODO: Update this import to your custom proxy contract
import {YourCustomProxy} from "{{.ProxyImportPath}}";
import { {{.ImplementationName}} } from "{{.ImportPath}}";

/**
 * @title Deploy{{.ImplementationName}}Proxy
 * @notice Deployment script for {{.ImplementationName}} with Custom Proxy
 * @dev Generated automatically by treb
 * @dev TODO: Update the proxy import and bytecode method
 */
contract Deploy{{.ImplementationName}}Proxy is ProxyDeployment {
    constructor() ProxyDeployment(
        "{{.ImplementationName}}",
        DeployStrategy.{{.Strategy}}
    ) {}

    /// @notice Get contract bytecode for the proxy
    function _getContractBytecode() internal pure override returns (bytes memory) {
        // TODO: Update this to use your custom proxy contract
        // return type(YourCustomProxy).creationCode;
        revert("Update this method to return your custom proxy bytecode");
    }

{{if .HasInitializer}}    /// @notice Get proxy initializer data
    function _getProxyInitializer() internal pure override returns (bytes memory) {
        // Initialize method arguments detected from ABI
{{.InitializerVars}}
        {{.InitializerEncode}}
    }
{{else}}    /// @notice Get proxy initializer data
    function _getProxyInitializer() internal pure override returns (bytes memory) {
        // No initialize method detected - proxy will be deployed without initialization
        return "";
    }
{{end}}
}`
}
