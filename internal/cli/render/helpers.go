package render

import (
	"strings"

	"github.com/fatih/color"
)

// FormatWarning formats a warning message with the warning icon
func FormatWarning(message string) string {
	// Extract just the error message part (after the last colon if it's an error chain)
	parts := strings.Split(message, ": ")
	msg := parts[len(parts)-1]

	// Convert to v1 format
	if strings.Contains(msg, "already exists") {
		// Extract tag name from "tag 'tagname' already exists"
		tagParts := strings.Split(msg, "'")
		if len(tagParts) >= 2 {
			tag := tagParts[1]
			return color.New(color.FgYellow).Sprintf("⚠️  Deployment already has tag '%s'", tag)
		}
	} else if strings.Contains(msg, "does not exist") {
		// Extract tag name from "tag 'tagname' does not exist"
		tagParts := strings.Split(msg, "'")
		if len(tagParts) >= 2 {
			tag := tagParts[1]
			return color.New(color.FgYellow).Sprintf("⚠️  Deployment doesn't have tag '%s'", tag)
		}
	}

	// Fallback for other warnings
	return color.New(color.FgYellow).Sprintf("⚠️  %s", msg)
}

// FormatError formats an error message with the error icon
func FormatError(message string) string {
	// Extract just the error message part (after the last colon if it's an error chain)
	parts := strings.Split(message, ": ")
	msg := parts[len(parts)-1]

	// Capitalize first letter
	if len(msg) > 0 {
		msg = strings.ToUpper(msg[:1]) + msg[1:]
	}

	return color.New(color.FgRed).Sprintf("❌ %s", msg)
}

// FormatSuccess formats a success message with the success icon
func FormatSuccess(message string) string {
	return color.New(color.FgGreen).Sprintf("✅ %s", message)
}
