package resolvers

import (
	"fmt"
	"strings"

	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/cli/pkg/interactive"
)

// ResolveContract resolves a contract by name or path, respecting the interactive context
func (c *Context) ResolveContract(nameOrPath string, filter contracts.QueryFilter) (*contracts.ContractInfo, error) {
	// Use the global indexer
	indexer, err := contracts.GetGlobalIndexer(c.projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize contract indexer: %w", err)
	}

	// First try to get by exact key (path:name format)
	if strings.Contains(nameOrPath, ":") {
		if contract, err := indexer.GetContract(nameOrPath); err == nil {
			// Check if it matches the filter
			if !filter.IncludeLibraries && contract.IsLibrary {
				return nil, fmt.Errorf("contract '%s' is a library, but libraries are not included in the filter", nameOrPath)
			}
			if !filter.IncludeInterface && contract.IsInterface {
				return nil, fmt.Errorf("contract '%s' is an interface, but interfaces are not included in the filter", nameOrPath)
			}
			if !filter.IncludeAbstract && contract.IsAbstract {
				return nil, fmt.Errorf("contract '%s' is abstract, but abstract contracts are not included in the filter", nameOrPath)
			}
			return contract, nil
		}
	}

	// Find matching contracts using the filter
	matches := indexer.FindContractByName(nameOrPath, filter)

	// If no exact matches, try searching for partial matches
	if len(matches) == 0 {
		matches = indexer.SearchContracts(nameOrPath)
		// Apply the filter manually since SearchContracts doesn't use filters
		var filteredMatches []*contracts.ContractInfo
		for _, contract := range matches {
			if !filter.IncludeLibraries && contract.IsLibrary {
				continue
			}
			if !filter.IncludeInterface && contract.IsInterface {
				continue
			}
			if !filter.IncludeAbstract && contract.IsAbstract {
				continue
			}
			filteredMatches = append(filteredMatches, contract)
		}
		matches = filteredMatches
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("contract '%s' not found", nameOrPath)
	}

	// Handle multiple matches
	if len(matches) > 1 {
		if c.interactive {
			prompt := fmt.Sprintf("Multiple contracts matching '%s' found. Select one:", nameOrPath)
			return interactive.SelectContract(matches, prompt)
		} else {
			// Non-interactive: return error with suggestions
			var suggestions []string
			for _, match := range matches {
				suggestion := fmt.Sprintf("  - %s (%s)", match.Name, match.Path)
				suggestions = append(suggestions, suggestion)
			}
			return nil, fmt.Errorf("multiple contracts found matching '%s' - use full path:contract format to disambiguate:\n%s", nameOrPath, strings.Join(suggestions, "\n"))
		}
	}

	return matches[0], nil
}

// ResolveContractForImplementation resolves a contract suitable for use as an implementation
// Uses ProjectFilter by default (excludes libraries, interfaces, and abstract contracts)
func (c *Context) ResolveContractForImplementation(nameOrPath string) (*contracts.ContractInfo, error) {
	return c.ResolveContract(nameOrPath, contracts.ProjectFilter())
}

// ResolveContractForProxy resolves a contract suitable for use as a proxy
// Uses DefaultFilter (includes libraries) since many proxy contracts come from libraries
func (c *Context) ResolveContractForProxy(nameOrPath string) (*contracts.ContractInfo, error) {
	return c.ResolveContract(nameOrPath, contracts.DefaultFilter())
}

// ResolveContractForLibrary resolves a contract suitable for library deployment
// Uses filter that only includes libraries
func (c *Context) ResolveContractForLibrary(nameOrPath string) (*contracts.ContractInfo, error) {
	filter := contracts.QueryFilter{
		IncludeLibraries: true,
		IncludeInterface: false,
		IncludeAbstract:  false,
	}
	return c.ResolveContract(nameOrPath, filter)
}

// MustResolveContract resolves a contract and panics if it fails
// Should only be used in contexts where failure is truly unexpected
func (c *Context) MustResolveContract(nameOrPath string, filter contracts.QueryFilter) *contracts.ContractInfo {
	contract, err := c.ResolveContract(nameOrPath, filter)
	if err != nil {
		panic(fmt.Sprintf("failed to resolve contract '%s': %v", nameOrPath, err))
	}
	return contract
}

// ResolveProxyContracts returns all available proxy contracts
func (c *Context) ResolveProxyContracts() ([]*contracts.ContractInfo, error) {
	// Use the global indexer (always includes libraries)
	indexer, err := contracts.GetGlobalIndexer(c.projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize contract indexer: %w", err)
	}

	// Get all proxy contracts (including from libraries)
	filter := contracts.DefaultFilter() // Include libraries for proxy contracts
	proxyContracts := indexer.GetProxyContractsFiltered(filter)
	if len(proxyContracts) == 0 {
		return nil, fmt.Errorf("no proxy contracts found. Make sure you have proxy contracts in your project")
	}

	return proxyContracts, nil
}

// SelectProxyContract prompts the user to select a proxy contract in interactive mode
func (c *Context) SelectProxyContract() (*contracts.ContractInfo, error) {
	if !c.interactive {
		return nil, fmt.Errorf("proxy contract selection requires interactive mode")
	}

	proxyContracts, err := c.ResolveProxyContracts()
	if err != nil {
		return nil, err
	}

	return interactive.SelectContract(proxyContracts, "Select proxy contract:")
}
