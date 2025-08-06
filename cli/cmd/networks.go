package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
)

var networksCmd = &cobra.Command{
	Use:   "networks",
	Short: "List available networks from foundry.toml",
	Long: `List all networks configured in the [rpc_endpoints] section of foundry.toml.

This command shows all available networks and attempts to fetch their chain IDs.`,
	RunE: runNetworks,
}

func init() {
	// Set command group
	networksCmd.GroupID = "management"
	
	// Register command
	rootCmd.AddCommand(networksCmd)
}

func runNetworks(cmd *cobra.Command, args []string) error {
	// Create network resolver
	resolver, err := network.NewResolver(".")
	if err != nil {
		return fmt.Errorf("failed to create network resolver: %w", err)
	}

	// Get all networks
	networks := resolver.GetNetworks()
	if len(networks) == 0 {
		fmt.Println("No networks configured in foundry.toml [rpc_endpoints]")
		return nil
	}

	fmt.Println("🌐 Available Networks:")
	fmt.Println()

	// Try to resolve each network
	for _, networkName := range networks {
		info, err := resolver.ResolveNetwork(networkName)
		if err != nil {
			fmt.Printf("  ❌ %s - Error: %v\n", networkName, err)
		} else {
			fmt.Printf("  ✅ %s - Chain ID: %d\n", networkName, info.ChainID)
		}
	}

	return nil
}