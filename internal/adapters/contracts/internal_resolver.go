package contracts

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// InternalScriptResolver handles script resolution without pkg dependencies
type InternalScriptResolver struct {
	indexer     *InternalIndexer
	projectRoot string
}

// NewInternalScriptResolver creates a new internal script resolver
func NewInternalScriptResolver(projectRoot string) (*InternalScriptResolver, error) {
	indexer := NewInternalIndexer(projectRoot)

	// Index contracts
	if err := indexer.Index(); err != nil {
		return nil, fmt.Errorf("failed to index contracts: %w", err)
	}

	return &InternalScriptResolver{
		indexer:     indexer,
		projectRoot: projectRoot,
	}, nil
}

// ResolveScript resolves a script path or name to script info
func (r *InternalScriptResolver) ResolveScript(ctx context.Context, pathOrName string) (*domain.ScriptInfo, error) {
	// Try direct lookup first
	contract, err := r.indexer.GetContract(pathOrName)
	if err == nil && (strings.HasPrefix(contract.Path, "script/") || strings.Contains(contract.Path, "/script/")) {
		return r.contractToScriptInfo(contract)
	}

	// Check if it's a direct file path that exists
	if strings.HasSuffix(pathOrName, ".sol") {
		fullPath := path.Join(r.projectRoot, pathOrName)
		if _, err := os.Stat(fullPath); err == nil {
			// Extract contract name from file path
			baseName := path.Base(pathOrName)
			contractName := strings.TrimSuffix(baseName, ".sol")

			// Try to find it in indexed contracts first
			contract, err := r.indexer.GetContract(pathOrName + ":" + contractName)
			if err == nil {
				return r.contractToScriptInfo(contract)
			}

			// If not indexed, create a basic script info
			return &domain.ScriptInfo{
				Path:         pathOrName,
				Name:         contractName,
				ContractName: contractName,
			}, nil
		}
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
		if script.Path == pathOrName || strings.HasSuffix(script.Path, "/"+pathOrName) {
			return r.contractToScriptInfo(script)
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
func (r *InternalScriptResolver) GetScriptParameters(ctx context.Context, script *domain.ScriptInfo) ([]domain.ScriptParameter, error) {
	// Find the contract from indexer to get artifact path
	contract, err := r.indexer.GetContract(script.ContractName)
	if err != nil {
		return nil, fmt.Errorf("failed to find contract: %w", err)
	}

	// Load artifact
	artifact, err := r.loadArtifact(contract.ArtifactPath)
	if err != nil {
		return nil, err
	}

	// Parse parameters from artifact metadata
	return r.parseParametersFromArtifact(artifact)
}

// contractToScriptInfo converts a contract info to script info
func (r *InternalScriptResolver) contractToScriptInfo(contract *domain.ContractInfo) (*domain.ScriptInfo, error) {
	// For now, we don't load the artifact until needed
	return &domain.ScriptInfo{
		Path:         contract.Path,
		Name:         contract.Name,
		ContractName: contract.Name,
	}, nil
}

// loadArtifact loads an artifact from disk
func (r *InternalScriptResolver) loadArtifact(artifactPath string) (*domain.Artifact, error) {
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read artifact: %w", err)
	}

	var artifact domain.Artifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		return nil, fmt.Errorf("failed to parse artifact: %w", err)
	}

	return &artifact, nil
}

// parseParametersFromArtifact extracts parameters from artifact metadata
func (r *InternalScriptResolver) parseParametersFromArtifact(artifact *domain.Artifact) ([]domain.ScriptParameter, error) {
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
func (r *InternalScriptResolver) parseCustomEnvString(envStr string) ([]domain.ScriptParameter, error) {
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

