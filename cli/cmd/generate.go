package cmd

import (
	"fmt"
	"strings"

	"github.com/trebuchet-org/treb-cli/cli/pkg/interactive"
	"github.com/spf13/cobra"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate [type]",
	Short: "Generate deployment scripts and other resources",
	Long: `Interactive generation of deployment scripts, migrations, and other resources.

Available generation types:
  deploy     - Generate deployment scripts for contracts
  migration  - Generate migration scripts (coming soon)
  upgrade    - Generate upgrade scripts (coming soon)

Examples:
  treb generate deploy
  treb generate migration
  treb generate upgrade
  treb generate  # Interactive type selection`,
	Args: cobra.MaximumNArgs(1),
	RunE: runGenerate,
}

func init() {
	rootCmd.AddCommand(generateCmd)
}

func runGenerate(cmd *cobra.Command, args []string) error {
	generator := interactive.NewGenerator(".")
	
	var generationType interactive.GenerateType
	
	// If no type specified, prompt for selection
	if len(args) == 0 {
		availableTypes := interactive.GetAvailableTypes()
		typeNames := make([]string, len(availableTypes))
		for i, t := range availableTypes {
			typeNames[i] = string(t)
		}
		
		selector := interactive.NewSelector()
		selectedType, _, err := selector.SimpleSelect("What would you like to generate?", typeNames, 0)
		if err != nil {
			return fmt.Errorf("type selection failed: %w", err)
		}
		
		generationType = interactive.GenerateType(selectedType)
	} else {
		// Validate provided type
		requestedType := strings.ToLower(args[0])
		switch requestedType {
		case "deploy", "deployment":
			generationType = interactive.GenerateTypeDeploy
		case "migration", "migrate":
			generationType = interactive.GenerateTypeMigration
		case "upgrade":
			generationType = interactive.GenerateTypeUpgrade
		default:
			return fmt.Errorf("unknown generation type: %s. Available types: deploy, migration, upgrade", args[0])
		}
	}
	
	// Execute the appropriate generator
	switch generationType {
	case interactive.GenerateTypeDeploy:
		return generator.GenerateDeployScript()
	case interactive.GenerateTypeMigration:
		return generator.GenerateMigrationScript()
	case interactive.GenerateTypeUpgrade:
		return generator.GenerateUpgradeScript()
	default:
		return fmt.Errorf("unsupported generation type: %s", generationType)
	}
}

