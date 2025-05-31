package interactive

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// PickDeployment helps users select a deployment when multiple matches exist (v2 registry)
func PickDeployment(matches []*types.Deployment, prompt string) (*types.Deployment, error) {
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

// formatDeploymentOptions creates display strings for deployment selection (v2 registry)
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