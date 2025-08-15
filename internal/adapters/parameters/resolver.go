package parameters

import (
	"context"
	"fmt"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ParameterResolverAdapter adapts the parameter resolver for the usecase layer
type ParameterResolverAdapter struct {
	resolver *InternalResolver
}

// NewParameterResolverAdapter creates a new parameter resolver adapter
func NewParameterResolverAdapter(
	cfg *config.RuntimeConfig,
	deploymentStore usecase.DeploymentStore,
	contractIndexer usecase.ContractIndexer,
) *ParameterResolverAdapter {
	resolver := NewInternalResolver(cfg, deploymentStore, contractIndexer)
	return &ParameterResolverAdapter{
		resolver: resolver,
	}
}

// ResolveParameters resolves parameter values from various sources
func (a *ParameterResolverAdapter) ResolveParameters(
	ctx context.Context,
	params []domain.ScriptParameter,
	values map[string]string,
) (map[string]string, error) {
	return a.resolver.ResolveParameters(ctx, params, values)
}

// ValidateParameters validates that all required parameters have values
func (a *ParameterResolverAdapter) ValidateParameters(
	ctx context.Context,
	params []domain.ScriptParameter,
	values map[string]string,
) error {
	return a.resolver.ValidateParameters(ctx, params, values)
}

// ParameterPrompterAdapter adapts the parameter prompter for the usecase layer
type ParameterPrompterAdapter struct {
	cfg             *config.RuntimeConfig
	deploymentStore usecase.DeploymentStore
	contractIndexer usecase.ContractIndexer
}

// NewParameterPrompterAdapter creates a new parameter prompter adapter
func NewParameterPrompterAdapter(
	cfg *config.RuntimeConfig,
	deploymentStore usecase.DeploymentStore,
	contractIndexer usecase.ContractIndexer,
) *ParameterPrompterAdapter {
	return &ParameterPrompterAdapter{
		cfg:             cfg,
		deploymentStore: deploymentStore,
		contractIndexer: contractIndexer,
	}
}

// PromptForParameters prompts the user for missing parameter values
func (p *ParameterPrompterAdapter) PromptForParameters(
	ctx context.Context,
	params []domain.ScriptParameter,
	existing map[string]string,
) (map[string]string, error) {
	// For now, we'll create a simple prompter that returns an error for missing params
	// In a full implementation, this would use a UI library to prompt the user
	var missing []string
	for _, param := range params {
		if !param.Optional && existing[param.Name] == "" {
			missing = append(missing, param.Name)
		}
	}
	
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required parameters (interactive mode not yet implemented): %s", strings.Join(missing, ", "))
	}
	
	return existing, nil
}