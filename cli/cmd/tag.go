package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/pkg/interactive"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

var (
	addTag    string
	removeTag string
)

var tagCmd = &cobra.Command{
	Use:   "tag <deployment|address>",
	Short: "Manage deployment tags",
	Long: `Add or remove version tags on deployments.
Without flags, shows current tags.

Examples:
  treb tag Counter:v1                  # Show current tags
  treb tag Counter:v1 --add v1.0.0     # Add a tag
  treb tag Counter:v1 --remove v1.0.0  # Remove a tag`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		identifier := args[0]

		// Validate flags
		if addTag != "" && removeTag != "" {
			checkError(fmt.Errorf("cannot use --add and --remove together"))
		}

		if err := manageDeploymentTags(identifier); err != nil {
			checkError(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)

	tagCmd.Flags().StringVar(&addTag, "add", "", "Add a tag to the deployment")
	tagCmd.Flags().StringVar(&removeTag, "remove", "", "Remove a tag from the deployment")
}

func manageDeploymentTags(identifier string) error {
	// Initialize v2 registry manager
	manager, err := registry.NewManager(".")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	// Find deployment
	deployment, err := findDeployment(identifier, manager)
	if err != nil {
		return err
	}

	// If no flags, just show current tags
	if addTag == "" && removeTag == "" {
		return showDeploymentTags(deployment)
	}

	// Add tag
	if addTag != "" {
		return addDeploymentTag(deployment, addTag, manager)
	}

	// Remove tag
	return removeDeploymentTag(deployment, removeTag, manager)
}

func findDeployment(identifier string, manager *registry.Manager) (*types.Deployment, error) {
	allDeployments := manager.GetAllDeployments()
	var matches []*types.Deployment

	// Check if identifier is an address (starts with 0x)
	if strings.HasPrefix(strings.ToLower(identifier), "0x") {
		// Try to find by address
		for _, deployment := range allDeployments {
			if strings.EqualFold(deployment.Address, identifier) {
				matches = append(matches, deployment)
			}
		}
	} else {
		// Search by contract name or deployment ID
		for _, deployment := range allDeployments {
			// Check contract name match
			if strings.EqualFold(deployment.ContractName, identifier) {
				matches = append(matches, deployment)
				continue
			}

			// Check full ID match
			if strings.Contains(strings.ToLower(deployment.ID), strings.ToLower(identifier)) {
				matches = append(matches, deployment)
				continue
			}
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no deployment found matching '%s'", identifier)
	}

	if len(matches) == 1 {
		return matches[0], nil
	}

	// Multiple matches - use interactive picker
	if IsNonInteractive() {
		return nil, fmt.Errorf("multiple deployments found matching '%s', please be more specific", identifier)
	}

	return interactive.PickDeployment(matches, "Multiple deployments found. Select one:")
}

func showDeploymentTags(deployment *types.Deployment) error {
	// Color styles
	titleStyle := color.New(color.FgCyan, color.Bold)
	labelStyle := color.New(color.FgWhite, color.Bold)
	addressStyle := color.New(color.FgGreen, color.Bold)
	tagStyle := color.New(color.FgCyan)

	displayName := deployment.ContractDisplayName()
	if deployment.Label != "" {
		displayName += ":" + deployment.Label
	}

	fmt.Println()
	titleStyle.Printf("Deployment: %s/%d/%s\n", deployment.Namespace, deployment.ChainID, displayName)

	labelStyle.Print("Address: ")
	addressStyle.Println(deployment.Address)

	labelStyle.Print("Tags:    ")
	if len(deployment.Tags) == 0 {
		color.New(color.Faint).Println("No tags")
	} else {
		// Sort tags for consistent display
		sortedTags := make([]string, len(deployment.Tags))
		copy(sortedTags, deployment.Tags)
		sort.Strings(sortedTags)

		for i, tag := range sortedTags {
			if i > 0 {
				fmt.Print(", ")
			}
			tagStyle.Print(tag)
		}
		fmt.Println()
	}
	fmt.Println()

	return nil
}

func addDeploymentTag(deployment *types.Deployment, tag string, manager *registry.Manager) error {
	// Check if tag already exists
	for _, existingTag := range deployment.Tags {
		if existingTag == tag {
			color.New(color.FgYellow).Printf("⚠️  Deployment already has tag '%s'\n", tag)
			return nil
		}
	}

	// Add tag and save
	if err := manager.AddTag(deployment.ID, tag); err != nil {
		return fmt.Errorf("failed to add tag: %w", err)
	}

	// Show success
	displayName := deployment.ContractDisplayName()
	if deployment.Label != "" {
		displayName += ":" + deployment.Label
	}

	color.New(color.FgGreen).Printf("✅ Added tag '%s' to %s/%d/%s\n",
		tag,
		deployment.Namespace,
		deployment.ChainID,
		displayName,
	)

	// Show all tags (reload deployment to get updated tags)
	fmt.Print("\nCurrent tags: ")
	tagStyle := color.New(color.FgCyan)
	// Get updated deployment from manager
	updatedDeployment, err := manager.GetDeployment(deployment.ID)
	if err != nil {
		return fmt.Errorf("failed to reload deployment: %w", err)
	}
	allTags := make([]string, len(updatedDeployment.Tags))
	copy(allTags, updatedDeployment.Tags)
	sort.Strings(allTags)
	for i, t := range allTags {
		if i > 0 {
			fmt.Print(", ")
		}
		tagStyle.Print(t)
	}
	fmt.Println()

	return nil
}

func removeDeploymentTag(deployment *types.Deployment, tag string, manager *registry.Manager) error {
	// Check if tag exists
	found := false
	for _, existingTag := range deployment.Tags {
		if existingTag == tag {
			found = true
			break
		}
	}

	if !found {
		color.New(color.FgYellow).Printf("⚠️  Deployment doesn't have tag '%s'\n", tag)
		return nil
	}

	// Remove tag and save
	if err := manager.RemoveTag(deployment.ID, tag); err != nil {
		return fmt.Errorf("failed to remove tag: %w", err)
	}

	// Show success
	displayName := deployment.ContractDisplayName()
	if deployment.Label != "" {
		displayName += ":" + deployment.Label
	}

	color.New(color.FgGreen).Printf("✅ Removed tag '%s' from %s/%d/%s\n",
		tag,
		deployment.Namespace,
		deployment.ChainID,
		displayName,
	)

	// Show remaining tags
	fmt.Print("\nRemaining tags: ")
	remainingTags := make([]string, 0)
	for _, t := range deployment.Tags {
		if t != tag {
			remainingTags = append(remainingTags, t)
		}
	}

	if len(remainingTags) == 0 {
		color.New(color.Faint).Print("No tags")
	} else {
		tagStyle := color.New(color.FgCyan)
		sort.Strings(remainingTags)
		for i, t := range remainingTags {
			if i > 0 {
				fmt.Print(", ")
			}
			tagStyle.Print(t)
		}
	}
	fmt.Println()

	return nil
}
