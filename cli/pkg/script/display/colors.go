package display

import (
	"fmt"
)

// Color constants for consistent formatting
const (
	ColorReset  = "\033[0m"
	ColorBold   = "\033[1m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[90m"
	ColorWhite  = "\033[97m"
)

// PrintSuccessMessage prints a success message in green
func PrintSuccessMessage(message string) {
	fmt.Printf("%s✓ %s%s\n", ColorGreen, message, ColorReset)
}

// PrintErrorMessage prints an error message in red
func PrintErrorMessage(message string) {
	fmt.Printf("%s✗ %s%s\n", ColorRed, message, ColorReset)
}

// PrintWarningMessage prints a warning message in yellow
func PrintWarningMessage(message string) {
	fmt.Printf("%s⚠ %s%s\n", ColorYellow, message, ColorReset)
}

// PrintInfoMessage prints an info message in blue
func PrintInfoMessage(message string) {
	fmt.Printf("%sℹ %s%s\n", ColorBlue, message, ColorReset)
}