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

// ResolveContract finds and potentially disambiguates a contract by name or path
func ResolveContract(nameOrPath string, filter contracts.QueryFilter) (*contracts.ContractInfo, error) {
	// Use the global indexer
	indexer, err := contracts.GetGlobalIndexer(".")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize contract indexer: %w", err)
	}

	// First try to get by exact key (path:name format)
	if strings.Contains(nameOrPath, ":") {
		if contract, err := indexer.GetContract(nameOrPath); err == nil {
			// Check if it's deployable
			if !contract.IsLibrary && !contract.IsInterface && !contract.IsAbstract {
				return contract, nil
			}
		}
	}

	// Find matching contracts using deployable filter
	matches := indexer.FindContractByName(nameOrPath, filter)

	// If no exact matches, try searching for partial matches
	if len(matches) == 0 {
		matches = indexer.SearchContracts(nameOrPath)
		// Apply the filter manually since SearchContracts doesn't use filters
		var filteredMatches []*contracts.ContractInfo
		for _, contract := range matches {
			if !filter.IncludeLibraries && contract.IsLibrary {
				continue
			}
			if !filter.IncludeInterface && contract.IsInterface {
				continue
			}
			if !filter.IncludeAbstract && contract.IsAbstract {
				continue
			}
			filteredMatches = append(filteredMatches, contract)
		}
		matches = filteredMatches
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("contract '%s' not found", nameOrPath)
	}

	// Use picker if multiple matches
	if len(matches) > 1 {
		prompt := fmt.Sprintf("Multiple contracts matching '%s' found. Select one:", nameOrPath)
		return SelectContract(matches, prompt)
	}

	return matches[0], nil
}

// formatContractOptions creates display strings for contract selection
func formatContractOptions(contracts []*contracts.ContractInfo) []string {
	options := make([]string, len(contracts))
	for i, contract := range contracts {
		// Format as "ContractName (path/to/file.sol)"
		relPath := contract.Path
		if strings.HasPrefix(relPath, "src/") {
			relPath = strings.TrimPrefix(relPath, "src/")
		}

		options[i] = fmt.Sprintf("%s (%s)",
			color.New(color.FgWhite, color.Bold).Sprint(contract.Name),
			color.New(color.FgBlue).Sprint(relPath),
		)
	}
	return options
}
