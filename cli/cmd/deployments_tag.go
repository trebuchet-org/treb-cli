package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/trebuchet-org/treb-cli/cli/internal/registry"
)

func tagDeployments(identifier, tag string, all, remove bool) error {
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

	// If multiple matches and not using --all, show selection
	if len(matches) > 1 && !all {
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
		fmt.Print("Select deployment (1-", len(matches), ") or use --all to tag all: ")
		var selection int
		fmt.Scanln(&selection)

		if selection < 1 || selection > len(matches) {
			return fmt.Errorf("invalid selection")
		}

		matches = []*registry.DeploymentInfo{matches[selection-1]}
	}

	// Tag or remove tags from the selected deployments
	var modified int
	var action string
	if remove {
		action = "removed"
	} else {
		action = "tagged"
	}

	for _, match := range matches {
		// Check if tag exists
		tagExists := false
		for _, existingTag := range match.Entry.Tags {
			if existingTag == tag {
				tagExists = true
				break
			}
		}

		if remove {
			// Remove the tag
			if tagExists {
				if err := registryManager.RemoveTag(match.Address, tag); err != nil {
					fmt.Printf("Failed to remove tag from %s: %v\n", match.Address.Hex(), err)
					continue
				}
				modified++
				fmt.Printf("Removed tag '%s' from %s\n", tag, match.Entry.GetDisplayName())
			} else {
				fmt.Printf("%s doesn't have tag '%s'\n", match.Entry.GetDisplayName(), tag)
			}
		} else {
			// Add the tag
			if !tagExists {
				if err := registryManager.AddTag(match.Address, tag); err != nil {
					fmt.Printf("Failed to tag %s: %v\n", match.Address.Hex(), err)
					continue
				}
				modified++
				fmt.Printf("Tagged %s with '%s'\n", match.Entry.GetDisplayName(), tag)
			} else {
				fmt.Printf("%s already has tag '%s'\n", match.Entry.GetDisplayName(), tag)
			}
		}
	}

	if modified > 0 {
		if err := registryManager.Save(); err != nil {
			return fmt.Errorf("failed to save registry: %w", err)
		}
		fmt.Printf("\nSuccessfully %s %d deployment(s)\n", action, modified)
	}

	return nil
}