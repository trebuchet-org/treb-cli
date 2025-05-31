package interactive

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// SelectDeployment helps users select a deployment when multiple matches exist
func SelectDeployment(matches []*registry.DeploymentInfo, prompt string) (*registry.DeploymentInfo, error) {
	if len(matches) == 0 {
		return nil, fmt.Errorf("no deployments found")
	}

	// If only one match, return it directly
	if len(matches) == 1 {
		return matches[0], nil
	}

	// Multiple matches - need user to disambiguate
	options := formatDeploymentOptions(matches)

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "👉 {{ . | cyan }}",
		Inactive: "   {{ . | faint }}",
		Selected: "👍 {{ . | green }}",
		Help:     color.New(color.FgYellow).Sprint("Use arrow keys to navigate, Enter to select"),
	}

	promptSelect := promptui.Select{
		Label:     prompt,
		Items:     options,
		Templates: templates,
		Size:      10,
	}

	index, _, err := promptSelect.Run()
	if err != nil {
		return nil, fmt.Errorf("selection cancelled: %w", err)
	}

	return matches[index], nil
}

// ResolveDeployment finds and potentially disambiguates a deployment by query
// It uses the provided chainID and env to narrow down results
func ResolveDeployment(query string, registryManager *registry.Manager, chainID uint64, env string) (*registry.DeploymentInfo, error) {
	// Query deployments using the registry manager with context
	matches := registryManager.QueryDeployments(query, chainID, env)

	if len(matches) == 0 {
		// Try without env constraint if no matches found
		if env != "" {
			matches = registryManager.QueryDeployments(query, chainID, "")
			if len(matches) == 0 {
				return nil, fmt.Errorf("no deployments found matching '%s' on this network", query)
			}
		} else {
			return nil, fmt.Errorf("no deployments found matching '%s'", query)
		}
	}

	// Use picker if multiple matches
	if len(matches) > 1 {
		prompt := fmt.Sprintf("Multiple deployments matching '%s' found. Select one:", query)
		return SelectDeployment(matches, prompt)
	}

	return matches[0], nil
}

// ResolveImplementationDeployment finds a deployment suitable as a proxy implementation
// It filters out proxy deployments to ensure only actual implementations are returned
func ResolveImplementationDeployment(query string, registryManager *registry.Manager, chainID uint64, env string) (*registry.DeploymentInfo, error) {
	// Query deployments using the registry manager with context
	matches := registryManager.QueryDeployments(query, chainID, env)

	// Filter out proxy deployments
	matches = FilterOutProxies(matches)

	if len(matches) == 0 {
		// Try without env constraint if no matches found
		if env != "" {
			matches = registryManager.QueryDeployments(query, chainID, "")
			matches = FilterOutProxies(matches)
			if len(matches) == 0 {
				return nil, fmt.Errorf("no implementation deployments found matching '%s' on this network", query)
			}
		} else {
			return nil, fmt.Errorf("no implementation deployments found matching '%s'", query)
		}
	}

	// Use picker if multiple matches
	if len(matches) > 1 {
		prompt := fmt.Sprintf("Multiple implementation deployments matching '%s' found. Select one:", query)
		return SelectDeployment(matches, prompt)
	}

	return matches[0], nil
}

// formatDeploymentOptions creates display strings for deployment selection
func formatDeploymentOptions(deployments []*registry.DeploymentInfo) []string {
	options := make([]string, len(deployments))
	for i, deployment := range deployments {
		entry := deployment.Entry

		// Format deployment info
		deploymentInfo := fmt.Sprintf("%s (%s)",
			color.New(color.FgWhite, color.Bold).Sprint(entry.ContractName),
			color.New(color.FgBlue).Sprint(entry.ShortID),
		)

		// Add label if present
		if entry.Label != "" {
			deploymentInfo += fmt.Sprintf(" [%s]", color.New(color.FgMagenta).Sprint(entry.Label))
		}

		// Add network and address
		addressDisplay := entry.Address.Hex()
		if len(addressDisplay) > 10 {
			addressDisplay = addressDisplay[:10] + "..."
		}

		networkInfo := fmt.Sprintf("%s: %s",
			color.New(color.FgGreen).Sprint(deployment.NetworkName),
			color.New(color.FgYellow).Sprint(addressDisplay),
		)

		// Add deployment status
		statusDisplay := ""
		if entry.Deployment.Status == "pending_safe" {
			statusDisplay = color.New(color.FgRed).Sprint(" (pending)")
		}

		options[i] = fmt.Sprintf("%s - %s%s", deploymentInfo, networkInfo, statusDisplay)
	}
	return options
}

// FilterByNetwork filters deployments to only those on the specified network
func FilterByNetwork(deployments []*registry.DeploymentInfo, chainID uint64) []*registry.DeploymentInfo {
	var filtered []*registry.DeploymentInfo
	chainIDStr := fmt.Sprintf("%d", chainID)

	for _, deployment := range deployments {
		if deployment.ChainID == chainIDStr {
			filtered = append(filtered, deployment)
		}
	}

	return filtered
}

// FilterByStatus filters deployments by their status
func FilterByStatus(deployments []*registry.DeploymentInfo, status string) []*registry.DeploymentInfo {
	var filtered []*registry.DeploymentInfo

	for _, deployment := range deployments {
		if strings.EqualFold(string(deployment.Entry.Deployment.Status), status) {
			filtered = append(filtered, deployment)
		}
	}

	return filtered
}

// FilterOutProxies filters out proxy deployments, keeping only implementations
func FilterOutProxies(deployments []*registry.DeploymentInfo) []*registry.DeploymentInfo {
	var filtered []*registry.DeploymentInfo
	
	for _, deployment := range deployments {
		// Exclude proxy deployments
		if deployment.Entry.Type != "proxy" {
			filtered = append(filtered, deployment)
		}
	}
	
	return filtered
}

// V2 Deployment Picker Functions

// PickDeploymentV2 helps users select a deployment when multiple matches exist (v2 registry)
func PickDeploymentV2(matches []*types.Deployment) (*types.Deployment, error) {
	if len(matches) == 0 {
		return nil, fmt.Errorf("no deployments found")
	}

	// If only one match, return it directly
	if len(matches) == 1 {
		return matches[0], nil
	}

	// Multiple matches - need user to disambiguate
	options := formatDeploymentOptionsV2(matches)

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "👉 {{ . | cyan }}",
		Inactive: "   {{ . | faint }}",
		Selected: "👍 {{ . | green }}",
		Help:     color.New(color.FgYellow).Sprint("Use arrow keys to navigate, Enter to select"),
	}

	promptSelect := promptui.Select{
		Label:     "Multiple deployments found. Select one:",
		Items:     options,
		Templates: templates,
		Size:      10,
	}

	index, _, err := promptSelect.Run()
	if err != nil {
		return nil, fmt.Errorf("selection cancelled: %w", err)
	}

	return matches[index], nil
}

// formatDeploymentOptionsV2 creates display strings for deployment selection (v2 registry)
func formatDeploymentOptionsV2(deployments []*types.Deployment) []string {
	options := make([]string, len(deployments))
	for i, deployment := range deployments {
		// Format deployment info
		displayName := deployment.ContractName
		if deployment.Label != "" {
			displayName += ":" + deployment.Label
		}

		deploymentInfo := fmt.Sprintf("%s (%s)",
			color.New(color.FgWhite, color.Bold).Sprint(displayName),
			color.New(color.FgBlue).Sprint(deployment.ID),
		)

		// Add network and address
		addressDisplay := deployment.Address
		if len(addressDisplay) > 10 {
			addressDisplay = addressDisplay[:10] + "..."
		}

		networkInfo := fmt.Sprintf("chain %d: %s",
			deployment.ChainID,
			color.New(color.FgYellow).Sprint(addressDisplay),
		)

		// Add tags if present
		tagsDisplay := ""
		if len(deployment.Tags) > 0 {
			tagsDisplay = fmt.Sprintf(" [%s]", 
				color.New(color.FgMagenta).Sprint(strings.Join(deployment.Tags, ", ")))
		}

		options[i] = fmt.Sprintf("%s - %s%s", deploymentInfo, networkInfo, tagsDisplay)
	}
	return options
}
