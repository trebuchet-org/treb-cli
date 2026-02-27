package resolvers

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ContractResolver handles contract resolution and selection
type ContractResolver struct {
	config   *config.RuntimeConfig
	repo     usecase.ContractRepository
	selector usecase.ContractSelector
}

// NewContractResolver creates a new contract resolver
func NewContractResolver(
	cfg *config.RuntimeConfig,
	repo usecase.ContractRepository,
	selector usecase.ContractSelector,
) *ContractResolver {
	return &ContractResolver{
		config:   cfg,
		repo:     repo,
		selector: selector,
	}
}

// ResolveContractWithFilter resolves a contract reference with filtering
func (r *ContractResolver) ResolveContract(ctx context.Context, query domain.ContractQuery) (*models.Contract, error) {
	// First try exact match (could be "Counter" or "src/Counter.sol:Counter")
	contracts, err := r.repo.SearchContracts(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search contracts: %w", err)
	}

	if len(contracts) == 0 {
		// Provide helpful error message based on what was filtered out
		return nil, domain.NoContractsMatchErr{Query: query}
	}

	if len(contracts) == 1 {
		return contracts[0], nil
	}

	// Multiple matches - use interactive selector if available
	if r.selector != nil && !r.config.NonInteractive {
		selected, err := r.selector.SelectContract(ctx, contracts, fmt.Sprintf("Multiple contracts found for '%v'. Select one:", query))
		if err != nil {
			return nil, fmt.Errorf("contract selection failed: %w", err)
		}
		return selected, nil
	}

	// Non-interactive mode with multiple matches
	return nil, domain.AmbiguousFilterErr{Query: query, Matches: contracts}
}

// GetProxyContracts returns all available proxy contracts
func (r *ContractResolver) GetProxyContracts(ctx context.Context) ([]*models.Contract, error) {
	var proxy = "Proxy"
	proxies, err := r.repo.SearchContracts(ctx, domain.ContractQuery{Query: &proxy})
	if err != nil {
		return nil, fmt.Errorf("failed to search proxy contracts: %w", err)
	}
	if len(proxies) == 0 {
		return nil, fmt.Errorf("no proxy contracts found in project")
	}
	return proxies, nil
}

// SelectProxyContract interactively selects a proxy contract
func (r *ContractResolver) SelectProxyContract(ctx context.Context) (*models.Contract, error) {
	// Get all proxy contracts
	proxies, err := r.GetProxyContracts(ctx)
	if err != nil {
		return nil, err
	}

	// If only one proxy, return it
	if len(proxies) == 1 {
		return proxies[0], nil
	}

	// Use interactive selector if available
	if r.selector != nil && !r.config.NonInteractive {
		selected, err := r.selector.SelectContract(ctx, proxies, "Select a proxy contract:")
		if err != nil {
			return nil, fmt.Errorf("proxy selection failed: %w", err)
		}
		return selected, nil
	}

	// Non-interactive mode with multiple proxies
	return nil, fmt.Errorf("multiple proxy contracts found. Please specify --proxy-contract or run in interactive mode")
}

// A bit of a heuristic, could do better using solidity ASTs, but should work for 99% of cases.
func (r *ContractResolver) IsLibrary(ctx context.Context, contract *models.Contract) (bool, error) {
	content, err := os.ReadFile(contract.Path)
	if err != nil {
		return false, err
	}

	return strings.Contains(string(content), fmt.Sprintf("library %s", contract.Name)), nil
}

// Ensure the adapter implements both interfaces
var _ usecase.ContractResolver = (*ContractResolver)(nil)
