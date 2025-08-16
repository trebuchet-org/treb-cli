package contracts

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ScriptResolver handles script resolution without pkg dependencies
type ScriptResolver struct {
	indexer     *Indexer
	projectRoot string
}

// NewScriptResolver creates a new internal script resolver
func NewScriptResolver(projectRoot string, indexer *Indexer) *ScriptResolver {
	return &ScriptResolver{
		indexer:     indexer,
		projectRoot: projectRoot,
	}
}

// ResolveScript resolves a script path or name to script info
func (r *ScriptResolver) ResolveScript(ctx context.Context, pathOrName string) (*domain.ScriptInfo, error) {
	// Don't automatically rebuild the index here - let the caller control when to rebuild
	// This prevents unnecessary builds and allows the run command to build once

	// Debug output
	if os.Getenv("TREB_TEST_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "DEBUG: Resolving script: %s\n", pathOrName)
		allContracts := r.indexer.GetAllContracts()
		fmt.Fprintf(os.Stderr, "DEBUG: Total contracts in index: %d\n", len(allContracts))
		scripts := r.indexer.GetScriptContracts()
		fmt.Fprintf(os.Stderr, "DEBUG: Script contracts found: %d\n", len(scripts))
		for _, s := range scripts {
			fmt.Fprintf(os.Stderr, "  - %s (path: %s)\n", s.Name, s.Path)
		}
	}

	// Try direct lookup first
	contract, err := r.indexer.GetContract(pathOrName)
	if err == nil && (strings.HasPrefix(contract.Path, "script/") || strings.Contains(contract.Path, "/script/")) {
		return r.contractToScriptInfo(contract)
	}

	// If that fails, search for scripts
	scripts := r.indexer.GetScriptContracts()

	// Look for exact name match
	for _, script := range scripts {
		if script.Name == pathOrName {
			return r.contractToScriptInfo(script)
		}
	}

	// Look for path match
	for _, script := range scripts {
		if script.Path == pathOrName {
			return r.contractToScriptInfo(script)
		}
	}

	// If a .sol file path is provided, try to match by path
	if strings.HasSuffix(pathOrName, ".sol") {
		for _, script := range scripts {
			if script.Path == pathOrName {
				return r.contractToScriptInfo(script)
			}
		}
	}

	// Partial match on name
	var matches []*domain.ContractInfo
	lowName := strings.ToLower(pathOrName)
	for _, script := range scripts {
		if strings.Contains(strings.ToLower(script.Name), lowName) {
			matches = append(matches, script)
		}
	}

	if len(matches) == 1 {
		return r.contractToScriptInfo(matches[0])
	} else if len(matches) > 1 {
		// Ambiguous match
		var names []string
		for _, m := range matches {
			names = append(names, fmt.Sprintf("%s (%s)", m.Name, m.Path))
		}
		return nil, fmt.Errorf("multiple scripts match '%s': %s", pathOrName, strings.Join(names, ", "))
	}

	return nil, fmt.Errorf("no script found matching '%s'", pathOrName)
}

// GetScriptParameters extracts parameters from a script's artifact
func (r *ScriptResolver) GetScriptParameters(ctx context.Context, script *domain.ScriptInfo) ([]domain.ScriptParameter, error) {
	// The script already has the artifact path from when it was resolved
	if script.ArtifactPath == "" {
		return nil, fmt.Errorf("script artifact path not set")
	}

	// Load artifact
	artifact, err := r.loadArtifact(script.ArtifactPath)
	if err != nil {
		return nil, err
	}

	// Parse parameters from artifact metadata
	return r.parseParametersFromArtifact(artifact)
}

// contractToScriptInfo converts a contract info to script info
func (r *ScriptResolver) contractToScriptInfo(contract *domain.ContractInfo) (*domain.ScriptInfo, error) {
	return &domain.ScriptInfo{
		Path:         contract.Path,
		Name:         contract.Name,
		ContractName: contract.Name,
		ArtifactPath: contract.ArtifactPath,
	}, nil
}

// loadArtifact loads an artifact from disk
func (r *ScriptResolver) loadArtifact(artifactPath string) (*domain.Artifact, error) {
	// Handle relative paths
	fullPath := artifactPath
	if !filepath.IsAbs(artifactPath) {
		fullPath = filepath.Join(r.projectRoot, artifactPath)
	}
	
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read artifact at %s: %w", fullPath, err)
	}

	var artifact domain.Artifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		return nil, fmt.Errorf("failed to parse artifact: %w", err)
	}

	return &artifact, nil
}

// parseParametersFromArtifact extracts parameters from artifact metadata
func (r *ScriptResolver) parseParametersFromArtifact(artifact *domain.Artifact) ([]domain.ScriptParameter, error) {
	// DevDoc is already parsed in the artifact metadata
	if len(artifact.Metadata.Output.DevDoc) == 0 {
		return nil, nil
	}

	// Extract devdoc
	var devdoc struct {
		Methods map[string]map[string]interface{} `json:"methods"`
	}

	if err := json.Unmarshal(artifact.Metadata.Output.DevDoc, &devdoc); err != nil {
		return nil, nil // No devdoc
	}

	// Look for run() method
	runMethod, exists := devdoc.Methods["run()"]
	if !exists {
		return nil, nil // No run() method
	}

	// Look for custom:env tag
	customEnv, exists := runMethod["custom:env"]
	if !exists {
		return nil, nil // No parameters
	}

	envStr, ok := customEnv.(string)
	if !ok {
		return nil, fmt.Errorf("custom:env is not a string")
	}

	// Parse parameters from env string
	return r.parseCustomEnvString(envStr)
}

// parseCustomEnvString parses the custom:env string into parameters
func (r *ScriptResolver) parseCustomEnvString(envStr string) ([]domain.ScriptParameter, error) {
	// Format: {type} name description{type2} name2 description2...
	var params []domain.ScriptParameter

	// Simple parser - in production would use regex
	parts := strings.Split(envStr, "{")
	for _, part := range parts[1:] { // Skip first empty part
		// Find closing brace
		endType := strings.Index(part, "}")
		if endType == -1 {
			continue
		}

		typeStr := part[:endType]
		rest := strings.TrimSpace(part[endType+1:])

		// Check if optional
		optional := false
		if strings.HasSuffix(typeStr, ":optional") {
			optional = true
			typeStr = strings.TrimSuffix(typeStr, ":optional")
		}

		// Split rest into name and description
		fields := strings.Fields(rest)
		if len(fields) < 1 {
			continue
		}

		name := fields[0]
		description := ""
		if len(fields) > 1 {
			description = strings.Join(fields[1:], " ")
			// Trim until next parameter
			if nextParam := strings.Index(description, "{"); nextParam != -1 {
				description = strings.TrimSpace(description[:nextParam])
			}
		}

		// Map type
		paramType := mapStringToParamType(typeStr)

		params = append(params, domain.ScriptParameter{
			Name:        name,
			Type:        paramType,
			Description: description,
			Optional:    optional,
		})
	}

	return params, nil
}

// mapStringToParamType maps string type to domain parameter type
func mapStringToParamType(typeStr string) domain.ParameterType {
	switch typeStr {
	case "string":
		return domain.ParamTypeString
	case "address":
		return domain.ParamTypeAddress
	case "uint256":
		return domain.ParamTypeUint256
	case "int256":
		return domain.ParamTypeInt256
	case "bytes32":
		return domain.ParamTypeBytes32
	case "bytes":
		return domain.ParamTypeBytes
	case "bool":
		return domain.ParamTypeBool
	case "sender":
		return domain.ParamTypeSender
	case "deployment":
		return domain.ParamTypeDeployment
	case "artifact":
		return domain.ParamTypeArtifact
	default:
		return domain.ParamTypeString
	}
}

// ScriptResolverAdapter adapts the script resolver for the usecase layer
type ScriptResolverAdapter struct {
	resolver        *ScriptResolver
	contractIndexer usecase.ContractIndexer
}

// NewScriptResolverAdapter creates a new script resolver adapter
func NewScriptResolverAdapter(cfg *config.RuntimeConfig, contractIndexer usecase.ContractIndexer) (*ScriptResolverAdapter, error) {
	// Get the indexer from the contract resolver adapter
	contractResolverAdapter, ok := contractIndexer.(*ContractResolverAdapter)
	if !ok {
		return nil, fmt.Errorf("expected ContractResolverAdapter, got %T", contractIndexer)
	}
	
	resolver := NewScriptResolver(cfg.ProjectRoot, contractResolverAdapter.GetIndexer())

	return &ScriptResolverAdapter{
		resolver: resolver,
		contractIndexer: contractIndexer,
	}, nil
}

// ResolveScript resolves a script path or name to script info
func (a *ScriptResolverAdapter) ResolveScript(ctx context.Context, pathOrName string) (*domain.ScriptInfo, error) {
	if os.Getenv("TREB_TEST_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "DEBUG: ScriptResolverAdapter.ResolveScript called for: %s\n", pathOrName)
	}
	// Refresh the index to pick up any newly generated scripts
	if err := a.contractIndexer.RefreshIndex(ctx); err != nil {
		return nil, fmt.Errorf("failed to refresh index: %w", err)
	}
	return a.resolver.ResolveScript(ctx, pathOrName)
}

// GetScriptParameters extracts parameters from a script's artifact
func (a *ScriptResolverAdapter) GetScriptParameters(ctx context.Context, script *domain.ScriptInfo) ([]domain.ScriptParameter, error) {
	return a.resolver.GetScriptParameters(ctx, script)
}