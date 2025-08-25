package cmd

import (
	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/pkg/anvil"
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Development utilities",
	Long:  `Development utilities for troubleshooting treb configuration and environment.`,
}

var devAnvilCmd = &cobra.Command{
	Use:   "anvil",
	Short: "Manage local anvil node",
	Long:  `Manage a local anvil node for testing deployments with CreateX factory automatically deployed.`,
}

var debugAnvilStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start local anvil node",
	Long:  `Start a local anvil node with CreateX factory deployed. Fails if already running.`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		port, _ := cmd.Flags().GetString("port")
		chainID, _ := cmd.Flags().GetString("chain-id")
		if err := anvil.StartAnvilInstance(name, port, chainID); err != nil {
			checkError(err)
		}
	},
}

var debugAnvilStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop local anvil node",
	Long:  `Stop the local anvil node if running.`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		port, _ := cmd.Flags().GetString("port")
		if err := anvil.StopAnvilInstance(name, port); err != nil {
			checkError(err)
		}
	},
}

var debugAnvilRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart local anvil node",
	Long:  `Restart the local anvil node with CreateX factory deployed.`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		port, _ := cmd.Flags().GetString("port")
		chainID, _ := cmd.Flags().GetString("chain-id")
		if err := anvil.RestartAnvilInstance(name, port, chainID); err != nil {
			checkError(err)
		}
	},
}

var debugAnvilLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show anvil logs",
	Long:  `Show logs from the local anvil node.`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		port, _ := cmd.Flags().GetString("port")
		if err := anvil.ShowAnvilLogsInstance(name, port); err != nil {
			checkError(err)
		}
	},
}

var debugAnvilStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show anvil status",
	Long:  `Show status of the local anvil node.`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		port, _ := cmd.Flags().GetString("port")
		if err := anvil.ShowAnvilStatusInstance(name, port); err != nil {
			checkError(err)
		}
	},
}

func init() {
	// Add anvil command with subcommands
	devAnvilCmd.AddCommand(debugAnvilStartCmd)
	devAnvilCmd.AddCommand(debugAnvilStopCmd)
	devAnvilCmd.AddCommand(debugAnvilRestartCmd)
	devAnvilCmd.AddCommand(debugAnvilLogsCmd)
	devAnvilCmd.AddCommand(debugAnvilStatusCmd)
	devCmd.AddCommand(devAnvilCmd)

	// Shared flags for all anvil subcommands
	for _, c := range []*cobra.Command{debugAnvilStartCmd, debugAnvilStopCmd, debugAnvilRestartCmd, debugAnvilLogsCmd, debugAnvilStatusCmd} {
		c.Flags().String("name", "anvil0", "Instance name (e.g. anvil0, anvil1)")
		c.Flags().String("port", "8545", "RPC port to bind")
		c.Flags().String("chain-id", "", "Chain ID to use for the instance (optional)")
	}
}
