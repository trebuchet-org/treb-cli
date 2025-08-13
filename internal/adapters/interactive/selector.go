package interactive

import (
	"context"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/sahilm/fuzzy"
	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// SelectorAdapter handles interactive selection
type SelectorAdapter struct {
	config *config.RuntimeConfig
}

// NewSelectorAdapter creates a new selector adapter
func NewSelectorAdapter(cfg *config.RuntimeConfig) (*SelectorAdapter, error) {
	return &SelectorAdapter{config: cfg}, nil
}

// SelectContract selects a contract from a list
func (s *SelectorAdapter) SelectContract(ctx context.Context, contracts []*domain.ContractInfo, prompt string) (*domain.ContractInfo, error) {
	// In non-interactive mode, we can't select
	if s.config.NonInteractive {
		return nil, fmt.Errorf("interactive selection not available in non-interactive mode")
	}

	if len(contracts) == 0 {
		return nil, fmt.Errorf("no contracts provided for selection")
	}

	// If only one match, return it directly
	if len(contracts) == 1 {
		return contracts[0], nil
	}

	// Multiple matches - need user to disambiguate
	options := formatContractOptions(contracts)

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "▸ {{ . | cyan }}",
		Inactive: "  {{ . | faint }}",
		Selected: "✓ {{ . | green }}",
		Help:     color.New(color.FgYellow).Sprint("Use arrow keys to navigate, Enter to select"),
	}

	promptSelect := promptui.Select{
		Label:             prompt,
		Items:             options,
		Templates:         templates,
		Size:              10,
		StartInSearchMode: true,
		Searcher:          createFuzzySearchFunc(options),
	}

	index, _, err := promptSelect.Run()
	if err != nil {
		return nil, fmt.Errorf("selection cancelled: %w", err)
	}

	return contracts[index], nil
}

// formatContractOptions creates display strings for contract selection
func formatContractOptions(contracts []*domain.ContractInfo) []string {
	options := make([]string, len(contracts))
	for i, contract := range contracts {
		// Format as "ContractName (path/to/file.sol)"
		relPath := contract.Path
		relPath = strings.TrimPrefix(relPath, "src/")

		// Add library/interface/abstract indicators
		var indicators []string
		if contract.IsLibrary {
			indicators = append(indicators, "library")
		}
		if contract.IsInterface {
			indicators = append(indicators, "interface")
		}
		if contract.IsAbstract {
			indicators = append(indicators, "abstract")
		}

		contractName := color.New(color.FgWhite, color.Bold).Sprint(contract.Name)
		pathStr := color.New(color.FgBlue).Sprint(relPath)

		if len(indicators) > 0 {
			indicatorStr := color.New(color.FgYellow).Sprintf("[%s]", strings.Join(indicators, ", "))
			options[i] = fmt.Sprintf("%s %s (%s)", contractName, indicatorStr, pathStr)
		} else {
			options[i] = fmt.Sprintf("%s (%s)", contractName, pathStr)
		}
	}
	return options
}

// createFuzzySearchFunc creates a fuzzy search function for promptui
func createFuzzySearchFunc(items []string) func(input string, index int) bool {
	return func(input string, index int) bool {
		// Empty search shows all items
		if input == "" {
			return true
		}

		// Convert to lowercase for case-insensitive search
		input = strings.ToLower(input)
		item := strings.ToLower(items[index])

		// First try simple substring match
		if strings.Contains(item, input) {
			return true
		}

		// Then try fuzzy match
		pattern := fuzzy.Find(input, []string{item})
		return len(pattern) > 0
	}
}

// Ensure the adapter implements the interface
var _ usecase.InteractiveSelector = (*SelectorAdapter)(nil)