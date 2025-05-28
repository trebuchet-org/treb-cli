package cmd

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/fatih/color"
	"github.com/trebuchet-org/treb-cli/cli/pkg/interactive"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
)

// pickDeployment finds deployments matching the identifier and allows selection if multiple matches
// Returns the selected deployment or error
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

	// Multiple matches - check if in non-interactive mode
	if IsNonInteractive() {
		return nil, fmt.Errorf("multiple deployments found matching '%s'. Please be more specific:\n%s", 
			identifier, formatMatchSuggestions(matches))
	}

	// Interactive mode - let user pick
	selector := interactive.NewSelector()
	var options []string
	for _, deployment := range matches {
		// Format option with network/namespace/name/address
		displayName := deployment.Entry.GetDisplayName()
		option := fmt.Sprintf("%s/%s/%s (%s)", 
			deployment.NetworkName, 
			deployment.Entry.Namespace, 
			displayName, 
			deployment.Address.Hex()[:10]+"...")
		options = append(options, option)
	}

	// Special formatting for single match
	if len(matches) == 1 {
		deployment := matches[0]
		displayName := deployment.Entry.GetDisplayName()
		
		// Show full deployment info
		fmt.Printf("\nFound deployment:\n")
		fmt.Printf("  Contract:  %s\n", displayName)
		fmt.Printf("  Network:   %s\n", deployment.NetworkName)
		fmt.Printf("  Namespace: %s\n", deployment.Entry.Namespace)
		fmt.Printf("  Address:   %s\n", deployment.Address.Hex())
		fmt.Printf("  Type:      %s\n", deployment.Entry.Type)
		
		// Confirm selection
		confirm, err := selector.PromptConfirm("Continue with this deployment?", true)
		if err != nil || !confirm {
			return nil, fmt.Errorf("deployment selection cancelled")
		}
		return matches[0], nil
	}

	_, selectedIndex, err := selector.SelectOption(
		fmt.Sprintf("Multiple deployments found matching '%s'. Select one:", identifier),
		options,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("deployment selection cancelled")
	}

	return matches[selectedIndex], nil
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

// findDeploymentByAddress finds a deployment by its address across all networks and namespaces
func findDeploymentByAddress(address common.Address, registryManager *registry.Manager) (*registry.DeploymentInfo, error) {
	allDeployments := registryManager.GetAllDeployments()
	
	for _, deployment := range allDeployments {
		if deployment.Address == address {
			return deployment, nil
		}
	}
	
	return nil, fmt.Errorf("no deployment found at address %s", address.Hex())
}

// formatDeploymentInfo formats deployment information for display
func formatDeploymentInfo(deployment *registry.DeploymentInfo) string {
	labelStyle := color.New(color.FgCyan, color.Bold)
	valueStyle := color.New(color.FgWhite)
	
	// Build the output
	var output strings.Builder
	
	labelStyle.Fprint(&output, "Contract:  ")
	valueStyle.Fprintln(&output, deployment.Entry.GetDisplayName())
	
	labelStyle.Fprint(&output, "Network:   ")
	valueStyle.Fprintln(&output, deployment.NetworkName)
	
	labelStyle.Fprint(&output, "Namespace: ")
	valueStyle.Fprintln(&output, deployment.Entry.Namespace)
	
	labelStyle.Fprint(&output, "Address:   ")
	valueStyle.Fprintln(&output, deployment.Address.Hex())
	
	labelStyle.Fprint(&output, "Type:      ")
	valueStyle.Fprintln(&output, deployment.Entry.Type)
	
	return output.String()
}

