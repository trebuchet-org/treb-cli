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
	ProxyTypeOZTransparent ProxyType = "TransparentUpgradeable"
	ProxyTypeOZUUPS        ProxyType = "UUPSUpgradeable"
	ProxyTypeCustom        ProxyType = "Custom"
)

// ScriptTemplate contains data for generating deploy scripts
type ScriptTemplate struct {
	ContractName        string
	SolidityFile        string
	ArtifactPath        string // Artifact path for the contract
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
	// Ensure script/deploy directory exists (including subdirectories if needed)
	scriptPath := g.GetDeployScriptPath(contractInfo)
	scriptDir := filepath.Dir(scriptPath)
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		return fmt.Errorf("failed to create script directory: %w", err)
	}

	// Parse target contract version
	targetVersion, err := g.parseContractVersion(contractInfo.Path)
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

	// Calculate relative import path from script location to contract
	importPath := g.calculateImportPath(scriptPath, contractInfo.Path)
	
	// Prepare template data
	// For SolidityFile, we need the path relative to src for artifact lookup
	solidityFilePath := contractInfo.Path
	if !strings.HasSuffix(solidityFilePath, ".sol") {
		solidityFilePath = contractInfo.Path
	}
	
	// Index contracts to get artifact path
	indexer, err := GetGlobalIndexer(g.projectRoot)
	if err != nil {
		return fmt.Errorf("failed to initialize contract indexer: %w", err)
	}
	artifactPath := indexer.ResolveContractKey(contractInfo)
	
	templateData := ScriptTemplate{
		ContractName: contractInfo.Name,
		SolidityFile: solidityFilePath,
		ArtifactPath: artifactPath,
		Strategy:     strategy,
		// Version removed - using tags instead
		ImportPath:          importPath,
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
	if _, err := os.Stat(scriptPath); err == nil {
		return fmt.Errorf("deploy script already exists: %s\nUse a different contract name or remove the existing script", scriptPath)
	}

	// Create output file
	file, err := os.Create(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to create deploy script: %w", err)
	}
	defer file.Close()

	// Execute template
	if err := tmpl.Execute(file, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	fmt.Printf("Generated deploy script: %s\n", scriptPath)
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

import {Deployment, DeployStrategy} from "treb-sol/Deployment.sol";
import { {{.ContractName}} } from "{{.ImportPath}}";

/**
 * @title Deploy{{.ContractName}}
 * @notice Deployment script for {{.ContractName}} contract
 * @dev Generated automatically by treb
 */
contract Deploy{{.ContractName}} is Deployment {
    constructor() Deployment(
        "{{.ArtifactPath}}",
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

import {Deployment, DeployStrategy} from "treb-sol/Deployment.sol";
// Target contract uses Solidity {{.TargetVersion}}, which is incompatible with this deployment script (0.8)
// Import commented out to avoid version conflicts. Using artifact-based deployment instead.
// import "{{.ImportPath}}";

/**
 * @title Deploy{{.ContractName}}
 * @notice Deployment script for {{.ContractName}} contract
 * @dev Generated automatically by treb
 * @dev Target contract version: {{.TargetVersion}} (cross-version deployment)
 */
contract Deploy{{.ContractName}} is Deployment {
    constructor() Deployment(
        "{{.ArtifactPath}}",
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
	// Ensure script/deploy directory exists (including subdirectories if needed)
	scriptPath := g.GetProxyScriptPath(contractInfo)
	scriptDir := filepath.Dir(scriptPath)
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		return fmt.Errorf("failed to create script directory: %w", err)
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
	case ProxyTypeOZTransparent:
		proxyImportPath = "@openzeppelin/contracts/proxy/transparent/TransparentUpgradeableProxy.sol"
	case ProxyTypeOZUUPS:
		proxyImportPath = "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol"
	case ProxyTypeCustom:
		// For custom proxy, we'll use a placeholder that users need to update
		proxyImportPath = "../../src/YourCustomProxy.sol"
	}

	// Prepare template data
	templateData := ProxyScriptTemplate{
		ImplementationName: contractInfo.Name,
		SolidityFile:       contractInfo.Path,
		Strategy:           strategy,
		ProxyType:          proxyType,
		ImportPath:         fmt.Sprintf("../../%s", contractInfo.Path),
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
	if _, err := os.Stat(scriptPath); err == nil {
		return fmt.Errorf("proxy deploy script already exists: %s\nUse a different contract name or remove the existing script", scriptPath)
	}

	// Create output file
	file, err := os.Create(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to create proxy deploy script: %w", err)
	}
	defer file.Close()

	// Execute template
	if err := tmpl.Execute(file, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	fmt.Printf("Generated proxy deploy script: %s\n", scriptPath)
	return nil
}

// getProxyDeployScriptTemplate returns the appropriate proxy template
func (g *Generator) getProxyDeployScriptTemplate(proxyType ProxyType) string {
	switch proxyType {
	case ProxyTypeOZTransparent:
		return g.getTransparentProxyTemplate()
	case ProxyTypeOZUUPS:
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

// ProxyScriptTemplateV2 contains data for generating proxy deploy scripts with new structure
type ProxyScriptTemplateV2 struct {
	ImplementationName     string
	ImplementationPath     string
	ProxyContractName      string
	ProxyArtifactPath      string
	ProxyImportPath        string
	Strategy               DeployStrategy
	ProxyType              ProxyType
	HasInitializer         bool
	InitializerVars        string
	InitializerEncode      string
	UseTypeCreationCode    bool
	HasConstructorOverride bool
	ConstructorOverride    string
}

// GenerateProxyDeployScriptV2 generates a proxy deployment script with the new structure
func (g *Generator) GenerateProxyDeployScriptV2(implementationInfo *ContractInfo, proxyInfo *ContractInfo, strategy DeployStrategy, proxyType ProxyType) error {
	// Resolve artifact paths
	indexer, err := GetGlobalIndexer(g.projectRoot)
	if err != nil {
		return fmt.Errorf("failed to initialize contract indexer: %w", err)
	}

	// Get implementation artifact path
	implArtifactPath := indexer.ResolveContractKey(&ContractInfo{
		Name: implementationInfo.Name,
		Path: implementationInfo.Path,
	})

	// Get proxy artifact path
	proxyArtifactPath := indexer.ResolveContractKey(proxyInfo)

	// Prepare import path for proxy
	proxyImportPath := proxyInfo.Path
	if !strings.HasPrefix(proxyImportPath, "@") && !strings.HasPrefix(proxyImportPath, "lib/") {
		proxyImportPath = filepath.Join("..", "..", proxyImportPath)
	} else if strings.HasPrefix(proxyImportPath, "lib/") {
		// Convert lib path to import format
		proxyImportPath = strings.TrimPrefix(proxyImportPath, "lib/")
	}

	// Parse ABI for initializer method
	abiParser := abi.NewParser(g.projectRoot)
	contractABI, err := abiParser.ParseContractABI(implementationInfo.Name)
	if err != nil {
		fmt.Printf("Warning: Could not parse ABI for %s: %v\n", implementationInfo.Name, err)
	}

	var hasInitializer bool
	var initializerVars, initializerEncode string
	if contractABI != nil {
		if initMethod := abiParser.FindInitializeMethod(contractABI); initMethod != nil {
			hasInitializer = true
			initializerVars, initializerEncode = abiParser.GenerateInitializerArgs(initMethod)
		}
	}

	// Check if we need constructor override (for TransparentUpgradeableProxy)
	hasConstructorOverride := proxyType == ProxyTypeOZTransparent
	var constructorOverride string
	if hasConstructorOverride {
		constructorOverride = `    /// @notice Get constructor arguments - override to include admin parameter
    function _getConstructorArgs() internal view override returns (bytes memory) {
        address admin = executor; // Use executor as the ProxyAdmin owner
        bytes memory initData = _getProxyInitializer();
        return abi.encode(implementationAddress, admin, initData);
    }`
	}

	// Create template data
	data := ProxyScriptTemplateV2{
		ImplementationName:     implementationInfo.Name,
		ImplementationPath:     implArtifactPath,
		ProxyContractName:      proxyInfo.Name,
		ProxyArtifactPath:      proxyArtifactPath,
		ProxyImportPath:        proxyImportPath,
		Strategy:               strategy,
		ProxyType:              proxyType,
		HasInitializer:         hasInitializer,
		InitializerVars:        initializerVars,
		InitializerEncode:      initializerEncode,
		UseTypeCreationCode:    true, // We can use type() for proxy contracts
		HasConstructorOverride: hasConstructorOverride,
		ConstructorOverride:    constructorOverride,
	}

	// Generate script content
	scriptContent, err := g.generateProxyScriptV2(data)
	if err != nil {
		return fmt.Errorf("failed to generate script: %w", err)
	}

	// Write script file
	scriptName := fmt.Sprintf("Deploy%sProxy", implementationInfo.Name)
	scriptPath := filepath.Join(g.projectRoot, "script", "deploy", fmt.Sprintf("%s.s.sol", scriptName))

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0755); err != nil {
		return fmt.Errorf("failed to create script directory: %w", err)
	}

	// Write the script
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		return fmt.Errorf("failed to write script file: %w", err)
	}

	fmt.Printf("\n✅ Generated proxy deploy script: %s\n", scriptPath)
	return nil
}

// generateProxyScriptV2 generates the proxy script content with the new structure
func (g *Generator) generateProxyScriptV2(data ProxyScriptTemplateV2) (string, error) {
	tmplStr := `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {ProxyDeployment, DeployStrategy} from "treb-sol/ProxyDeployment.sol";
import {{{.ProxyContractName}}} from "{{.ProxyImportPath}}";

/**
 * @title Deploy{{.ImplementationName}}Proxy
 * @notice Deployment script for {{.ImplementationName}} with {{.ProxyType}} Proxy
 * @dev Generated automatically by treb
 */
contract Deploy{{.ImplementationName}}Proxy is ProxyDeployment {
    constructor() ProxyDeployment(
        "{{.ProxyArtifactPath}}",
        "{{.ImplementationPath}}",
        DeployStrategy.{{.Strategy}}
    ) {}

    /// @notice Get contract bytecode for the proxy
    function _getContractBytecode() internal pure override returns (bytes memory) {
        return type({{.ProxyContractName}}).creationCode;
    }
{{if .HasConstructorOverride}}
{{.ConstructorOverride}}
{{end}}
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

	tmpl, err := template.New("proxyScript").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
