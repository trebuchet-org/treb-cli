package usecase

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
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
	scriptGenerator  ScriptGenerator
	fileWriter       FileWriter
	sink             ProgressSink
}

// NewGenerateDeploymentScript creates a new GenerateDeploymentScript use case
func NewGenerateDeploymentScript(
	cfg *config.RuntimeConfig,
	contractResolver ContractResolver,
	abiParser ABIParser,
	scriptGenerator ScriptGenerator,
	fileWriter FileWriter,
	sink ProgressSink,
) *GenerateDeploymentScript {
	return &GenerateDeploymentScript{
		config:           cfg,
		contractResolver: contractResolver,
		abiParser:        abiParser,
		scriptGenerator:  scriptGenerator,
		fileWriter:       fileWriter,
		sink:             sink,
	}
}

// Run executes the generate deployment script use case
func (uc *GenerateDeploymentScript) Run(ctx context.Context, params GenerateScriptParams) (*GenerateScriptResult, error) {
	// Resolve the main artifact
	contractInfo, err := uc.contractResolver.ResolveContract(ctx, domain.ContractQuery{Query: &params.ArtifactRef})
	if err != nil {
		return nil, err
	}

	// Determine script type
	scriptType := domain.ScriptTypeContract
	if ok, err := uc.contractResolver.IsLibrary(ctx, contractInfo); err != nil {
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

	// Parse ABI
	uc.sink.OnProgress(ctx, ProgressEvent{
		Stage:   "parsing",
		Message: "Parsing contract ABI",
		Spinner: true,
	})

	contractABI, err := uc.abiParser.ParseContractABI(ctx, contractInfo.Name)
	if err != nil {
		// Non-fatal: assume no constructor
		contractABI = &domain.ContractABI{
			Name:           contractInfo.Name,
			HasConstructor: false,
		}
	}

	// Build artifact path if not already specified
	artifactPath := fmt.Sprintf("%s:%s", contractInfo.Path, contractInfo.Name)

	// Determine script path
	scriptPath := uc.determineScriptPath(contractInfo.Name, scriptType, params.CustomPath)

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
		ContractName: contractInfo.Name,
		ArtifactPath: artifactPath,
		Strategy:     params.Strategy,
		ScriptPath:   scriptPath,
	}

	// Add constructor info
	if contractABI.HasConstructor && contractABI.Constructor != nil {
		template.ConstructorInfo = &domain.ConstructorInfo{
			HasConstructor: true,
			Parameters:     contractABI.Constructor.Inputs,
		}
	}

	// Handle proxy-specific logic
	if params.UseProxy {
		proxyInfo, err := uc.resolveProxyInfo(ctx, params.ProxyContract, contractABI)
		if err != nil {
			return nil, err
		}
		template.ProxyInfo = proxyInfo
	}

	// Generate script
	uc.sink.OnProgress(ctx, ProgressEvent{
		Stage:   "generating",
		Message: "Generating deployment script",
		Spinner: true,
	})

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

	uc.sink.OnProgress(ctx, ProgressEvent{
		Stage:   "complete",
		Message: "Script generated successfully",
	})

	return &GenerateScriptResult{
		ScriptPath:   scriptPath,
		ScriptType:   scriptType,
		Instructions: instructions,
	}, nil
}

// resolveProxyInfo resolves proxy deployment information
func (uc *GenerateDeploymentScript) resolveProxyInfo(ctx context.Context, proxyContract string, implABI *domain.ContractABI) (*domain.ScriptProxyInfo, error) {
	var proxyInfo *domain.ContractInfo
	var err error

	if proxyContract != "" {
		// Specific proxy provided
		proxyInfo, err = uc.contractResolver.ResolveContract(ctx, domain.ContractQuery{Query: &proxyContract})
		if err != nil {
			return nil, fmt.Errorf("failed to resolve proxy contract: %w", err)
		}
	} else {
		// Use the contract resolver's interactive selection
		proxyInfo, err = uc.contractResolver.SelectProxyContract(ctx)
		if err != nil {
			return nil, err
		}
	}

	// Build proxy artifact path
	proxyArtifact := fmt.Sprintf("%s:%s", proxyInfo.Path, proxyInfo.Name)

	result := &domain.ScriptProxyInfo{
		ProxyName:     proxyInfo.Name,
		ProxyPath:     proxyInfo.Path,
		ProxyArtifact: proxyArtifact,
	}

	// Check for initializer method
	if implABI != nil {
		initMethod := uc.abiParser.FindInitializeMethod(implABI)
		if initMethod != nil {
			result.InitializerInfo = &domain.InitializerInfo{
				MethodName: initMethod.Name,
				Parameters: initMethod.Inputs,
			}
		}
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
func (uc *GenerateDeploymentScript) buildInstructions(scriptType domain.ScriptType, scriptPath string, network *domain.Network) []string {
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
