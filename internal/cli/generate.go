package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// NewGenerateCmd creates the generate command group
func NewGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "gen",
		Aliases: []string{"generate"},
		Short:   "Generate deployment scripts",
		Long: `Generate deployment scripts for contracts and libraries.

This command creates template scripts using treb-sol's base contracts.
The generated scripts handle both direct deployments and common proxy patterns.`,
	}

	cmd.AddCommand(newGenerateDeployCmd())

	return cmd
}

// newGenerateDeployCmd creates the generate deploy subcommand
func newGenerateDeployCmd() *cobra.Command {
	var (
		useProxy      bool
		proxyContract string
		strategy      string
		scriptPath    string
	)

	cmd := &cobra.Command{
		Use:   "deploy <artifact>",
		Short: "Generate a deployment script for a contract or library",
		Long: `Generate a deployment script for a contract or library.

This command automatically detects whether the artifact is a library or contract
and generates the appropriate deployment script.

For contracts, you can optionally generate a proxy deployment pattern by using
the --proxy flag. If --proxy is specified without a value, an interactive
proxy selection will be shown.

Examples:
  # Library deployment
  treb gen deploy MathUtils
  treb gen deploy src/libs/StringUtils.sol:StringUtils
  
  # Contract deployment
  treb gen deploy Counter
  treb gen deploy src/Token.sol:Token
  
  # Proxy deployment (interactive proxy selection)
  treb gen deploy Counter --proxy
  
  # Proxy deployment with specific proxy
  treb gen deploy Counter --proxy --proxy-contract TransparentUpgradeableProxy
  treb gen deploy MyToken --proxy --proxy-contract src/proxies/CustomProxy.sol:CustomProxy
  
  # Custom script path
  treb gen deploy Counter --script-path script/deploy/CustomDeploy.s.sol`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get app from context
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			// Parse deployment strategy
			deployStrategy := domain.StrategyCreate3
			if strategy != "" {
				switch strings.ToUpper(strategy) {
				case "CREATE2":
					deployStrategy = domain.StrategyCreate2
				case "CREATE3":
					deployStrategy = domain.StrategyCreate3
				default:
					return fmt.Errorf("invalid deployment strategy: %s (valid: CREATE2, CREATE3)", strategy)
				}
			}

			// Build parameters
			params := usecase.GenerateScriptParams{
				ArtifactRef:   args[0],
				UseProxy:      useProxy,
				ProxyContract: proxyContract,
				Strategy:      deployStrategy,
				CustomPath:    scriptPath,
			}

			// Run use case
			result, err := app.GenerateDeploymentScript.Run(cmd.Context(), params)
			if err != nil {
				return err
			}

			app.GenerateRenderer.Render(result)
			return nil
		},
	}

	// Add flags
	cmd.Flags().BoolVar(&useProxy, "proxy", false, "Generate proxy deployment script")
	cmd.Flags().StringVar(&proxyContract, "proxy-contract", "", "Specific proxy contract to use (optional)")
	cmd.Flags().StringVar(&strategy, "strategy", "", "Deployment strategy: CREATE2 or CREATE3 (default: CREATE3)")
	cmd.Flags().StringVar(&scriptPath, "script-path", "", "Custom path for the generated script")

	return cmd
}

