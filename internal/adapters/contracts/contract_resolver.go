package contracts

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ContractResolver handles contract resolution and selection
type ContractResolver struct {
	config          *config.RuntimeConfig
	indexer         *Indexer
	selector        usecase.InteractiveSelector
}

// NewContractResolver creates a new contract resolver
func NewContractResolver(cfg *config.RuntimeConfig, indexer *Indexer, selector usecase.InteractiveSelector) *ContractResolver {
	return &ContractResolver{
		config:   cfg,
		indexer:  indexer,
		selector: selector,
	}
}

// ResolveContract resolves a contract reference to a contract
func (r *ContractResolver) ResolveContract(ctx context.Context, contractRef string) (*domain.ContractInfo, error) {
	// Use default filter (all contracts)
	filter := usecase.ContractFilter{
		IncludeLibraries: true,
		IncludeInterface: true,
		IncludeAbstract:  true,
	}
	return r.ResolveContractWithFilter(ctx, contractRef, filter)
}

// ResolveContractWithFilter resolves a contract reference with filtering
func (r *ContractResolver) ResolveContractWithFilter(ctx context.Context, contractRef string, filter usecase.ContractFilter) (*domain.ContractInfo, error) {
	// First try exact match (could be "Counter" or "src/Counter.sol:Counter")
	contract, err := r.indexer.GetContract(contractRef)
	if err == nil && contract != nil {
		// Check if it matches the filter
		if err := r.validateContractAgainstFilter(contract, filter, contractRef); err != nil {
			return nil, err
		}
		return contract, nil
	}

	// Search for contracts
	contracts := r.indexer.SearchContracts(contractRef)
	
	// Filter contracts based on criteria
	filtered := r.filterContracts(contracts, filter)
	
	if len(filtered) == 0 {
		// Provide helpful error message based on what was filtered out
		if len(contracts) > 0 {
			return nil, r.buildFilterErrorMessage(contractRef, contracts, filter)
		}
		return nil, fmt.Errorf("contract '%s' not found", contractRef)
	}

	if len(filtered) == 1 {
		return filtered[0], nil
	}

	// Multiple matches - use interactive selector if available
	if r.selector != nil && !r.config.NonInteractive {
		selected, err := r.selector.SelectContract(ctx, filtered, fmt.Sprintf("Multiple contracts found for '%s'. Select one:", contractRef))
		if err != nil {
			return nil, fmt.Errorf("contract selection failed: %w", err)
		}
		return selected, nil
	}

	// Non-interactive mode with multiple matches
	return nil, r.buildAmbiguousErrorMessage(contractRef, filtered)
}

// GetProxyContracts returns all available proxy contracts
func (r *ContractResolver) GetProxyContracts(ctx context.Context) ([]*domain.ContractInfo, error) {
	// Common proxy contract patterns
	proxyPatterns := []string{
		"Proxy",
		"UpgradeableProxy",
		"TransparentUpgradeableProxy",
		"UUPSProxy",
		"ERC1967Proxy",
		"BeaconProxy",
	}

	var proxies []*domain.ContractInfo
	seen := make(map[string]bool)

	for _, pattern := range proxyPatterns {
		contracts := r.indexer.SearchContracts(pattern)
		for _, contract := range contracts {
			// Filter to only include actual proxy contracts
			if r.isProxyContract(contract) {
				key := fmt.Sprintf("%s:%s", contract.Path, contract.Name)
				if !seen[key] {
					seen[key] = true
					proxies = append(proxies, contract)
				}
			}
		}
	}

	if len(proxies) == 0 {
		return nil, fmt.Errorf("no proxy contracts found in project")
	}

	return proxies, nil
}

// SelectProxyContract interactively selects a proxy contract
func (r *ContractResolver) SelectProxyContract(ctx context.Context) (*domain.ContractInfo, error) {
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

// filterContracts filters contracts based on the provided filter
func (r *ContractResolver) filterContracts(contracts []*domain.ContractInfo, filter usecase.ContractFilter) []*domain.ContractInfo {
	var filtered []*domain.ContractInfo
	for _, contract := range contracts {
		if r.matchesFilter(contract, filter) {
			filtered = append(filtered, contract)
		}
	}
	return filtered
}

// matchesFilter checks if a contract matches the filter criteria
func (r *ContractResolver) matchesFilter(contract *domain.ContractInfo, filter usecase.ContractFilter) bool {
	if contract.IsLibrary && !filter.IncludeLibraries {
		return false
	}
	if contract.IsInterface && !filter.IncludeInterface {
		return false
	}
	if contract.IsAbstract && !filter.IncludeAbstract {
		return false
	}
	return true
}

// validateContractAgainstFilter validates a single contract against the filter
func (r *ContractResolver) validateContractAgainstFilter(contract *domain.ContractInfo, filter usecase.ContractFilter, ref string) error {
	if contract.IsLibrary && !filter.IncludeLibraries {
		return fmt.Errorf("contract '%s' is a library, but libraries are not included in the filter", ref)
	}
	if contract.IsInterface && !filter.IncludeInterface {
		return fmt.Errorf("contract '%s' is an interface, but interfaces are not included in the filter", ref)
	}
	if contract.IsAbstract && !filter.IncludeAbstract {
		return fmt.Errorf("contract '%s' is abstract, but abstract contracts are not included in the filter", ref)
	}
	return nil
}

// isProxyContract determines if a contract is likely a proxy contract
func (r *ContractResolver) isProxyContract(contract *domain.ContractInfo) bool {
	// Check if the contract name contains "Proxy"
	if !strings.Contains(contract.Name, "Proxy") {
		return false
	}
	
	// Exclude test contracts
	if strings.Contains(contract.Path, "test/") || strings.Contains(contract.Path, "Test") {
		return false
	}
	
	// Exclude mock contracts
	if strings.Contains(contract.Name, "Mock") || strings.Contains(contract.Path, "mock/") {
		return false
	}
	
	return true
}

// buildFilterErrorMessage builds an error message when contracts were filtered out
func (r *ContractResolver) buildFilterErrorMessage(ref string, contracts []*domain.ContractInfo, filter usecase.ContractFilter) error {
	var filteredTypes []string
	hasLibrary := false
	hasInterface := false
	hasAbstract := false

	for _, contract := range contracts {
		if contract.IsLibrary {
			hasLibrary = true
		}
		if contract.IsInterface {
			hasInterface = true
		}
		if contract.IsAbstract {
			hasAbstract = true
		}
	}

	if hasLibrary && !filter.IncludeLibraries {
		filteredTypes = append(filteredTypes, "libraries")
	}
	if hasInterface && !filter.IncludeInterface {
		filteredTypes = append(filteredTypes, "interfaces")
	}
	if hasAbstract && !filter.IncludeAbstract {
		filteredTypes = append(filteredTypes, "abstract contracts")
	}

	if len(filteredTypes) > 0 {
		return fmt.Errorf("found contracts matching '%s', but they are %s which are excluded by the current filter", 
			ref, strings.Join(filteredTypes, " or "))
	}

	return fmt.Errorf("no contracts matching '%s' meet the filter criteria", ref)
}

// buildAmbiguousErrorMessage builds an error message for ambiguous matches
func (r *ContractResolver) buildAmbiguousErrorMessage(ref string, contracts []*domain.ContractInfo) error {
	// Sort contracts by artifact path for consistent output
	sortedContracts := make([]*domain.ContractInfo, len(contracts))
	copy(sortedContracts, contracts)
	
	sort.Slice(sortedContracts, func(i, j int) bool {
		// Sort by full artifact path (path:name)
		artifactI := fmt.Sprintf("%s:%s", sortedContracts[i].Path, sortedContracts[i].Name)
		artifactJ := fmt.Sprintf("%s:%s", sortedContracts[j].Path, sortedContracts[j].Name)
		return artifactI < artifactJ
	})

	var suggestions []string
	for _, contract := range sortedContracts {
		suggestion := fmt.Sprintf("  - %s (%s)", contract.Name, contract.Path)
		suggestions = append(suggestions, suggestion)
	}

	return fmt.Errorf("multiple contracts found matching '%s' - use full path:contract format to disambiguate:\n%s",
		ref, strings.Join(suggestions, "\n"))
}

// ContractResolverAdapter adapts the contract resolver for the usecase layer
type ContractResolverAdapter struct {
	indexer  *Indexer
	resolver *ContractResolver
}

// NewContractResolverAdapter creates a new contract resolver adapter
func NewContractResolverAdapter(cfg *config.RuntimeConfig, selector usecase.InteractiveSelector) (*ContractResolverAdapter, error) {
	indexer := NewIndexer(cfg.ProjectRoot)
	
	// Index contracts
	if err := indexer.Index(); err != nil {
		return nil, fmt.Errorf("failed to index contracts: %w", err)
	}

	resolver := NewContractResolver(cfg, indexer, selector)
	
	return &ContractResolverAdapter{
		indexer:  indexer,
		resolver: resolver,
	}, nil
}

// ResolveContract resolves a contract reference to a contract
func (a *ContractResolverAdapter) ResolveContract(ctx context.Context, contractRef string) (*domain.ContractInfo, error) {
	return a.resolver.ResolveContract(ctx, contractRef)
}

// ResolveContractWithFilter resolves a contract reference with filtering
func (a *ContractResolverAdapter) ResolveContractWithFilter(ctx context.Context, contractRef string, filter usecase.ContractFilter) (*domain.ContractInfo, error) {
	return a.resolver.ResolveContractWithFilter(ctx, contractRef, filter)
}

// GetProxyContracts returns all available proxy contracts
func (a *ContractResolverAdapter) GetProxyContracts(ctx context.Context) ([]*domain.ContractInfo, error) {
	return a.resolver.GetProxyContracts(ctx)
}

// SelectProxyContract interactively selects a proxy contract
func (a *ContractResolverAdapter) SelectProxyContract(ctx context.Context) (*domain.ContractInfo, error) {
	return a.resolver.SelectProxyContract(ctx)
}

// GetContract retrieves a contract by key (delegated to indexer for compatibility)
func (a *ContractResolverAdapter) GetContract(ctx context.Context, key string) (*domain.ContractInfo, error) {
	return a.indexer.GetContract(key)
}

// SearchContracts searches for contracts matching a pattern (delegated to indexer for compatibility)
func (a *ContractResolverAdapter) SearchContracts(ctx context.Context, pattern string) []*domain.ContractInfo {
	return a.indexer.SearchContracts(pattern)
}

// GetContractByArtifact retrieves a contract by its artifact path (delegated to indexer for compatibility)
func (a *ContractResolverAdapter) GetContractByArtifact(ctx context.Context, artifact string) *domain.ContractInfo {
	return a.indexer.GetContractByArtifact(artifact)
}

// RefreshIndex refreshes the contract index (delegated to indexer for compatibility)
func (a *ContractResolverAdapter) RefreshIndex(ctx context.Context) error {
	if os.Getenv("TREB_TEST_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "DEBUG: RefreshIndex called on ContractResolverAdapter\n")
	}
	return a.indexer.Index()
}

// GetIndexer returns the internal indexer (for sharing with script resolver)
func (a *ContractResolverAdapter) GetIndexer() *Indexer {
	return a.indexer
}

// Ensure the adapter implements both interfaces
var _ usecase.ContractResolver = (*ContractResolverAdapter)(nil)
var _ usecase.ContractIndexer = (*ContractResolverAdapter)(nil)