package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/internal/cli/render"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// NewDevCmd creates the dev command with subcommands
func NewDevCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Development utilities",
		Long:  `Development utilities for troubleshooting treb configuration and environment.`,
	}

	// Add anvil subcommand
	cmd.AddCommand(newDevAnvilCmd())

	return cmd
}

// newDevAnvilCmd creates the anvil management command
func newDevAnvilCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "anvil",
		Short: "Manage local anvil node",
		Long:  `Manage a local anvil node for testing deployments with CreateX factory automatically deployed.`,
	}

	// Add subcommands
	cmd.AddCommand(newDevAnvilStartCmd())
	cmd.AddCommand(newDevAnvilStopCmd())
	cmd.AddCommand(newDevAnvilRestartCmd())
	cmd.AddCommand(newDevAnvilStatusCmd())
	cmd.AddCommand(newDevAnvilLogsCmd())

	return cmd
}

// anvilFlags holds common flags for anvil commands
type anvilFlags struct {
	name    string
	port    string
	chainID string
}

// addAnvilFlags adds common flags to an anvil command
func addAnvilFlags(cmd *cobra.Command, flags *anvilFlags) {
	cmd.Flags().StringVar(&flags.name, "name", "anvil0", "Instance name (e.g. anvil0, anvil1)")
	cmd.Flags().StringVar(&flags.port, "port", "8545", "RPC port to bind")
	cmd.Flags().StringVar(&flags.chainID, "chain-id", "", "Chain ID to use for the instance (optional)")
}

// newDevAnvilStartCmd creates the start command
func newDevAnvilStartCmd() *cobra.Command {
	flags := &anvilFlags{}
	
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start local anvil node",
		Long:  `Start a local anvil node with CreateX factory deployed. Fails if already running.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnvilCommand(cmd, "start", flags)
		},
	}

	addAnvilFlags(cmd, flags)
	return cmd
}

// newDevAnvilStopCmd creates the stop command
func newDevAnvilStopCmd() *cobra.Command {
	flags := &anvilFlags{}
	
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop local anvil node",
		Long:  `Stop the local anvil node if running.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnvilCommand(cmd, "stop", flags)
		},
	}

	addAnvilFlags(cmd, flags)
	return cmd
}

// newDevAnvilRestartCmd creates the restart command
func newDevAnvilRestartCmd() *cobra.Command {
	flags := &anvilFlags{}
	
	cmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart local anvil node",
		Long:  `Restart the local anvil node with CreateX factory deployed.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnvilCommand(cmd, "restart", flags)
		},
	}

	addAnvilFlags(cmd, flags)
	return cmd
}

// newDevAnvilStatusCmd creates the status command
func newDevAnvilStatusCmd() *cobra.Command {
	flags := &anvilFlags{}
	
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show anvil status",
		Long:  `Show status of the local anvil node.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnvilCommand(cmd, "status", flags)
		},
	}

	addAnvilFlags(cmd, flags)
	return cmd
}

// newDevAnvilLogsCmd creates the logs command
func newDevAnvilLogsCmd() *cobra.Command {
	flags := &anvilFlags{}
	
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Show anvil logs",
		Long:  `Show logs from the local anvil node.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnvilCommand(cmd, "logs", flags)
		},
	}

	addAnvilFlags(cmd, flags)
	return cmd
}

// runAnvilCommand executes an anvil management command
func runAnvilCommand(cmd *cobra.Command, operation string, flags *anvilFlags) error {
	// Get app instance
	app, err := getApp(cmd)
	if err != nil {
		return err
	}

	// Execute anvil operation
	params := usecase.ManageAnvilParams{
		Operation: operation,
		Name:      flags.name,
		Port:      flags.port,
		ChainID:   flags.chainID,
	}

	result, err := app.ManageAnvil.Execute(cmd.Context(), params)
	if err != nil {
		return err
	}

	// For logs operation, we need special handling
	if operation == "logs" {
		// Display header
		renderer := render.NewAnvilRenderer()
		if err := renderer.RenderLogsHeader(result); err != nil {
			return err
		}
		
		// Stream logs
		return app.AnvilManager.StreamLogs(cmd.Context(), result.Instance, os.Stdout)
	}

	// Render the result
	renderer := render.NewAnvilRenderer()
	return renderer.Render(result)
}