package script

import (
	"fmt"
	"strings"

	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/resolvers"
)

// ParameterResolver resolves meta types to their actual values
type ParameterResolver struct {
	projectPath    string
	context        *resolvers.Context
	trebConfig     *config.TrebConfig
	interactive    bool
	namespace      string
	network        string
	chainID        uint64
	contractIndexer *contracts.Indexer
	registryManager *registry.Manager
}

// NewParameterResolver creates a new parameter resolver
func NewParameterResolver(projectPath string, trebConfig *config.TrebConfig, namespace, network string, chainID uint64, interactive bool) (*ParameterResolver, error) {
	ctx := resolvers.NewContext(projectPath, interactive)
	
	// Initialize contract indexer
	indexer, err := contracts.GetGlobalIndexer(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize contract indexer: %w", err)
	}

	// Initialize registry manager
	manager, err := registry.NewManager(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize registry manager: %w", err)
	}

	return &ParameterResolver{
		projectPath:     projectPath,
		context:         ctx,
		trebConfig:      trebConfig,
		interactive:     interactive,
		namespace:       namespace,
		network:         network,
		chainID:         chainID,
		contractIndexer: indexer,
		registryManager: manager,
	}, nil
}

// ResolveValue resolves a parameter value based on its type
func (r *ParameterResolver) ResolveValue(param Parameter, value string) (string, error) {
	// If empty and optional, return as is
	if value == "" && param.Optional {
		return "", nil
	}

	switch param.Type {
	case TypeSender:
		return r.resolveSender(value)
	case TypeDeployment:
		return r.resolveDeployment(value)
	case TypeArtifact:
		return r.resolveArtifact(value)
	default:
		// For base types, return as is (already validated)
		return value, nil
	}
}

// resolveSender validates that a sender exists in the configuration
func (r *ParameterResolver) resolveSender(senderID string) (string, error) {
	if r.trebConfig == nil {
		return "", fmt.Errorf("no treb configuration found")
	}

	// Check if sender exists
	if _, exists := r.trebConfig.Senders[senderID]; !exists {
		// List available senders
		var available []string
		for id := range r.trebConfig.Senders {
			available = append(available, id)
		}
		return "", fmt.Errorf("sender '%s' not found. Available senders: %s", senderID, strings.Join(available, ", "))
	}

	return senderID, nil
}

// resolveDeployment resolves a deployment reference to an address
func (r *ParameterResolver) resolveDeployment(deploymentRef string) (string, error) {
	// Use the deployment resolver
	deployment, err := r.context.ResolveDeployment(deploymentRef, r.registryManager, r.chainID, r.namespace)
	if err != nil {
		return "", fmt.Errorf("failed to resolve deployment '%s': %w", deploymentRef, err)
	}

	return deployment.Address, nil
}

// resolveArtifact resolves an artifact reference to the format "path/to/file.sol:ContractName"
func (r *ParameterResolver) resolveArtifact(artifactRef string) (string, error) {
	// Try to resolve as a contract name
	contractInfo, err := r.context.ResolveContract(artifactRef, contracts.AllFilter())
	if err != nil {
		return "", fmt.Errorf("failed to resolve artifact '%s': %w", artifactRef, err)
	}

	// Build the artifact path in the format expected by forge
	// Remove "src/" or "script/" prefix if present for the artifact format
	contractPath := contractInfo.Path
	if strings.HasPrefix(contractPath, "src/") || strings.HasPrefix(contractPath, "script/") {
		contractPath = contractPath[strings.Index(contractPath, "/")+1:]
	}

	// Return in the format "path/to/Contract.sol:ContractName"
	return fmt.Sprintf("%s:%s", contractPath, contractInfo.Name), nil
}

// ResolveAll resolves all parameter values
func (r *ParameterResolver) ResolveAll(params []Parameter, envVars map[string]string) (map[string]string, error) {
	resolved := make(map[string]string)
	var validationErrors []string

	for _, param := range params {
		value, exists := envVars[param.Name]
		if !exists {
			value = "" // Will be handled by validation/interactive mode
		}

		// Validate the raw value first
		parser := NewParameterParser()
		if err := parser.ValidateValue(param, value); err != nil {
			if !r.interactive {
				validationErrors = append(validationErrors, fmt.Sprintf("%s: %v", param.Name, err))
			}
			// In interactive mode, we'll prompt for invalid values later
			resolved[param.Name] = "" // Mark as needing resolution
			continue
		}

		// Resolve meta types
		if value != "" {
			resolvedValue, err := r.ResolveValue(param, value)
			if err != nil {
				if !r.interactive {
					validationErrors = append(validationErrors, fmt.Sprintf("%s: %v", param.Name, err))
				}
				resolved[param.Name] = "" // Mark as needing resolution
				continue
			}
			resolved[param.Name] = resolvedValue
		} else if !param.Optional {
			// Mark as needing resolution
			resolved[param.Name] = ""
		}
	}

	// Return validation errors in non-interactive mode
	if len(validationErrors) > 0 && !r.interactive {
		return resolved, fmt.Errorf("parameter validation failed:\n  %s", strings.Join(validationErrors, "\n  "))
	}

	return resolved, nil
}