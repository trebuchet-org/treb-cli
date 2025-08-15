package contracts

import (
	"context"
	"fmt"

	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// ScriptResolverAdapter adapts the script resolver for the usecase layer
type ScriptResolverAdapter struct {
	resolver *InternalScriptResolver
}

// NewScriptResolverAdapter creates a new script resolver adapter
func NewScriptResolverAdapter(cfg *config.RuntimeConfig) (*ScriptResolverAdapter, error) {
	resolver, err := NewInternalScriptResolver(cfg.ProjectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to create script resolver: %w", err)
	}

	return &ScriptResolverAdapter{
		resolver: resolver,
	}, nil
}

// ResolveScript resolves a script path or name to script info
func (a *ScriptResolverAdapter) ResolveScript(ctx context.Context, pathOrName string) (*domain.ScriptInfo, error) {
	return a.resolver.ResolveScript(ctx, pathOrName)
}

// GetScriptParameters extracts parameters from a script's artifact
func (a *ScriptResolverAdapter) GetScriptParameters(ctx context.Context, script *domain.ScriptInfo) ([]domain.ScriptParameter, error) {
	return a.resolver.GetScriptParameters(ctx, script)
}