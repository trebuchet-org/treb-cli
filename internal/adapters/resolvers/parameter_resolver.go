package resolvers

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ParameterResolver handles parameter resolution without pkg dependencies
type ParameterResolver struct {
	cfg             *config.RuntimeConfig
	deploymentRepo  usecase.DeploymentRepository
	contractIndexer usecase.ContractRepository
}

// NewParameterResolver creates a new internal parameter resolver
func NewParameterResolver(
	cfg *config.RuntimeConfig,
	deploymentRepo usecase.DeploymentRepository,
	contractIndexer usecase.ContractRepository,
) *ParameterResolver {
	return &ParameterResolver{
		cfg:             cfg,
		deploymentRepo:  deploymentRepo,
		contractIndexer: contractIndexer,
	}
}

// ResolveParameters resolves parameter values from various sources
func (r *ParameterResolver) ResolveParameters(
	ctx context.Context,
	params []domain.ScriptParameter,
	values map[string]string,
) (map[string]string, error) {
	resolved := make(map[string]string)

	// Copy existing values
	for k, v := range values {
		resolved[k] = v
	}

	// Resolve each parameter
	for _, param := range params {
		// Skip if already has a value
		if resolved[param.Name] != "" {
			continue
		}

		// Try to resolve based on type
		value, err := r.resolveParameter(ctx, param, resolved)
		if err != nil {
			if !param.Optional {
				return nil, fmt.Errorf("failed to resolve parameter %s: %w", param.Name, err)
			}
			// Optional parameter, skip
			continue
		}

		if value != "" {
			resolved[param.Name] = value
		}
	}

	return resolved, nil
}

// ValidateParameters validates that all required parameters have values
func (r *ParameterResolver) ValidateParameters(
	ctx context.Context,
	params []domain.ScriptParameter,
	values map[string]string,
) error {
	var missing []string
	for _, param := range params {
		if !param.Optional && values[param.Name] == "" {
			missing = append(missing, param.Name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required parameters: %s", strings.Join(missing, ", "))
	}

	return nil
}

// resolveParameter attempts to resolve a single parameter
func (r *ParameterResolver) resolveParameter(
	ctx context.Context,
	param domain.ScriptParameter,
	existingValues map[string]string,
) (string, error) {
	// Try environment variable first
	envVar := fmt.Sprintf("TREB_%s", strings.ToUpper(param.Name))
	if value := os.Getenv(envVar); value != "" {
		return value, nil
	}

	// Try based on parameter type
	switch param.Type {
	case domain.ParamTypeSender:
		return r.resolveSender(ctx, param.Name)

	case domain.ParamTypeDeployment:
		return r.resolveDeployment(ctx, param.Name, existingValues)

	case domain.ParamTypeArtifact:
		return r.resolveArtifact(ctx, param.Name)

	default:
		// For basic types, check environment or config
		return r.resolveFromConfig(param.Name)
	}
}

// resolveSender resolves a sender parameter
func (r *ParameterResolver) resolveSender(ctx context.Context, name string) (string, error) {
	// Check if there's a sender configured with this name
	if r.cfg.TrebConfig != nil {
		if sender, ok := r.cfg.TrebConfig.Senders[name]; ok {
			// Return the account address for the sender
			if sender.Address != "" {
				return sender.Address, nil
			}
			// For private key senders, we might need to derive the address
			// This would require eth crypto utilities
		}

		// Try default sender names
		defaultNames := []string{"default", "deployer", r.cfg.Namespace}
		for _, defaultName := range defaultNames {
			if sender, ok := r.cfg.TrebConfig.Senders[defaultName]; ok && sender.Address != "" {
				return sender.Address, nil
			}
		}
	}

	return "", fmt.Errorf("sender %s not found in configuration", name)
}

// resolveDeployment resolves a deployment parameter
func (r *ParameterResolver) resolveDeployment(ctx context.Context, name string, existingValues map[string]string) (string, error) {
	// Extract contract name from parameter name or existing values
	contractName := name
	if hint, ok := existingValues[name+"_contract"]; ok {
		contractName = hint
	}

	// Look up deployment in registry
	filter := domain.DeploymentFilter{
		Namespace:    r.cfg.Namespace,
		ChainID:      r.cfg.Network.ChainID,
		ContractName: contractName,
	}

	deployments, err := r.deploymentRepo.ListDeployments(ctx, filter)
	if err != nil {
		return "", fmt.Errorf("failed to list deployments: %w", err)
	}

	if len(deployments) == 0 {
		return "", fmt.Errorf("no deployment found for %s", contractName)
	}

	// Return the most recent deployment
	return deployments[0].Address, nil
}

// resolveArtifact resolves an artifact parameter
func (r *ParameterResolver) resolveArtifact(ctx context.Context, name string) (string, error) {
	// Search for contracts matching the name
	contracts := r.contractIndexer.SearchContracts(ctx, domain.ContractQuery{Query: &name})

	// Look for exact match first
	for _, contract := range contracts {
		if contract.Name == name {
			return contract.Path, nil
		}
	}

	// Try case-insensitive match
	for _, contract := range contracts {
		if strings.EqualFold(contract.Name, name) {
			return contract.Path, nil
		}
	}

	return "", fmt.Errorf("artifact %s not found", name)
}

// resolveFromConfig resolves a parameter from configuration
func (r *ParameterResolver) resolveFromConfig(name string) (string, error) {
	// Check environment with common prefixes
	prefixes := []string{"TREB_", "FOUNDRY_", ""}
	for _, prefix := range prefixes {
		envVar := prefix + strings.ToUpper(name)
		if value := os.Getenv(envVar); value != "" {
			return value, nil
		}
	}

	return "", nil
}

var _ usecase.ParameterResolver = (*ParameterResolver)(nil)
