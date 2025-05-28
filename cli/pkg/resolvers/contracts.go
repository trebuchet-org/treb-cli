package resolvers

import (
	"fmt"

	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/cli/pkg/interactive"
)

// ResolveContract resolves a contract by name or path, respecting the interactive context
func (c *Context) ResolveContract(nameOrPath string, filter contracts.QueryFilter) (*contracts.ContractInfo, error) {
	if c.interactive {
		// Use interactive resolution
		return interactive.ResolveContract(nameOrPath, filter)
	} else {
		// Use non-interactive resolution
		return interactive.ResolveContractNonInteractive(nameOrPath, filter)
	}
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
		IncludeLibraries:  true,
		IncludeInterface:  false,
		IncludeAbstract:   false,
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