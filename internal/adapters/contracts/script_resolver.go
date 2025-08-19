package contracts

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ScriptResolver handles script resolution without pkg dependencies
type ScriptResolver struct {
	resolver    usecase.ContractResolver
	projectRoot string
}

// NewScriptResolver creates a new internal script resolver
func NewScriptResolver(projectRoot string, resolver usecase.ContractResolver) *ScriptResolver {
	return &ScriptResolver{
		resolver:    resolver,
		projectRoot: projectRoot,
	}
}

// ResolveScript resolves a script path or name to script info
func (r *ScriptResolver) ResolveScript(ctx context.Context, pathOrName string) (*domain.ContractInfo, error) {
	script := "^script"
	return r.resolver.ResolveContract(ctx, domain.ContractQuery{
		Query:       &pathOrName,
		PathPattern: &script,
	})
}

// GetScriptParameters extracts parameters from a script's artifact
func (r *ScriptResolver) GetScriptParameters(ctx context.Context, script *domain.ContractInfo) ([]domain.ScriptParameter, error) {
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
