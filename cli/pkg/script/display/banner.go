package display

import (
	"fmt"
	"strings"
)

// PrintDeploymentBanner prints a banner for deployment operations
func PrintDeploymentBanner(scriptName, network, namespace string, dryRun bool) {
	fmt.Println()
	fmt.Printf("%s🚀 Running Deployment Script%s\n", ColorBold, ColorReset)
	fmt.Printf("%s%s%s\n", ColorGray, strings.Repeat("─", 50), ColorReset)
	fmt.Printf("  Script:    %s%s%s\n", ColorCyan, scriptName, ColorReset)
	fmt.Printf("  Network:   %s%s%s\n", ColorBlue, network, ColorReset)
	fmt.Printf("  Namespace: %s%s%s\n", ColorPurple, namespace, ColorReset)

	if dryRun {
		fmt.Printf("  Mode:      %sDRY RUN%s\n", ColorYellow, ColorReset)
	} else {
		fmt.Printf("  Mode:      %sLIVE%s\n", ColorGreen, ColorReset)
	}

	fmt.Printf("%s%s%s\n", ColorGray, strings.Repeat("─", 50), ColorReset)
	fmt.Println()
}
