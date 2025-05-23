package interactive

import (
	"fmt"
	"strconv"
	"strings"
)

// Selector handles interactive selection with fzf-like interface
type Selector struct{}

// NewSelector creates a new interactive selector
func NewSelector() *Selector {
	return &Selector{}
}

// SelectOption provides an fzf-like selection interface
func (s *Selector) SelectOption(prompt string, options []string, defaultIndex int) (string, int, error) {
	if len(options) == 0 {
		return "", -1, fmt.Errorf("no options provided")
	}

	currentIndex := defaultIndex
	if currentIndex < 0 || currentIndex >= len(options) {
		currentIndex = 0
	}

	for {
		// Clear screen and show options
		fmt.Print("\033[2J\033[H") // Clear screen and move cursor to top
		fmt.Printf("%s\n", prompt)
		fmt.Println("Use ↑/↓ arrow keys (k/j) to navigate, Enter to select, 'q' to quit")
		fmt.Println("Or type a number to select directly:")
		fmt.Println()

		// Display options with current selection highlighted
		for i, option := range options {
			if i == currentIndex {
				fmt.Printf("  \033[7m► %d) %s\033[0m\n", i+1, option) // Highlighted
			} else {
				fmt.Printf("    %d) %s\n", i+1, option)
			}
		}

		fmt.Printf("\nSelection [%d]: ", currentIndex+1)

		// Read single character or line input
		var input string
		if _, err := fmt.Scanln(&input); err != nil {
			input = ""
		}
		input = strings.TrimSpace(input)

		switch input {
		case "":
			// Enter pressed - select current option
			return options[currentIndex], currentIndex, nil
		case "q", "quit":
			return "", -1, fmt.Errorf("selection cancelled")
		case "k", "up":
			// Move up
			if currentIndex > 0 {
				currentIndex--
			} else {
				currentIndex = len(options) - 1 // Wrap to bottom
			}
		case "j", "down":
			// Move down
			if currentIndex < len(options)-1 {
				currentIndex++
			} else {
				currentIndex = 0 // Wrap to top
			}
		default:
			// Try to parse as number
			if num, err := strconv.Atoi(input); err == nil && num >= 1 && num <= len(options) {
				return options[num-1], num-1, nil
			}
			// Invalid input, continue loop
		}
	}
}

// SimpleSelect provides a simple numbered selection (fallback for environments without terminal control)
func (s *Selector) SimpleSelect(prompt string, options []string, defaultIndex int) (string, int, error) {
	if len(options) == 0 {
		return "", -1, fmt.Errorf("no options provided")
	}

	fmt.Printf("%s\n", prompt)
	for i, option := range options {
		marker := " "
		if i == defaultIndex {
			marker = "*"
		}
		fmt.Printf("  %s %d) %s\n", marker, i+1, option)
	}

	fmt.Printf("Enter choice [%d]: ", defaultIndex+1)
	
	var input string
	if _, err := fmt.Scanln(&input); err != nil {
		// If no input, use default
		input = ""
	}
	input = strings.TrimSpace(input)
	
	if input == "" {
		return options[defaultIndex], defaultIndex, nil
	}

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(options) {
		return "", -1, fmt.Errorf("invalid choice: %s", input)
	}

	return options[choice-1], choice-1, nil
}