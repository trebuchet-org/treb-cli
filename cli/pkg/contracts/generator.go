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

type ProxyScriptTemplate struct {
	ProxyName                  string
	ProxyImportPath            string
	ImplementationName         string
	ProxyArtifactPath          string
	ImplementationArtifactPath string
	Strategy                   DeployStrategy
	ProxyType                  ProxyType
	HasInitializer             bool
	InitializerVars            string
	InitializerEncode          string
	UseTypeCreationCode        bool
	HasConstructorOverride     bool
	ConstructorOverride        string
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

import {TrebScript} from "treb-sol/TrebScript.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/internal/sender/Deployer.sol";
{{if .ImportPath}}
import { {{.ContractName}} } from "{{.ImportPath}}";
{{end}}

/**
 * @title Deploy{{.ContractName}}
 * @notice Deployment script for {{.ContractName}} contract
 * @dev Generated automatically by treb
 */
contract Deploy{{.ContractName}} is TrebScript {
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    function run() public broadcast {
        // Get the sender
        Senders.Sender storage sender = sender("default");
        
        // Read label from environment (e.g., --env LABEL=v1)
        string memory label = vm.envOr("LABEL", string(""));
        
        // Deploy {{.ContractName}} using {{.Strategy}}
        address deployed = sender.{{if eq .Strategy "CREATE3"}}create3{{else}}create2{{end}}("{{.ArtifactPath}}")
            .setLabel(label)
            .deploy({{if .HasConstructor}}_getConstructorArgs(){{end}});
        
        // Deployment events are automatically emitted for registry tracking
    }
{{if .HasConstructor}}
    /// @notice Get constructor arguments
    function _getConstructorArgs() internal pure returns (bytes memory) {
        // TODO: Update these constructor arguments
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

import {TrebScript} from "treb-sol/TrebScript.sol";
import {Senders} from "treb-sol/internal/sender/Senders.sol";
import {Deployer} from "treb-sol/internal/sender/Deployer.sol";

/**
 * @title Deploy{{.ContractName}}
 * @notice Deployment script for {{.ContractName}} contract
 * @dev Generated automatically by treb
 * @dev Target contract version: {{.TargetVersion}} (cross-version deployment)
 */
contract Deploy{{.ContractName}} is TrebScript {
    using Deployer for Senders.Sender;
    using Deployer for Deployer.Deployment;

    function run() public broadcast {
        // Get the sender
        Senders.Sender storage sender = sender("default");
        
        // Read label from environment (e.g., --env LABEL=v1)
        string memory label = vm.envOr("LABEL", string(""));
        
        // Deploy {{.ContractName}} using {{.Strategy}}
        // Note: Using artifact path for cross-version compatibility
        address deployed = sender.{{if eq .Strategy "CREATE3"}}create3{{else}}create2{{end}}("{{.ArtifactPath}}")
            .setLabel(label)
            .deploy({{if .HasConstructor}}_getConstructorArgs(){{end}});
        
        // Deployment events are automatically emitted for registry tracking
    }
{{if .HasConstructor}}
    /// @notice Get constructor arguments
    function _getConstructorArgs() internal pure returns (bytes memory) {
        // TODO: Update these constructor arguments
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

// GenerateProxyDeployScriptV2 generates a proxy deployment script with the new structure
func (g *Generator) GenerateProxyDeployScript(implementationInfo *ContractInfo, proxyInfo *ContractInfo, strategy DeployStrategy, proxyType ProxyType) error {
	// Get implementation artifact path
	implArtifactPath := fmt.Sprintf("%s:%s", implementationInfo.Path, implementationInfo.Name)

	// Get proxy artifact path
	proxyArtifactPath := fmt.Sprintf("%s:%s", proxyInfo.Path, proxyInfo.Name)

	// Prepare import path for proxy
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
	hasConstructorOverride := proxyType == ProxyTypeOZTransparent || proxyType == ProxyTypeCustom
	var constructorOverride string
	switch proxyType {
	case ProxyTypeOZTransparent:
		constructorOverride = `    /// @notice Get constructor arguments - override to include admin parameter
    function _getConstructorArgs() internal view override returns (bytes memory) {
        address admin = executor; // Use executor as the ProxyAdmin owner
        bytes memory initData = _getProxyInitializer();
        return abi.encode(implementationAddress, admin, initData);
    }`
	case ProxyTypeCustom:
		constructorOverride = `    /// @notice Get constructor arguments - override for custom proxy
    function _getConstructorArgs() internal view override returns (bytes memory) {
        bytes memory initData = _getProxyInitializer();
        // TODO: configure constructor args based on the proxy implementation
        return abi.encode(implementationAddress, initData);
    }`
	}

	// Create template data
	data := ProxyScriptTemplate{
		ProxyName:                  proxyInfo.Name,
		ProxyImportPath:            proxyInfo.Path,
		ImplementationName:         implementationInfo.Name,
		ImplementationArtifactPath: implArtifactPath,
		ProxyArtifactPath:          proxyArtifactPath,
		Strategy:                   strategy,
		ProxyType:                  proxyType,
		HasInitializer:             hasInitializer,
		InitializerVars:            initializerVars,
		InitializerEncode:          initializerEncode,
		UseTypeCreationCode:        true, // We can use type() for proxy contracts
		HasConstructorOverride:     hasConstructorOverride,
		ConstructorOverride:        constructorOverride,
	}

	// Generate script content
	scriptContent, err := g.generateProxyScript(data)
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
func (g *Generator) generateProxyScript(data ProxyScriptTemplate) (string, error) {
	tmplStr := `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {ProxyDeployment, DeployStrategy} from "treb-sol/ProxyDeployment.sol";

/**
 * @title Deploy{{.ImplementationName}}Proxy
 * @notice Deployment script for {{.ImplementationName}} with {{.ProxyType}} Proxy
 * @dev Generated automatically by treb
 */

import { {{.ProxyName}} } from "{{.ProxyImportPath}}";

contract Deploy{{.ImplementationName}}Proxy is ProxyDeployment {
    constructor() ProxyDeployment(
        "{{.ProxyArtifactPath}}",
        "{{.ImplementationArtifactPath}}",
        DeployStrategy.{{.Strategy}}
    ) {}

{{if .HasConstructorOverride}}
{{.ConstructorOverride}}
{{end}}
{{if .HasInitializer}}    /// @notice Get proxy initializer data
    function _getProxyInitializer() internal view override returns (bytes memory) {
        // Initialize method arguments detected from ABI
{{.InitializerVars}}
        {{.InitializerEncode}}
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
