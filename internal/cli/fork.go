package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/internal/cli/render"
	cfg "github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// NewForkCmd creates the fork command group with subcommands
func NewForkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fork",
		Short: "Manage network fork mode",
		Long:  `Fork mode lets you test deployment scripts against local forks of live networks with snapshot/revert workflow.`,
	}

	cmd.AddCommand(newForkEnterCmd())
	cmd.AddCommand(newForkExitCmd())
	cmd.AddCommand(newForkRevertCmd())
	cmd.AddCommand(newForkRestartCmd())
	cmd.AddCommand(newForkStatusCmd())
	cmd.AddCommand(newForkHistoryCmd())

	return cmd
}

// newForkEnterCmd creates the fork enter subcommand
func newForkEnterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enter <network>",
		Short: "Enter fork mode for a network",
		Long: `Start a local Anvil fork of the specified network, backup registry files,
and prepare the environment for fork-mode testing.

The network's RPC endpoint in foundry.toml must use an environment variable
(e.g., ${SEPOLIA_RPC_URL}) so that treb can override it for the fork.`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE:         runForkEnter,
	}

	return cmd
}

func runForkEnter(cmd *cobra.Command, args []string) error {
	app, err := getApp(cmd)
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	network := args[0]
	projectRoot := app.Config.ProjectRoot

	// Check raw RPC endpoint format (before env var expansion)
	rawValue, err := cfg.LoadRawRPCEndpoint(projectRoot, network)
	if err != nil {
		return fmt.Errorf("failed to read RPC endpoint for '%s': %w", network, err)
	}

	envVarName, isEnvVar := cfg.DetectEnvVar(rawValue)
	if !isEnvVar {
		// RPC endpoint is hardcoded - needs migration for fork to work
		nonInteractive, _ := cmd.Flags().GetBool("non-interactive")

		if nonInteractive {
			// Auto-migrate in non-interactive mode
			fmt.Fprintf(os.Stderr, "Migrating hardcoded RPC endpoint for '%s' to environment variable...\n", network)
			if err := cfg.MigrateRPCEndpoint(projectRoot, network, rawValue); err != nil {
				return fmt.Errorf("failed to migrate RPC endpoint: %w", err)
			}
			envVarName = cfg.GenerateEnvVarName(network)
			fmt.Fprintf(os.Stderr, "Migrated: foundry.toml now uses ${%s}, value appended to .env\n", envVarName)
		} else {
			// Prompt user in interactive mode
			fmt.Fprintf(os.Stderr, "The RPC endpoint for '%s' is a hardcoded URL in foundry.toml.\n", network)
			fmt.Fprintf(os.Stderr, "Fork mode requires an environment variable reference (e.g., ${%s}).\n", cfg.GenerateEnvVarName(network))
			fmt.Fprintf(os.Stderr, "\nWould you like to migrate it? This will:\n")
			fmt.Fprintf(os.Stderr, "  1. Update foundry.toml to use ${%s}\n", cfg.GenerateEnvVarName(network))
			fmt.Fprintf(os.Stderr, "  2. Add %s=%s to .env\n\n", cfg.GenerateEnvVarName(network), rawValue)
			fmt.Fprintf(os.Stderr, "Migrate? [y/N] ")

			var answer string
			if _, err := fmt.Scanln(&answer); err != nil || (answer != "y" && answer != "Y") {
				return fmt.Errorf("fork mode requires environment variable RPC endpoints. Aborting")
			}

			if err := cfg.MigrateRPCEndpoint(projectRoot, network, rawValue); err != nil {
				return fmt.Errorf("failed to migrate RPC endpoint: %w", err)
			}
			envVarName = cfg.GenerateEnvVarName(network)
			fmt.Fprintf(os.Stderr, "Migrated successfully.\n\n")
		}
	}

	// Resolve network to get RPC URL and chain ID
	resolvedNetwork, err := app.NetworkResolver.ResolveNetwork(ctx, network)
	if err != nil {
		return fmt.Errorf("failed to resolve network '%s': %w", network, err)
	}

	// Execute the use case
	params := usecase.EnterForkParams{
		Network:    network,
		RPCURL:     resolvedNetwork.RPCURL,
		ChainID:    resolvedNetwork.ChainID,
		EnvVarName: envVarName,
	}

	result, err := app.EnterFork.Execute(ctx, params)
	if err != nil {
		return err
	}

	// Render result
	renderer := render.NewForkRenderer()
	return renderer.RenderEnter(result)
}

// newForkExitCmd creates the fork exit subcommand
func newForkExitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exit [network]",
		Short: "Exit fork mode for a network",
		Long: `Stop the forked Anvil instance, restore registry files to pre-fork state,
and clean up all fork state files.

If no network is specified, uses the currently configured network.
Use --all to exit all active forks.`,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE:         runForkExit,
	}

	cmd.Flags().Bool("all", false, "Exit all active forks")

	return cmd
}

func runForkExit(cmd *cobra.Command, args []string) error {
	app, err := getApp(cmd)
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	allFlag, _ := cmd.Flags().GetBool("all")

	network := ""
	if len(args) > 0 {
		network = args[0]
	} else if !allFlag {
		// Use current configured network
		if app.Config.Network != nil {
			network = app.Config.Network.Name
		}
	}

	params := usecase.ExitForkParams{
		Network: network,
		All:     allFlag,
	}

	result, err := app.ExitFork.Execute(ctx, params)
	if err != nil {
		return err
	}

	renderer := render.NewForkRenderer()
	return renderer.RenderExit(result)
}

// newForkRevertCmd creates the fork revert subcommand
func newForkRevertCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revert [network]",
		Short: "Revert the last treb run on a fork",
		Long: `Undo the last treb run by restoring EVM state and registry files from the
most recent snapshot.

Use --all to revert all runs and restore to the initial fork state.
If no network is specified, uses the currently configured network.`,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE:         runForkRevert,
	}

	cmd.Flags().Bool("all", false, "Revert all runs to initial fork state")

	return cmd
}

func runForkRevert(cmd *cobra.Command, args []string) error {
	app, err := getApp(cmd)
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	allFlag, _ := cmd.Flags().GetBool("all")

	network := ""
	if len(args) > 0 {
		network = args[0]
	} else {
		// Use current configured network
		if app.Config.Network != nil {
			network = app.Config.Network.Name
		}
	}

	params := usecase.RevertForkParams{
		Network: network,
		All:     allFlag,
	}

	result, err := app.RevertFork.Execute(ctx, params)
	if err != nil {
		return err
	}

	renderer := render.NewForkRenderer()
	return renderer.RenderRevert(result)
}

// newForkRestartCmd creates the fork restart subcommand
func newForkRestartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restart [network]",
		Short: "Restart a crashed fork",
		Long: `Restart a fork whose Anvil process has crashed. This will:
1. Stop the dead process (if needed)
2. Restore registry files to the initial fork state
3. Start a fresh fork from the original RPC
4. Re-run SetupFork script (if configured)
5. Take a new initial snapshot

If no network is specified, uses the currently configured network.`,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE:         runForkRestart,
	}

	return cmd
}

func runForkRestart(cmd *cobra.Command, args []string) error {
	app, err := getApp(cmd)
	if err != nil {
		return err
	}

	ctx := cmd.Context()

	network := ""
	if len(args) > 0 {
		network = args[0]
	} else {
		// Use current configured network
		if app.Config.Network != nil {
			network = app.Config.Network.Name
		}
	}

	params := usecase.RestartForkParams{
		Network: network,
	}

	result, err := app.RestartFork.Execute(ctx, params)
	if err != nil {
		return err
	}

	renderer := render.NewForkRenderer()
	return renderer.RenderRestart(result)
}

// newForkStatusCmd creates the fork status subcommand
func newForkStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of all active forks",
		Long: `Show the state of all active forks including network, chain ID, fork URL,
anvil health, uptime, snapshot count, and fork-added deployment count.`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE:         runForkStatus,
	}

	return cmd
}

func runForkStatus(cmd *cobra.Command, _ []string) error {
	app, err := getApp(cmd)
	if err != nil {
		return err
	}

	ctx := cmd.Context()

	result, err := app.ForkStatus.Execute(ctx)
	if err != nil {
		return err
	}

	renderer := render.NewForkRenderer()
	return renderer.RenderStatus(result)
}

// newForkHistoryCmd creates the fork history subcommand
func newForkHistoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history [network]",
		Short: "Show command history for a fork",
		Long: `Show chronological list of commands run against a fork and their snapshot points.

If no network is specified, uses the currently configured network.`,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE:         runForkHistory,
	}

	return cmd
}

func runForkHistory(cmd *cobra.Command, args []string) error {
	app, err := getApp(cmd)
	if err != nil {
		return err
	}

	ctx := cmd.Context()

	network := ""
	if len(args) > 0 {
		network = args[0]
	}

	params := usecase.ForkHistoryParams{
		Network: network,
	}

	result, err := app.ForkHistory.Execute(ctx, params)
	if err != nil {
		return err
	}

	renderer := render.NewForkRenderer()
	return renderer.RenderHistory(result)
}
