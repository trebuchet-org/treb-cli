package interactive

import (
	"fmt"

	"github.com/manifoldco/promptui"
)

// Selector handles interactive selection with fzf-like interface
type Selector struct{}

// NewSelector creates a new interactive selector
func NewSelector() *Selector {
	return &Selector{}
}

// SelectOption provides an interactive selection interface using promptui
func (s *Selector) SelectOption(prompt string, options []string, defaultIndex int) (string, int, error) {
	if len(options) == 0 {
		return "", -1, fmt.Errorf("no options provided")
	}

	// Ensure default index is valid
	if defaultIndex < 0 || defaultIndex >= len(options) {
		defaultIndex = 0
	}

	promptSelect := promptui.Select{
		Label:     prompt,
		Items:     options,
		CursorPos: defaultIndex,
		Size:      10,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}?",
			Active:   "\U0001F449 {{ . | cyan }}",
			Inactive: "  {{ . | faint }}",
			Selected: "\U0001F44D {{ . | green }}",
		},
	}

	selectedIndex, selectedValue, err := promptSelect.Run()
	if err != nil {
		return "", -1, fmt.Errorf("selection cancelled: %w", err)
	}

	return selectedValue, selectedIndex, nil
}

// SimpleSelect provides the same interface as SelectOption (both use promptui now)
func (s *Selector) SimpleSelect(prompt string, options []string, defaultIndex int) (string, int, error) {
	// Just use the same implementation as SelectOption since promptui handles terminal detection
	return s.SelectOption(prompt, options, defaultIndex)
}

// PromptString provides an interactive text input prompt
func (s *Selector) PromptString(prompt string, defaultValue string) (string, error) {
	promptInput := promptui.Prompt{
		Label:   prompt,
		Default: defaultValue,
		Templates: &promptui.PromptTemplates{
			Prompt:  "{{ . | bold }}{{ if .Default }} ({{ .Default }}){{ end }}: ",
			Valid:   "{{ . | green }} ✓ ",
			Invalid: "{{ . | red }} ✗ ",
			Success: "{{ . | bold }}{{ if .Default }} ({{ .Default }}){{ end }}: {{ . | faint }}",
		},
	}

	result, err := promptInput.Run()
	if err != nil {
		return "", fmt.Errorf("input cancelled: %w", err)
	}

	return result, nil
}

// PromptConfirm provides a yes/no confirmation prompt
func (s *Selector) PromptConfirm(prompt string, defaultValue bool) (bool, error) {
	label := prompt
	var defaultString string
	if defaultValue {
		defaultString = "y"
	} else {
		defaultString = "n"
	}

	promptConfirm := promptui.Prompt{
		Label:     label,
		IsConfirm: true,
		Default:   defaultString,
	}

	result, err := promptConfirm.Run()

	if err != nil {
		// If user pressed Enter without input, use default
		if err == promptui.ErrAbort {
			return defaultValue, nil
		}
		return false, fmt.Errorf("confirmation cancelled: %w", err)
	}

	if result == "" {
		return defaultValue, nil
	} else {
		return result == "y", nil
	}
}
