package interactive

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
)

// SelectContract helps users select a contract when multiple matches exist
func SelectContract(matches []*contracts.ContractInfo, prompt string) (*contracts.ContractInfo, error) {
	if len(matches) == 0 {
		return nil, fmt.Errorf("no contracts found")
	}

	// If only one match, return it directly
	if len(matches) == 1 {
		return matches[0], nil
	}

	// Multiple matches - need user to disambiguate
	options := formatContractOptions(matches)

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

// formatContractOptions creates display strings for contract selection
func formatContractOptions(contracts []*contracts.ContractInfo) []string {
	options := make([]string, len(contracts))
	for i, contract := range contracts {
		// Format as "ContractName (path/to/file.sol)"
		relPath := contract.Path
		relPath = strings.TrimPrefix(relPath, "src/")

		options[i] = fmt.Sprintf("%s (%s)",
			color.New(color.FgWhite, color.Bold).Sprint(contract.Name),
			color.New(color.FgBlue).Sprint(relPath),
		)
	}
	return options
}
