package usecase

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// ResolveContract is the use case for resolving contract references
type ResolveContract struct {
	config          *config.RuntimeConfig
	contractIndexer ContractIndexer
	selector        InteractiveSelector
	sink            ProgressSink
}

// NewResolveContract creates a new ResolveContract use case
func NewResolveContract(
	cfg *config.RuntimeConfig,
	contractIndexer ContractIndexer,
	selector InteractiveSelector,
	sink ProgressSink,
) *ResolveContract {
	return &ResolveContract{
		config:          cfg,
		contractIndexer: contractIndexer,
		selector:        selector,
		sink:            sink,
	}
}

// ResolveContract resolves a contract reference to a contract
func (uc *ResolveContract) ResolveContract(ctx context.Context, contractRef string) (*domain.ContractInfo, error) {
	// Use default filter (all contracts)
	filter := ContractFilter{
		IncludeLibraries: true,
		IncludeInterface: true,
		IncludeAbstract:  true,
	}
	return uc.ResolveContractWithFilter(ctx, contractRef, filter)
}

// ResolveContractWithFilter resolves a contract reference with filtering
func (uc *ResolveContract) ResolveContractWithFilter(ctx context.Context, contractRef string, filter ContractFilter) (*domain.ContractInfo, error) {
	// Report progress
	uc.sink.OnProgress(ctx, ProgressEvent{
		Stage:   "resolving",
		Message: fmt.Sprintf("Resolving contract: %s", contractRef),
		Spinner: true,
	})

	// First try exact match (path:name format)
	if strings.Contains(contractRef, ":") {
		contract := uc.contractIndexer.GetContractByArtifact(ctx, contractRef)
		if contract != nil {
			// Check if it matches the filter
			if err := uc.validateContractAgainstFilter(contract, filter, contractRef); err != nil {
				return nil, err
			}
			return contract, nil
		}
	}

	// Search for contracts
	contracts := uc.contractIndexer.SearchContracts(ctx, contractRef)
	
	// Filter contracts based on criteria
	filtered := uc.filterContracts(contracts, filter)
	
	if len(filtered) == 0 {
		// Provide helpful error message based on what was filtered out
		if len(contracts) > 0 {
			return nil, uc.buildFilterErrorMessage(contractRef, contracts, filter)
		}
		return nil, fmt.Errorf("contract '%s' not found", contractRef)
	}

	if len(filtered) == 1 {
		return filtered[0], nil
	}

	// Multiple matches - use interactive selector if available
	if uc.selector != nil && !uc.config.NonInteractive {
		selected, err := uc.selector.SelectContract(ctx, filtered, fmt.Sprintf("Multiple contracts found for '%s'. Select one:", contractRef))
		if err != nil {
			return nil, fmt.Errorf("contract selection failed: %w", err)
		}
		return selected, nil
	}

	// Non-interactive mode with multiple matches
	return nil, uc.buildAmbiguousErrorMessage(contractRef, filtered)
}

// GetProxyContracts returns all available proxy contracts
func (uc *ResolveContract) GetProxyContracts(ctx context.Context) ([]*domain.ContractInfo, error) {
	// Report progress
	uc.sink.OnProgress(ctx, ProgressEvent{
		Stage:   "searching",
		Message: "Finding proxy contracts",
		Spinner: true,
	})

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
		contracts := uc.contractIndexer.SearchContracts(ctx, pattern)
		for _, contract := range contracts {
			// Filter to only include actual proxy contracts
			if uc.isProxyContract(contract) {
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
func (uc *ResolveContract) SelectProxyContract(ctx context.Context) (*domain.ContractInfo, error) {
	// Get all proxy contracts
	proxies, err := uc.GetProxyContracts(ctx)
	if err != nil {
		return nil, err
	}

	// If only one proxy, return it
	if len(proxies) == 1 {
		return proxies[0], nil
	}

	// Use interactive selector if available
	if uc.selector != nil && !uc.config.NonInteractive {
		selected, err := uc.selector.SelectContract(ctx, proxies, "Select a proxy contract:")
		if err != nil {
			return nil, fmt.Errorf("proxy selection failed: %w", err)
		}
		return selected, nil
	}

	// Non-interactive mode with multiple proxies
	return nil, fmt.Errorf("multiple proxy contracts found. Please specify --proxy-contract or run in interactive mode")
}

// filterContracts filters contracts based on the provided filter
func (uc *ResolveContract) filterContracts(contracts []*domain.ContractInfo, filter ContractFilter) []*domain.ContractInfo {
	var filtered []*domain.ContractInfo
	for _, contract := range contracts {
		if uc.matchesFilter(contract, filter) {
			filtered = append(filtered, contract)
		}
	}
	return filtered
}

// matchesFilter checks if a contract matches the filter criteria
func (uc *ResolveContract) matchesFilter(contract *domain.ContractInfo, filter ContractFilter) bool {
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
func (uc *ResolveContract) validateContractAgainstFilter(contract *domain.ContractInfo, filter ContractFilter, ref string) error {
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
func (uc *ResolveContract) isProxyContract(contract *domain.ContractInfo) bool {
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
func (uc *ResolveContract) buildFilterErrorMessage(ref string, contracts []*domain.ContractInfo, filter ContractFilter) error {
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
func (uc *ResolveContract) buildAmbiguousErrorMessage(ref string, contracts []*domain.ContractInfo) error {
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

// Ensure the use case implements the interface
var _ ContractResolver = (*ResolveContract)(nil)