package cli

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
)

// contractItem represents a selectable contract in the multi-select
type contractItem struct {
	creation models.ContractCreation
	selected bool
}

// multiSelectModel is the bubbletea model for multi-select
type multiSelectModel struct {
	items    []contractItem
	cursor   int
	selected map[int]bool
	title    string
	done     bool
}

// initialModel creates the initial model for multi-select
func initialMultiSelectModel(creations []models.ContractCreation, title string) multiSelectModel {
	items := make([]contractItem, len(creations))
	selected := make(map[int]bool)
	for i, creation := range creations {
		items[i] = contractItem{creation: creation, selected: false}
		selected[i] = false
	}
	return multiSelectModel{
		items:    items,
		cursor:   0,
		selected: selected,
		title:    title,
		done:     false,
	}
}

// Init is the initial command for bubbletea
func (m multiSelectModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m multiSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.done = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case " ":
			// Toggle selection
			m.selected[m.cursor] = !m.selected[m.cursor]
			m.items[m.cursor].selected = m.selected[m.cursor]
		case "enter":
			// Check if at least one item is selected
			hasSelection := false
			for _, selected := range m.selected {
				if selected {
					hasSelection = true
					break
				}
			}
			if hasSelection {
				m.done = true
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

// View renders the UI
func (m multiSelectModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder
	b.WriteString(color.New(color.FgCyan, color.Bold).Sprintf("%s\n\n", m.title))

	for i, item := range m.items {
		cursor := " "
		if m.cursor == i {
			cursor = color.New(color.FgCyan).Sprint("▸")
		}

		checkbox := " "
		if m.selected[i] {
			checkbox = color.New(color.FgGreen).Sprint("✓")
		} else {
			checkbox = color.New(color.FgWhite).Sprint("○")
		}

		address := color.New(color.FgWhite).Sprint(item.creation.Address)
		kind := color.New(color.FgYellow).Sprintf("(%s)", item.creation.Kind)

		b.WriteString(fmt.Sprintf("%s %s %s %s\n", cursor, checkbox, address, kind))
	}

	b.WriteString("\n")
	b.WriteString(color.New(color.FgYellow).Sprint("↑/↓: move  Space: toggle  Enter: confirm  q: quit\n"))

	return b.String()
}

// SelectContracts shows a multi-select interface and returns selected contract indices
func SelectContracts(creations []models.ContractCreation, title string) ([]int, error) {
	if len(creations) == 0 {
		return nil, fmt.Errorf("no contracts to select")
	}

	model := initialMultiSelectModel(creations, title)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("multi-select failed: %w", err)
	}

	m := finalModel.(multiSelectModel)
	if !m.done {
		return nil, fmt.Errorf("selection cancelled")
	}

	// Collect selected indices
	var selectedIndices []int
	for i, isSelected := range m.selected {
		if isSelected {
			selectedIndices = append(selectedIndices, i)
		}
	}

	if len(selectedIndices) == 0 {
		return nil, fmt.Errorf("no contracts selected")
	}

	return selectedIndices, nil
}

