package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/trebuchet-org/treb-cli/cli/pkg/interactive"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
)

// pickDeployment finds and allows selection of deployments matching the identifier
// Returns the selected deployment or nil if cancelled
func pickDeployment(identifier string, registryManager *registry.Manager) (*registry.DeploymentInfo, error) {
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
		return matches[0], nil
	}

	// Multiple matches - check if non-interactive mode
	if IsNonInteractive() {
		return nil, fmt.Errorf("multiple deployments found matching '%s' - use --network and/or --namespace flags to narrow results:\n%s", 
			identifier, formatMatchSuggestions(matches))
	}

	// Multiple matches - use interactive selector
	// Sort matches by network, then env, then contract name
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].NetworkName != matches[j].NetworkName {
			return matches[i].NetworkName < matches[j].NetworkName
		}
		if matches[i].Entry.Namespace != matches[j].Entry.Namespace {
			return matches[i].Entry.Namespace < matches[j].Entry.Namespace
		}
		return matches[i].Entry.ContractName < matches[j].Entry.ContractName
	})

	// Find max identifier length for alignment
	maxIdLen := 0
	for _, match := range matches {
		displayName := match.Entry.GetDisplayName()
		// Add tags to display name
		if len(match.Entry.Tags) > 0 {
			displayName += fmt.Sprintf(" (%s)", match.Entry.Tags[0])
		}
		fullId := fmt.Sprintf("%s/%s/%s", match.NetworkName, match.Entry.Namespace, displayName)
		if len(fullId) > maxIdLen {
			maxIdLen = len(fullId)
		}
	}

	// Create options with aligned addresses
	options := make([]string, len(matches))
	for i, match := range matches {
		displayName := match.Entry.GetDisplayName()
		// Add tags to display name
		if len(match.Entry.Tags) > 0 {
			displayName += fmt.Sprintf(" (%s)", match.Entry.Tags[0])
		}
		fullId := fmt.Sprintf("%s/%s/%s", match.NetworkName, match.Entry.Namespace, displayName)
		padding := strings.Repeat(" ", maxIdLen-len(fullId))
		options[i] = fmt.Sprintf("%s%s  %s", fullId, padding, match.Address.Hex())
	}

	selector := interactive.NewSelector()
	_, selectedIndex, err := selector.SelectOption(
		fmt.Sprintf("Multiple deployments found matching '%s'", identifier),
		options,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("selection cancelled")
	}

	return matches[selectedIndex], nil
}

// pickMultipleDeployments finds deployments and allows selection of one or all
// Returns the selected deployments or nil if cancelled
func pickMultipleDeployments(identifier string, registryManager *registry.Manager, allowAll bool) ([]*registry.DeploymentInfo, error) {
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
		selected, err := pickDeployment(identifier, registryManager)
		if err != nil {
			return nil, err
		}
		return []*registry.DeploymentInfo{selected}, nil
	}

	// Check if non-interactive mode
	if IsNonInteractive() {
		// In non-interactive mode, default to applying to all matches
		return matches, nil
	}

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
	selected, err := pickDeployment(identifier, registryManager)
	if err != nil {
		return nil, err
	}
	return []*registry.DeploymentInfo{selected}, nil
}

// formatMatchSuggestions formats multiple matches into a helpful error message
func formatMatchSuggestions(matches []*registry.DeploymentInfo) string {
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

// pickDeploymentWithFilters finds and allows selection of deployments with network and namespace filters
func pickDeploymentWithFilters(identifier string, registryManager *registry.Manager, networkFilter, namespaceFilter string) (*registry.DeploymentInfo, error) {
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

	// Multiple matches - check if non-interactive mode
	if IsNonInteractive() {
		return nil, fmt.Errorf("multiple deployments found matching '%s' - narrow results further:\n%s", 
			identifier, formatMatchSuggestions(matches))
	}

	// Multiple matches - use interactive selector
	// Sort matches by network, then env, then contract name
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].NetworkName != matches[j].NetworkName {
			return matches[i].NetworkName < matches[j].NetworkName
		}
		if matches[i].Entry.Namespace != matches[j].Entry.Namespace {
			return matches[i].Entry.Namespace < matches[j].Entry.Namespace
		}
		return matches[i].Entry.ContractName < matches[j].Entry.ContractName
	})

	// Find max identifier length for alignment
	maxIdLen := 0
	for _, match := range matches {
		displayName := match.Entry.GetDisplayName()
		// Add tags to display name
		if len(match.Entry.Tags) > 0 {
			displayName += fmt.Sprintf(" (%s)", match.Entry.Tags[0])
		}
		fullId := fmt.Sprintf("%s/%s/%s", match.NetworkName, match.Entry.Namespace, displayName)
		if len(fullId) > maxIdLen {
			maxIdLen = len(fullId)
		}
	}

	// Create options with aligned addresses
	options := make([]string, len(matches))
	for i, match := range matches {
		displayName := match.Entry.GetDisplayName()
		// Add tags to display name
		if len(match.Entry.Tags) > 0 {
			displayName += fmt.Sprintf(" (%s)", match.Entry.Tags[0])
		}
		fullId := fmt.Sprintf("%s/%s/%s", match.NetworkName, match.Entry.Namespace, displayName)
		padding := strings.Repeat(" ", maxIdLen-len(fullId))
		options[i] = fmt.Sprintf("%s%s  %s", fullId, padding, match.Address.Hex())
	}

	selector := interactive.NewSelector()
	_, selectedIndex, err := selector.SelectOption(
		fmt.Sprintf("Multiple deployments found matching '%s'", identifier),
		options,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("selection cancelled")
	}

	return matches[selectedIndex], nil
}
