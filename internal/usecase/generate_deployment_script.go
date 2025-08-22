package usecase

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
)

// GenerateScriptParams contains parameters for generating a deployment script
type GenerateScriptParams struct {
	ArtifactRef   string
	UseProxy      bool
	ProxyContract string // optional, interactive if empty with UseProxy
	Strategy      domain.ScriptDeploymentStrategy
	CustomPath    string // optional custom script path
}

// GenerateScriptResult contains the result of script generation
type GenerateScriptResult struct {
	ScriptPath   string
	ScriptType   domain.ScriptType
	Instructions []string // deployment instructions for user
}

// GenerateDeploymentScript is the use case for generating deployment scripts
type GenerateDeploymentScript struct {
	config           *config.RuntimeConfig
	contractResolver ContractResolver
	abiParser        ABIParser
	abiResolver      ABIResolver
	scriptGenerator  ScriptGenerator
	fileWriter       FileWriter
}

// NewGenerateDeploymentScript creates a new GenerateDeploymentScript use case
func NewGenerateDeploymentScript(
	cfg *config.RuntimeConfig,
	contractResolver ContractResolver,
	abiParser ABIParser,
	abiResolver ABIResolver,
	scriptGenerator ScriptGenerator,
	fileWriter FileWriter,
) *GenerateDeploymentScript {
	return &GenerateDeploymentScript{
		config:           cfg,
		contractResolver: contractResolver,
		abiParser:        abiParser,
		abiResolver:      abiResolver,
		scriptGenerator:  scriptGenerator,
		fileWriter:       fileWriter,
	}
}

// Run executes the generate deployment script use case
func (uc *GenerateDeploymentScript) Run(ctx context.Context, params GenerateScriptParams) (*GenerateScriptResult, error) {
	// Resolve the main artifact
	contract, err := uc.contractResolver.ResolveContract(ctx, domain.ContractQuery{Query: &params.ArtifactRef})
	if err != nil {
		return nil, err
	}

	// Determine script type
	scriptType := domain.ScriptTypeContract
	if ok, err := uc.contractResolver.IsLibrary(ctx, contract); err != nil {
		return nil, err
	} else if ok {
		scriptType = domain.ScriptTypeLibrary
		// Validate proxy usage
		if params.UseProxy {
			return nil, fmt.Errorf("libraries cannot be deployed with proxies")
		}
	} else if params.UseProxy {
		scriptType = domain.ScriptTypeProxy
	}

	abi, err := uc.abiResolver.Get(ctx, contract.Artifact)

	// Build artifact path if not already specified
	artifactPath := fmt.Sprintf("%s:%s", contract.Path, contract.Name)

	// Determine script path
	scriptPath := uc.determineScriptPath(contract.Name, scriptType, params.CustomPath)

	// Ensure directory exists
	if err := uc.fileWriter.EnsureDirectory(ctx, filepath.Dir(scriptPath)); err != nil {
		return nil, fmt.Errorf("failed to create script directory: %w", err)
	}

	// Check if script exists
	exists, err := uc.fileWriter.FileExists(ctx, scriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check script existence: %w", err)
	}
	if exists && params.CustomPath == "" {
		return nil, fmt.Errorf("script already exists: %s\nUse --script-path flag to specify a different location", scriptPath)
	}

	// Build script template
	template := &domain.ScriptTemplate{
		Type:         scriptType,
		ContractName: contract.Name,
		ArtifactPath: artifactPath,
		Strategy:     params.Strategy,
		ScriptPath:   scriptPath,
		ABI:          abi,
	}

	// Handle proxy-specific logic
	if params.UseProxy {
		proxyInfo, err := uc.resolveProxyInfo(ctx, params.ProxyContract)
		if err != nil {
			return nil, err
		}
		template.ProxyInfo = proxyInfo
	}

	scriptContent, err := uc.scriptGenerator.GenerateScript(ctx, template)
	if err != nil {
		return nil, fmt.Errorf("failed to generate script: %w", err)
	}

	// Write script
	if err := uc.fileWriter.WriteScript(ctx, scriptPath, scriptContent); err != nil {
		return nil, fmt.Errorf("failed to write script: %w", err)
	}

	// Build result
	instructions := uc.buildInstructions(scriptType, scriptPath, uc.config.Network)

	return &GenerateScriptResult{
		ScriptPath:   scriptPath,
		ScriptType:   scriptType,
		Instructions: instructions,
	}, nil
}

// resolveProxyInfo resolves proxy deployment information
func (uc *GenerateDeploymentScript) resolveProxyInfo(ctx context.Context, proxyContract string) (*domain.ScriptProxyInfo, error) {
	var proxy *models.Contract
	var err error

	if proxyContract != "" {
		// Specific proxy provided
		proxy, err = uc.contractResolver.ResolveContract(ctx, domain.ContractQuery{Query: &proxyContract})
		if err != nil {
			return nil, fmt.Errorf("failed to resolve proxy contract: %w", err)
		}
	} else {
		// Use the contract resolver's interactive selection
		proxy, err = uc.contractResolver.SelectProxyContract(ctx)
		if err != nil {
			return nil, err
		}
	}

	// Build proxy artifact path
	proxyArtifact := fmt.Sprintf("%s:%s", proxy.Path, proxy.Name)

	result := &domain.ScriptProxyInfo{
		ProxyName:     proxy.Name,
		ProxyPath:     proxy.Path,
		ProxyArtifact: proxyArtifact,
	}

	return result, nil
}

// determineScriptPath determines the path for the generated script
func (uc *GenerateDeploymentScript) determineScriptPath(contractName string, scriptType domain.ScriptType, customPath string) string {
	if customPath != "" {
		return customPath
	}

	var filename string
	switch scriptType {
	case domain.ScriptTypeProxy:
		filename = fmt.Sprintf("Deploy%sProxy.s.sol", contractName)
	default:
		filename = fmt.Sprintf("Deploy%s.s.sol", contractName)
	}

	return filepath.Join("script", "deploy", filename)
}

// buildInstructions builds deployment instructions for the user
func (uc *GenerateDeploymentScript) buildInstructions(scriptType domain.ScriptType, scriptPath string, network *config.Network) []string {
	var instructions []string

	switch scriptType {
	case domain.ScriptTypeLibrary:
		instructions = append(instructions, "This library will be deployed with CREATE2 for deterministic addresses.")
	case domain.ScriptTypeProxy:
		instructions = append(instructions, "This script will deploy both the implementation and proxy contracts.")
		instructions = append(instructions, "Make sure to update the initializer parameters if needed.")
	}

	instructions = append(instructions, "", "To deploy, run:")
	if network != nil {
		instructions = append(instructions, fmt.Sprintf("  treb run %s --network %s", scriptPath, network.Name))
	} else {
		instructions = append(instructions, fmt.Sprintf("  treb run %s --network <network>", scriptPath))
	}

	return instructions
}
