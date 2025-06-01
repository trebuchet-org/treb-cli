package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Version information - set by main.go
var (
	buildVersion = "dev"
	buildCommit  = "unknown"
	buildDate    = "unknown"
)

// SetVersionInfo sets the version information from main.go
func SetVersionInfo(version, commit, date string) {
	buildVersion = version
	buildCommit = commit
	buildDate = date
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show treb version",
	Long:  `Display the current version of treb CLI with build information.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("treb %s\n", buildVersion)

		// Only show additional info if not default values
		if buildCommit != "unknown" || buildDate != "unknown" {
			fmt.Println()
			if buildCommit != "unknown" {
				// Truncate commit to 7 characters for display
				shortCommit := buildCommit
				if len(shortCommit) > 7 {
					shortCommit = shortCommit[:7]
				}
				fmt.Printf("commit: %s\n", shortCommit)
			}
			if buildDate != "unknown" {
				// Format the date nicely
				fmt.Printf("built:  %s\n", formatBuildDate(buildDate))
			}
		}
	},
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

func init() {
	// Add to root command
}
