package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/internal/config"
)

// NewVersionCmd creates the version command
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number of treb",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("treb %s\n", config.Version)

			// Only show additional info if not default values
			if config.Commit != "unknown" || config.Date != "unknown" {
				fmt.Println()
				if config.Commit != "unknown" {
					// Truncate commit to 7 characters for display
					shortCommit := config.Commit
					if len(shortCommit) > 7 {
						shortCommit = shortCommit[:7]
					}
					fmt.Printf("commit: %s\n", shortCommit)
				}
				if config.Date != "unknown" {
					// Format the date nicely
					fmt.Printf("built:  %s\n", formatBuildDate(config.Date))
				}
			}
		},
	}
}

// formatBuildDate formats the ISO 8601 date to a more readable format
func formatBuildDate(date string) string {
	// If it looks like ISO 8601 (2025-01-26T15:04:05Z), format it nicely
	if strings.Contains(date, "T") && strings.HasSuffix(date, "Z") {
		// Convert 2025-01-26T15:04:05Z to "2025-01-26 15:04:05 UTC"
		parts := strings.Split(date, "T")
		if len(parts) == 2 {
			datePart := parts[0]
			timePart := strings.TrimSuffix(parts[1], "Z")
			return fmt.Sprintf("%s %s UTC", datePart, timePart)
		}
	}
	return date
}
