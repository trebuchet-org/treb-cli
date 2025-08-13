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
The generated scripts handle both direct deployments and proxy patterns.`,
	}

	cmd.AddCommand(newGenerateDeployCmd())
	// Future: cmd.AddCommand(newGenerateProxyCmd())

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
				ArtifactPath:  args[0],
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

			// Display result
			fmt.Printf("\n✅ Generated deployment script: %s\n", result.ScriptPath)
			for _, instruction := range result.Instructions {
				fmt.Println(instruction)
			}

			return nil
		},
	}

	// Add flags
	cmd.Flags().BoolVar(&useProxy, "proxy", false, "Generate proxy deployment script")
	cmd.Flags().StringVar(&proxyContract, "proxy-contract", "", "Specific proxy contract to use (optional)")
	cmd.Flags().StringVar(&strategy, "strategy", "", "Deployment strategy: CREATE2 or CREATE3 (default: CREATE3)")
	cmd.Flags().StringVar(&scriptPath, "script-path", "", "Custom path for the generated script")

	// Override the help template to only show non-interactive flag
	cmd.SetUsageTemplate(`Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}

Global Flags:
      --non-interactive   Disable interactive prompts{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`)

	return cmd
}