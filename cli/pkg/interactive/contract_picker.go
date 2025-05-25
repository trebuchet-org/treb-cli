package interactive

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
)

// SelectContract helps users select a contract when multiple matches exist
func SelectContract(matches []contracts.ContractDiscovery, prompt string) (*contracts.ContractDiscovery, error) {
	if len(matches) == 0 {
		return nil, fmt.Errorf("no contracts found")
	}

	// If only one match, return it directly
	if len(matches) == 1 {
		return &matches[0], nil
	}

	// Multiple matches - need user to disambiguate
	discovery := contracts.NewDiscovery(".")
	options := discovery.GetFormattedOptions(matches)

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

	return &matches[index], nil
}

// ResolveContract finds and potentially disambiguates a contract by name or path
func ResolveContract(nameOrPath string) (*contracts.ContractDiscovery, error) {
	discovery := contracts.NewDiscovery(".")
	
	// Find matching contracts
	matches, err := discovery.FindContract(nameOrPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find contract: %w", err)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("contract '%s' not found", nameOrPath)
	}

	// Use picker if multiple matches
	if len(matches) > 1 {
		prompt := fmt.Sprintf("Multiple contracts named '%s' found. Select one:", nameOrPath)
		return SelectContract(matches, prompt)
	}

	return &matches[0], nil
}