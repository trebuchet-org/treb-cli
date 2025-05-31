package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
)

var (
	addTag    string
	removeTag string
)

var tagV1Cmd = &cobra.Command{
	Use:   "tag-v1 <contract|address>",
	Short: "Manage deployment tags (legacy)",
	Long: `Add or remove version tags on deployments in v1 registry.
Without flags, shows current tags.

Examples:
  treb tag-v1 Counter                     # Show current tags
  treb tag-v1 Counter --add v1.0.0        # Add a tag
  treb tag-v1 Counter --remove v1.0.0     # Remove a tag`,
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
	rootCmd.AddCommand(tagV1Cmd)
	tagV1Cmd.Flags().StringVar(&addTag, "add", "", "Add a tag to the deployment")
	tagV1Cmd.Flags().StringVar(&removeTag, "remove", "", "Remove a tag from the deployment")
}

func manageDeploymentTags(identifier string) error {
	// Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	// Use shared picker
	deployment, err := pickDeployment(identifier, registryManager)
	if err != nil {
		return err
	}

	// If no flags, just show current tags
	if addTag == "" && removeTag == "" {
		return showDeploymentTags(deployment)
	}

	// Add tag
	if addTag != "" {
		return addDeploymentTag(deployment, addTag, registryManager)
	}

	// Remove tag
	return removeDeploymentTag(deployment, removeTag, registryManager)
}

func showDeploymentTags(deployment *registry.DeploymentInfo) error {
	// Color styles
	titleStyle := color.New(color.FgCyan, color.Bold)
	labelStyle := color.New(color.FgWhite, color.Bold)
	addressStyle := color.New(color.FgGreen, color.Bold)
	tagStyle := color.New(color.FgCyan)

	displayName := deployment.Entry.GetDisplayName()
	fmt.Println()
	titleStyle.Printf("Deployment: %s/%s/%s\n", deployment.NetworkName, deployment.Entry.Namespace, displayName)

	labelStyle.Print("Address: ")
	addressStyle.Println(deployment.Address.Hex())

	labelStyle.Print("Tags:    ")
	if len(deployment.Entry.Tags) == 0 {
		color.New(color.Faint).Println("No tags")
	} else {
		for i, tag := range deployment.Entry.Tags {
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

func addDeploymentTag(deployment *registry.DeploymentInfo, tag string, registryManager *registry.Manager) error {
	// Check if tag already exists
	for _, existingTag := range deployment.Entry.Tags {
		if existingTag == tag {
			color.New(color.FgYellow).Printf("⚠️  Deployment already has tag '%s'\n", tag)
			return nil
		}
	}

	// Add the tag
	if err := registryManager.AddTag(deployment.Address, tag); err != nil {
		return fmt.Errorf("failed to add tag: %w", err)
	}

	// Save changes
	if err := registryManager.Save(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	// Show success
	color.New(color.FgGreen).Printf("✅ Added tag '%s' to %s/%s/%s\n",
		tag,
		deployment.NetworkName,
		deployment.Entry.Namespace,
		deployment.Entry.GetDisplayName(),
	)

	// Show all tags
	fmt.Print("\nCurrent tags: ")
	tagStyle := color.New(color.FgCyan)
	allTags := append(deployment.Entry.Tags, tag)
	for i, t := range allTags {
		if i > 0 {
			fmt.Print(", ")
		}
		tagStyle.Print(t)
	}
	fmt.Println()

	return nil
}

func removeDeploymentTag(deployment *registry.DeploymentInfo, tag string, registryManager *registry.Manager) error {
	// Check if tag exists
	found := false
	for _, existingTag := range deployment.Entry.Tags {
		if existingTag == tag {
			found = true
			break
		}
	}

	if !found {
		color.New(color.FgYellow).Printf("⚠️  Deployment doesn't have tag '%s'\n", tag)
		return nil
	}

	// Remove the tag
	if err := registryManager.RemoveTag(deployment.Address, tag); err != nil {
		return fmt.Errorf("failed to remove tag: %w", err)
	}

	// Save changes
	if err := registryManager.Save(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	// Show success
	color.New(color.FgGreen).Printf("✅ Removed tag '%s' from %s/%s/%s\n",
		tag,
		deployment.NetworkName,
		deployment.Entry.Namespace,
		deployment.Entry.GetDisplayName(),
	)

	// Show remaining tags
	fmt.Print("\nRemaining tags: ")
	if len(deployment.Entry.Tags) == 1 {
		color.New(color.Faint).Print("No tags")
	} else {
		tagStyle := color.New(color.FgCyan)
		for i, t := range deployment.Entry.Tags {
			if t != tag {
				if i > 0 {
					fmt.Print(", ")
				}
				tagStyle.Print(t)
			}
		}
	}
	fmt.Println()

	return nil
}
