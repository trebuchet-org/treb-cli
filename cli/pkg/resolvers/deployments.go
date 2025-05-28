package resolvers

import (
	"fmt"
	"strings"

	"github.com/trebuchet-org/treb-cli/cli/pkg/interactive"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
)

// ResolveDeployment resolves a deployment by identifier, respecting the interactive context
func (c *Context) ResolveDeployment(identifier string, registryManager *registry.Manager) (*registry.DeploymentInfo, error) {
	if c.interactive {
		// Use interactive resolution with basic picker
		return c.resolveDeploymentInteractive(identifier, registryManager, "", "")
	} else {
		// Use non-interactive resolution - error on multiple matches
		return c.resolveDeploymentNonInteractive(identifier, registryManager, "", "")
	}
}

// ResolveDeploymentWithFilters resolves a deployment with network and namespace filters
func (c *Context) ResolveDeploymentWithFilters(identifier string, registryManager *registry.Manager, networkFilter, namespaceFilter string) (*registry.DeploymentInfo, error) {
	if c.interactive {
		return c.resolveDeploymentInteractive(identifier, registryManager, networkFilter, namespaceFilter)
	} else {
		return c.resolveDeploymentNonInteractive(identifier, registryManager, networkFilter, namespaceFilter)
	}
}

// ResolveDeploymentForProxy resolves a deployment suitable as a proxy implementation
// Filters out proxy deployments to ensure only actual implementations are returned
func (c *Context) ResolveDeploymentForProxy(identifier string, registryManager *registry.Manager, chainID uint64, namespace string) (*registry.DeploymentInfo, error) {
	if c.interactive {
		// Use interactive resolution for implementation deployments
		return interactive.ResolveImplementationDeployment(identifier, registryManager, chainID, namespace)
	} else {
		// Use non-interactive resolution for implementation deployments
		matches := registryManager.QueryDeployments(identifier, chainID, namespace)
		
		// Filter out proxy deployments
		matches = interactive.FilterOutProxies(matches)
		
		if len(matches) == 0 {
			// Try without namespace constraint if no matches found
			if namespace != "" {
				matches = registryManager.QueryDeployments(identifier, chainID, "")
				matches = interactive.FilterOutProxies(matches)
				if len(matches) == 0 {
					return nil, fmt.Errorf("no implementation deployments found matching '%s' on this network", identifier)
				}
			} else {
				return nil, fmt.Errorf("no implementation deployments found matching '%s'", identifier)
			}
		}

		// Error if multiple matches in non-interactive mode
		if len(matches) > 1 {
			return nil, fmt.Errorf("multiple implementation deployments found matching '%s' - use network/namespace filters to narrow results:\n%s", 
				identifier, c.formatDeploymentSuggestions(matches))
		}

		return matches[0], nil
	}
}

// ResolveMultipleDeployments resolves deployments and allows selection of one or all
func (c *Context) ResolveMultipleDeployments(identifier string, registryManager *registry.Manager, allowAll bool) ([]*registry.DeploymentInfo, error) {
	// Get all deployments
	allDeployments := registryManager.GetAllDeployments()

	// Find matching deployments
	var matches []*registry.DeploymentInfo
	identifierLower := strings.ToLower(identifier)

	for _, deployment := range allDeployments {
		// Check if identifier is an address
		if strings.ToLower(deployment.Address.Hex()) == identifierLower {
			matches = append(matches, deployment)
			continue
		}

		// Check if identifier matches or is contained in display name
		displayName := deployment.Entry.GetDisplayName()
		if strings.EqualFold(displayName, identifier) || strings.Contains(strings.ToLower(displayName), identifierLower) {
			matches = append(matches, deployment)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no deployment found matching '%s'", identifier)
	}

	// If single match, return it
	if len(matches) == 1 {
		return matches, nil
	}

	// Multiple matches
	if !allowAll {
		// Pick single deployment
		selected, err := c.ResolveDeployment(identifier, registryManager)
		if err != nil {
			return nil, err
		}
		return []*registry.DeploymentInfo{selected}, nil
	}

	if c.interactive {
		// Ask if user wants to apply to all
		selector := interactive.NewSelector()
		applyAll, err := selector.PromptConfirm(
			fmt.Sprintf("Found %d deployments matching '%s'. Apply to all?", len(matches), identifier),
			false,
		)
		if err != nil {
			return nil, fmt.Errorf("selection cancelled")
		}

		if applyAll {
			return matches, nil
		}

		// Pick single deployment
		selected, err := c.ResolveDeployment(identifier, registryManager)
		if err != nil {
			return nil, err
		}
		return []*registry.DeploymentInfo{selected}, nil
	} else {
		// In non-interactive mode, default to applying to all matches
		return matches, nil
	}
}

// resolveDeploymentInteractive uses the interactive deployment picker
func (c *Context) resolveDeploymentInteractive(identifier string, registryManager *registry.Manager, networkFilter, namespaceFilter string) (*registry.DeploymentInfo, error) {
	// Always use the filtering logic - it handles interactive selection internally
	return c.resolveDeploymentWithFiltersLogic(identifier, registryManager, networkFilter, namespaceFilter, true)
}

// resolveDeploymentNonInteractive provides non-interactive deployment resolution
func (c *Context) resolveDeploymentNonInteractive(identifier string, registryManager *registry.Manager, networkFilter, namespaceFilter string) (*registry.DeploymentInfo, error) {
	return c.resolveDeploymentWithFiltersLogic(identifier, registryManager, networkFilter, namespaceFilter, false)
}

// resolveDeploymentWithFiltersLogic implements the core filtering and resolution logic
func (c *Context) resolveDeploymentWithFiltersLogic(identifier string, registryManager *registry.Manager, networkFilter, namespaceFilter string, isInteractive bool) (*registry.DeploymentInfo, error) {
	// Get all deployments
	allDeployments := registryManager.GetAllDeployments()

	// Apply filters first
	var filteredDeployments []*registry.DeploymentInfo
	for _, deployment := range allDeployments {
		// Apply network filter
		if networkFilter != "" && !strings.EqualFold(deployment.NetworkName, networkFilter) {
			continue
		}
		
		// Apply namespace filter
		if namespaceFilter != "" && !strings.EqualFold(deployment.Entry.Namespace, namespaceFilter) {
			continue
		}
		
		filteredDeployments = append(filteredDeployments, deployment)
	}

	// Now search within filtered deployments
	var matches []*registry.DeploymentInfo
	identifierLower := strings.ToLower(identifier)

	for _, deployment := range filteredDeployments {
		// Check if identifier is an address
		if strings.ToLower(deployment.Address.Hex()) == identifierLower {
			matches = append(matches, deployment)
			continue
		}

		// Check if identifier matches or is contained in display name
		displayName := deployment.Entry.GetDisplayName()
		if strings.EqualFold(displayName, identifier) || strings.Contains(strings.ToLower(displayName), identifierLower) {
			matches = append(matches, deployment)
		}
	}

	if len(matches) == 0 {
		filters := ""
		if networkFilter != "" || namespaceFilter != "" {
			var filterParts []string
			if networkFilter != "" {
				filterParts = append(filterParts, fmt.Sprintf("network=%s", networkFilter))
			}
			if namespaceFilter != "" {
				filterParts = append(filterParts, fmt.Sprintf("namespace=%s", namespaceFilter))
			}
			filters = fmt.Sprintf(" (filters: %s)", strings.Join(filterParts, ", "))
		}
		return nil, fmt.Errorf("no deployment found matching '%s'%s", identifier, filters)
	}

	// If single match, return it
	if len(matches) == 1 {
		return matches[0], nil
	}

	// Multiple matches
	if isInteractive {
		// Use interactive selector
		return interactive.SelectDeployment(matches, fmt.Sprintf("Multiple deployments found matching '%s'", identifier))
	} else {
		// Error in non-interactive mode
		return nil, fmt.Errorf("multiple deployments found matching '%s' - narrow results further:\n%s", 
			identifier, c.formatDeploymentSuggestions(matches))
	}
}

// formatDeploymentSuggestions formats multiple deployment matches into a helpful error message
func (c *Context) formatDeploymentSuggestions(matches []*registry.DeploymentInfo) string {
	if len(matches) == 0 {
		return ""
	}

	var suggestions []string
	for _, match := range matches {
		displayName := match.Entry.GetDisplayName()
		suggestion := fmt.Sprintf("  - %s/%s/%s (%s)", 
			match.NetworkName, 
			match.Entry.Namespace, 
			displayName, 
			match.Address.Hex()[:10]+"...")
		suggestions = append(suggestions, suggestion)
	}

	return strings.Join(suggestions, "\n")
}