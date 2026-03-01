package render

import (
	"fmt"
	"io"
	"sort"

	"github.com/fatih/color"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// TagRenderer renders tag operation results
type TagRenderer struct {
	out io.Writer
}

// NewTagRenderer creates a new tag renderer
func NewTagRenderer(out io.Writer) *TagRenderer {
	return &TagRenderer{out: out}
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
func (r *TagRenderer) renderShowTags(deployment *models.Deployment, displayName string) error {
	// Color styles
	titleStyle := color.New(color.FgCyan, color.Bold)
	labelStyle := color.New(color.FgWhite, color.Bold)
	addressStyle := color.New(color.FgGreen, color.Bold)
	tagStyle := color.New(color.FgCyan)

	fmt.Fprintln(r.out)
	titleStyle.Fprintf(r.out, "Deployment: %s/%d/%s\n", deployment.Namespace, deployment.ChainID, displayName)

	labelStyle.Fprint(r.out, "Address: ")
	addressStyle.Fprintln(r.out, deployment.Address)

	labelStyle.Fprint(r.out, "Tags:    ")
	if len(deployment.Tags) == 0 {
		color.New(color.Faint).Fprintln(r.out, "No tags")
	} else {
		// Sort tags for consistent display
		sortedTags := make([]string, len(deployment.Tags))
		copy(sortedTags, deployment.Tags)
		sort.Strings(sortedTags)

		for i, tag := range sortedTags {
			if i > 0 {
				fmt.Fprint(r.out, ", ")
			}
			tagStyle.Fprint(r.out, tag)
		}
		fmt.Fprintln(r.out)
	}
	fmt.Fprintln(r.out)

	return nil
}

// renderAddTag displays the result of adding a tag
func (r *TagRenderer) renderAddTag(deployment *models.Deployment, displayName, tag string, currentTags []string) error {

	// Show success
	color.New(color.FgGreen).Fprintf(r.out, "✅ Added tag '%s' to %s/%d/%s\n",
		tag,
		deployment.Namespace,
		deployment.ChainID,
		displayName,
	)

	// Show all tags
	fmt.Fprint(r.out, "\nCurrent tags: ")
	tagStyle := color.New(color.FgCyan)

	allTags := make([]string, len(currentTags))
	copy(allTags, currentTags)
	sort.Strings(allTags)

	for i, t := range allTags {
		if i > 0 {
			fmt.Fprint(r.out, ", ")
		}
		tagStyle.Fprint(r.out, t)
	}
	fmt.Fprintln(r.out)

	return nil
}

// renderRemoveTag displays the result of removing a tag
func (r *TagRenderer) renderRemoveTag(deployment *models.Deployment, displayName, tag string, currentTags []string) error {

	// Show success
	color.New(color.FgGreen).Fprintf(r.out, "✅ Removed tag '%s' from %s/%d/%s\n",
		tag,
		deployment.Namespace,
		deployment.ChainID,
		displayName,
	)

	// Show remaining tags
	fmt.Fprint(r.out, "\nRemaining tags: ")
	if len(currentTags) == 0 {
		color.New(color.Faint).Fprint(r.out, "No tags")
	} else {
		tagStyle := color.New(color.FgCyan)
		sortedTags := make([]string, len(currentTags))
		copy(sortedTags, currentTags)
		sort.Strings(sortedTags)

		for i, t := range sortedTags {
			if i > 0 {
				fmt.Fprint(r.out, ", ")
			}
			tagStyle.Fprint(r.out, t)
		}
	}
	fmt.Fprintln(r.out)

	return nil
}
