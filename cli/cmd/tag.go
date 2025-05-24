package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/internal/registry"
)

var tagCmd = &cobra.Command{
	Use:   "tag <contract> <tag>",
	Short: "Tag a deployment with a version",
	Long: `Add version tags to deployments for tracking releases.

Examples:
  treb tag Counter v1.0.0
  treb tag 0x1234... production-release`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		identifier := args[0]
		tag := args[1]

		if err := tagDeployments(identifier, tag, false, false); err != nil {
			checkError(err)
		}
	},
}

func init() {
	// Add flags if needed in the future
}

func tagDeployments(identifier string, tag string, all bool, remove bool) error {
	// Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

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
		return fmt.Errorf("no deployment found matching '%s'", identifier)
	}

	// If not tagging all and multiple matches exist, show selection
	if !all && len(matches) > 1 {
		fmt.Printf("Multiple deployments found matching '%s':\n\n", identifier)

		// Sort matches by network, then env, then contract name
		sort.Slice(matches, func(i, j int) bool {
			if matches[i].NetworkName != matches[j].NetworkName {
				return matches[i].NetworkName < matches[j].NetworkName
			}
			if matches[i].Entry.Environment != matches[j].Entry.Environment {
				return matches[i].Entry.Environment < matches[j].Entry.Environment
			}
			return matches[i].Entry.ContractName < matches[j].Entry.ContractName
		})

		for i, match := range matches {
			displayName := match.Entry.GetDisplayName()
			fullId := fmt.Sprintf("%s/%s/%s", match.NetworkName, match.Entry.Environment, displayName)
			fmt.Printf("%d. %s\n   Address: %s\n\n", i+1, fullId, match.Address.Hex())
		}

		// Ask user to select
		fmt.Print("Select deployment (1-", len(matches), "): ")
		var selection int
		fmt.Scanln(&selection)

		if selection < 1 || selection > len(matches) {
			return fmt.Errorf("invalid selection")
		}

		matches = []*registry.DeploymentInfo{matches[selection-1]}
	}

	// Apply tag operation to selected deployments
	modified := 0
	for _, deployment := range matches {
		// Check if tag exists
		tagExists := false
		for _, existingTag := range deployment.Entry.Tags {
			if existingTag == tag {
				tagExists = true
				break
			}
		}

		if remove {
			// Remove the tag
			if tagExists {
				if err := registryManager.RemoveTag(deployment.Address, tag); err != nil {
					fmt.Printf("Failed to remove tag from %s: %v\n", deployment.Address.Hex(), err)
					continue
				}
				modified++
				fmt.Printf("Removed tag '%s' from %s\n", tag, deployment.Entry.GetDisplayName())
			} else {
				fmt.Printf("%s doesn't have tag '%s'\n", deployment.Entry.GetDisplayName(), tag)
			}
		} else {
			// Add the tag
			if !tagExists {
				if err := registryManager.AddTag(deployment.Address, tag); err != nil {
					fmt.Printf("Failed to tag %s: %v\n", deployment.Address.Hex(), err)
					continue
				}
				modified++
				fmt.Printf("Tagged %s with '%s'\n", deployment.Entry.GetDisplayName(), tag)
			} else {
				fmt.Printf("%s already has tag '%s'\n", deployment.Entry.GetDisplayName(), tag)
			}
		}
	}

	if modified > 0 {
		if err := registryManager.Save(); err != nil {
			return fmt.Errorf("failed to save registry: %w", err)
		}
		action := "tagged"
		if remove {
			action = "removed"
		}
		fmt.Printf("\nSuccessfully %s %d deployment(s)\n", action, modified)
	}

	return nil
}