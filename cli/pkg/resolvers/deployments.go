package resolvers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// ResolveDeployment resolves a deployment by identifier
// Supports formats:
// - Contract name: "Counter"
// - Contract with label: "Counter:v2"
// - Namespace/contract: "staging/Counter"
// - Chain/contract: "11155111/Counter"
// - Namespace/chain/contract: "staging/11155111/Counter"
// - Full deployment ID
// - Address (requires chainID)
func (c *Context) ResolveDeployment(identifier string, manager *registry.Manager, chainID uint64, namespace string) (*types.Deployment, error) {
	var deployment *types.Deployment
	var err error

	// Check if identifier is an address (starts with 0x and is 42 chars)
	if strings.HasPrefix(identifier, "0x") && len(identifier) == 42 {
		// Look up by address
		if chainID == 0 {
			return nil, fmt.Errorf("chain ID is required when looking up by address")
		}
		deployment, err = manager.GetDeploymentByAddress(chainID, identifier)
		if err != nil {
			return nil, fmt.Errorf("deployment not found at address %s on chain %d", identifier, chainID)
		}
		return deployment, nil
	}

	// Parse deployment ID (could be Contract, Contract:label, namespace/Contract, etc.)
	deployments := manager.GetAllDeployments()

	// Filter by namespace if provided
	if namespace != "" {
		filtered := make([]*types.Deployment, 0)
		for _, d := range deployments {
			if d.Namespace == namespace {
				filtered = append(filtered, d)
			}
		}
		deployments = filtered
	}

	// Filter by chain if provided
	if chainID != 0 {
		filtered := make([]*types.Deployment, 0)
		for _, d := range deployments {
			if d.ChainID == chainID {
				filtered = append(filtered, d)
			}
		}
		deployments = filtered
	}

	// Look for matches based on various identifier formats
	matches := make([]*types.Deployment, 0)

	// Try to parse identifier parts
	parts := strings.Split(identifier, "/")

	for _, d := range deployments {
		matched := false

		// Simple match: just contract name or contract:label
		if d.ContractName == identifier || d.GetShortID() == identifier {
			matched = true
		}

		// Match namespace/contract or namespace/contract:label
		if len(parts) == 2 {
			namespace := parts[0]
			contractPart := parts[1]

			// Check if first part is a namespace
			if d.Namespace == namespace && (d.ContractName == contractPart || d.GetShortID() == contractPart) {
				matched = true
			}

			// Check if first part is a chain ID
			if chainID, err := strconv.ParseUint(parts[0], 10, 64); err == nil {
				if d.ChainID == chainID && (d.ContractName == contractPart || d.GetShortID() == contractPart) {
					matched = true
				}
			}
		}

		// Match namespace/chain/contract or similar complex patterns
		if len(parts) == 3 {
			// Could be namespace/chainID/contract
			if d.Namespace == parts[0] {
				if chainID, err := strconv.ParseUint(parts[1], 10, 64); err == nil {
					if d.ChainID == chainID && (d.ContractName == parts[2] || d.GetShortID() == parts[2]) {
						matched = true
					}
				}
			}
		}

		// Match against the full deployment ID
		if d.ID == identifier {
			matched = true
		}

		if matched {
			matches = append(matches, d)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no deployments found matching '%s'", identifier)
	} else if len(matches) == 1 {
		return matches[0], nil
	} else {
		// Multiple matches
		if c.interactive {
			return c.selectDeployment(matches, fmt.Sprintf("Multiple deployments found for '%s'", identifier))
		} else {
			// Non-interactive: return error with suggestions
			var suggestions []string
			for _, match := range matches {
				suggestion := fmt.Sprintf("  - %s (chain:%d/%s/%s)",
					match.ID, match.ChainID, match.Namespace, match.GetDisplayName())
				suggestions = append(suggestions, suggestion)
			}
			return nil, fmt.Errorf("multiple deployments found matching '%s' - be more specific:\n%s",
				identifier, strings.Join(suggestions, "\n"))
		}
	}
}

// selectDeployment helps users select a deployment when multiple matches exist
func (c *Context) selectDeployment(matches []*types.Deployment, prompt string) (*types.Deployment, error) {
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

// formatDeploymentOptions creates display strings for deployment selection
func formatDeploymentOptions(deployments []*types.Deployment) []string {
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
