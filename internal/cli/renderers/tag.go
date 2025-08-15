package renderers

import (
	"fmt"
	"sort"

	"github.com/fatih/color"
	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// TagRenderer renders tag operation results
type TagRenderer struct {
	config *config.RuntimeConfig
}

// NewTagRenderer creates a new tag renderer
func NewTagRenderer(cfg *config.RuntimeConfig) *TagRenderer {
	return &TagRenderer{config: cfg}
}

// Render displays the tag operation result
func (r *TagRenderer) Render(result *usecase.TagDeploymentResult) error {
	if result == nil {
		return fmt.Errorf("no result to render")
	}

	deployment := result.Deployment
	displayName := deployment.ContractName
	if deployment.Label != "" {
		displayName += ":" + deployment.Label
	}

	switch result.Operation {
	case "show":
		return r.renderShowTags(deployment, displayName)
	case "add":
		return r.renderAddTag(deployment, displayName, result.Tag, result.CurrentTags)
	case "remove":
		return r.renderRemoveTag(deployment, displayName, result.Tag, result.CurrentTags)
	default:
		return fmt.Errorf("unknown operation: %s", result.Operation)
	}
}

// renderShowTags displays current tags for a deployment
func (r *TagRenderer) renderShowTags(deployment *domain.Deployment, displayName string) error {
	// Color styles
	titleStyle := color.New(color.FgCyan, color.Bold)
	labelStyle := color.New(color.FgWhite, color.Bold)
	addressStyle := color.New(color.FgGreen, color.Bold)
	tagStyle := color.New(color.FgCyan)

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

// renderAddTag displays the result of adding a tag
func (r *TagRenderer) renderAddTag(deployment *domain.Deployment, displayName, tag string, currentTags []string) error {

	// Show success
	color.New(color.FgGreen).Printf("✅ Added tag '%s' to %s/%d/%s\n",
		tag,
		deployment.Namespace,
		deployment.ChainID,
		displayName,
	)

	// Show all tags
	fmt.Print("\nCurrent tags: ")
	tagStyle := color.New(color.FgCyan)

	allTags := make([]string, len(currentTags))
	copy(allTags, currentTags)
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

// renderRemoveTag displays the result of removing a tag
func (r *TagRenderer) renderRemoveTag(deployment *domain.Deployment, displayName, tag string, currentTags []string) error {

	// Show success
	color.New(color.FgGreen).Printf("✅ Removed tag '%s' from %s/%d/%s\n",
		tag,
		deployment.Namespace,
		deployment.ChainID,
		displayName,
	)

	// Show remaining tags
	fmt.Print("\nRemaining tags: ")
	if len(currentTags) == 0 {
		color.New(color.Faint).Print("No tags")
	} else {
		tagStyle := color.New(color.FgCyan)
		sortedTags := make([]string, len(currentTags))
		copy(sortedTags, currentTags)
		sort.Strings(sortedTags)

		for i, t := range sortedTags {
			if i > 0 {
				fmt.Print(", ")
			}
			tagStyle.Print(t)
		}
	}
	fmt.Println()

	return nil
}

