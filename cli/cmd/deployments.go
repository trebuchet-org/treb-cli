package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bogdan/fdeploy/cli/internal/registry"
	"github.com/bogdan/fdeploy/cli/pkg/network"
	"github.com/spf13/cobra"
)

var (
	showContract string
	fromBroadcast bool
)

var deploymentsCmd = &cobra.Command{
	Use:   "deployments",
	Short: "Deployment management commands",
	Long: `Manage deployments including showing deployment details,
listing all deployments, and tracking verification status.`,
	Aliases: []string{"deployment", "registry"},
}

var deploymentsShowCmd = &cobra.Command{
	Use:   "show [identifier]",
	Short: "Show deployment information",
	Long: `Show detailed deployment information.
	
The identifier can be:
- A contract address (0x...)
- A display name (e.g., "Counter" or "Counter:v2")
- Part of a display name (will show matches)

If multiple deployments match, you'll be prompted to select one.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		identifier := args[0]
		
		if err := showDeploymentByIdentifier(identifier); err != nil {
			checkError(err)
		}
	},
}

var deploymentsSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync registry from broadcast files",
	Long:  `Sync the registry with information from Foundry broadcast files.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := syncRegistry(); err != nil {
			checkError(err)
		}
		
		fmt.Println("Registry synced from broadcast files")
	},
}

var deploymentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all deployments",
	Long:  `List all deployments across networks and environments.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := listDeployments(); err != nil {
			checkError(err)
		}
	},
}

var deploymentsNetworksCmd = &cobra.Command{
	Use:   "networks",
	Short: "Show networks and deployment counts",
	Long:  `Show all networks in the registry with deployment counts.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := showNetworks(); err != nil {
			checkError(err)
		}
	},
}

var deploymentsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show registry status and summary",
	Long:  `Show overall registry status including project info and deployment statistics.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := showRegistryStatus(); err != nil {
			checkError(err)
		}
	},
}

var deploymentsCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean invalid or dummy registry entries",
	Long:  `Remove invalid registry entries, including old dummy entries that were never actually deployed.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := cleanRegistry(); err != nil {
			checkError(err)
		}
	},
}

func init() {
	deploymentsCmd.AddCommand(deploymentsShowCmd)
	deploymentsCmd.AddCommand(deploymentsSyncCmd)
	deploymentsCmd.AddCommand(deploymentsListCmd)
	deploymentsCmd.AddCommand(deploymentsNetworksCmd)
	deploymentsCmd.AddCommand(deploymentsStatusCmd)
	deploymentsCmd.AddCommand(deploymentsCleanCmd)
	
	// Get configured defaults (empty if no config file)
	defaultEnv, defaultNetwork, _, _ := GetConfiguredDefaults()
	
	// Create flags with defaults (empty if no config)
	deploymentsShowCmd.Flags().StringVar(&env, "env", defaultEnv, "Environment to show")
	deploymentsShowCmd.Flags().StringVar(&networkName, "network", defaultNetwork, "Network to show deployments for")
	deploymentsSyncCmd.Flags().BoolVar(&fromBroadcast, "from-broadcast", true, "Sync from broadcast files")
	
	// Mark flags as required if they don't have defaults
	if defaultEnv == "" {
		deploymentsShowCmd.MarkFlagRequired("env")
	}
	if defaultNetwork == "" {
		deploymentsShowCmd.MarkFlagRequired("network")
	}
}

func showDeploymentByIdentifier(identifier string) error {
	// Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}
	
	// Get all deployments
	allDeployments := registryManager.GetAllDeployments()
	
	// Find matching deployments
	var matches []*registry.DeploymentInfo
	identifierLower := strings.ToLower(identifier)
	
	for _, deployment := range allDeployments {
		// Check if identifier is an address
		if strings.ToLower(deployment.Address.Hex()) == identifierLower {
			matches = append(matches, deployment)
			continue
		}
		
		// Check if identifier matches or is contained in display name
		displayName := deployment.Entry.GetDisplayName()
		if strings.EqualFold(displayName, identifier) || strings.Contains(strings.ToLower(displayName), identifierLower) {
			matches = append(matches, deployment)
		}
	}
	
	if len(matches) == 0 {
		return fmt.Errorf("no deployment found matching '%s'", identifier)
	}
	
	// If single match, show it
	if len(matches) == 1 {
		return showDeploymentInfo(matches[0])
	}
	
	// Multiple matches - show selection
	fmt.Printf("Multiple deployments found matching '%s':\n\n", identifier)
	
	// Sort matches by network, then env, then contract name
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].NetworkName != matches[j].NetworkName {
			return matches[i].NetworkName < matches[j].NetworkName
		}
		if matches[i].Entry.Environment != matches[j].Entry.Environment {
			return matches[i].Entry.Environment < matches[j].Entry.Environment
		}
		return matches[i].Entry.ContractName < matches[j].Entry.ContractName
	})
	
	for i, match := range matches {
		displayName := match.Entry.GetDisplayName()
		fullId := fmt.Sprintf("%s/%s/%s", match.NetworkName, match.Entry.Environment, displayName)
		fmt.Printf("%d. %s\n   Address: %s\n\n", i+1, fullId, match.Address.Hex())
	}
	
	// Ask user to select
	fmt.Print("Select deployment (1-", len(matches), "): ")
	var selection int
	fmt.Scanln(&selection)
	
	if selection < 1 || selection > len(matches) {
		return fmt.Errorf("invalid selection")
	}
	
	return showDeploymentInfo(matches[selection-1])
}

func showDeploymentInfo(deployment *registry.DeploymentInfo) error {
	// Get network info
	resolver := network.NewResolver(".")
	networkInfo, err := resolver.ResolveNetworkByChainID(deployment.ChainID)
	if err != nil {
		return fmt.Errorf("failed to resolve network: %w", err)
	}
	
	// Display deployment information
	displayName := deployment.Entry.GetDisplayName()
	fmt.Printf("📝 Deployment: %s\n", displayName)
	fmt.Printf("   Environment: %s\n\n", deployment.Entry.Environment)
	
	// Basic deployment info
	fmt.Printf("🚀 Contract Information:\n")
	fmt.Printf("   Address: %s\n", deployment.Address.Hex())
	fmt.Printf("   Type: %s\n", deployment.Entry.Type)
	fmt.Printf("   Salt: %s\n", deployment.Entry.Salt)
	if deployment.Entry.InitCodeHash != "" {
		fmt.Printf("   Init Code Hash: %s\n", deployment.Entry.InitCodeHash)
	}
	fmt.Println()
	
	// Deployment details
	fmt.Printf("📊 Deployment Details:\n")
	if deployment.Entry.Deployment.TxHash != nil && deployment.Entry.Deployment.TxHash.Hex() != "0x0000000000000000000000000000000000000000000000000000000000000000" {
		fmt.Printf("   Transaction: %s\n", deployment.Entry.Deployment.TxHash.Hex())
	}
	if deployment.Entry.Deployment.BlockNumber > 0 {
		fmt.Printf("   Block: %d\n", deployment.Entry.Deployment.BlockNumber)
	}
	fmt.Printf("   Network: %s (Chain ID: %s)\n", networkInfo.Name, deployment.ChainID)
	fmt.Printf("   Deployed: %s\n", deployment.Entry.Deployment.Timestamp.Format("2006-01-02 15:04:05"))
	if deployment.Entry.Deployment.BroadcastFile != "" {
		fmt.Printf("   Broadcast File: %s\n", deployment.Entry.Deployment.BroadcastFile)
	}
	fmt.Println()
	
	// Metadata
	fmt.Printf("📋 Contract Metadata:\n")
	if deployment.Entry.Metadata.ContractPath != "" {
		fmt.Printf("   Path: %s\n", deployment.Entry.Metadata.ContractPath)
	}
	fmt.Printf("   Version: %s\n", deployment.Entry.Metadata.ContractVersion)
	fmt.Printf("   Compiler: %s\n", deployment.Entry.Metadata.Compiler)
	if deployment.Entry.Metadata.SourceCommit != "" {
		fmt.Printf("   Source Commit: %s\n", deployment.Entry.Metadata.SourceCommit)
	}
	if deployment.Entry.Metadata.SourceHash != "" {
		fmt.Printf("   Source Hash: %s\n", deployment.Entry.Metadata.SourceHash)
	}
	fmt.Println()
	
	// Verification status
	fmt.Printf("🔍 Verification:\n")
	fmt.Printf("   Status: %s\n", deployment.Entry.Verification.Status)
	if deployment.Entry.Verification.ExplorerUrl != "" {
		fmt.Printf("   Explorer: %s\n", deployment.Entry.Verification.ExplorerUrl)
	}
	if deployment.Entry.Verification.Reason != "" {
		fmt.Printf("   Reason: %s\n", deployment.Entry.Verification.Reason)
	}
	
	return nil
}

func showDeployment(contract string) error {
	// Resolve network configuration
	resolver := network.NewResolver(".")
	networkInfo, err := resolver.ResolveNetwork(networkName)
	if err != nil {
		return fmt.Errorf("failed to resolve network: %w", err)
	}

	// Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	deployment := registryManager.GetDeployment(contract, env, networkInfo.ChainID)
	if deployment == nil {
		return fmt.Errorf("no deployment found for %s in %s environment on network %s (chain ID %d)", contract, env, networkInfo.Name, networkInfo.ChainID)
	}

	// Display deployment information in a readable format
	displayName := deployment.GetDisplayName()
	fmt.Printf("📝 Deployment: %s\n", displayName)
	fmt.Printf("   Environment: %s\n\n", deployment.Environment)
	
	// Basic deployment info
	fmt.Printf("🚀 Contract Information:\n")
	fmt.Printf("   Address: %s\n", deployment.Address.Hex())
	fmt.Printf("   Type: %s\n", deployment.Type)
	fmt.Printf("   Salt: %s\n", deployment.Salt)
	if deployment.InitCodeHash != "" {
		fmt.Printf("   Init Code Hash: %s\n", deployment.InitCodeHash)
	}
	fmt.Println()
	
	// Deployment details
	fmt.Printf("📊 Deployment Details:\n")
	if deployment.Deployment.TxHash != nil && deployment.Deployment.TxHash.Hex() != "0x0000000000000000000000000000000000000000000000000000000000000000" {
		fmt.Printf("   Transaction: %s\n", deployment.Deployment.TxHash.Hex())
	}
	if deployment.Deployment.BlockNumber > 0 {
		fmt.Printf("   Block: %d\n", deployment.Deployment.BlockNumber)
	}
	fmt.Printf("   Network: %s (Chain ID: %d)\n", networkInfo.Name, networkInfo.ChainID)
	fmt.Printf("   Deployed: %s\n", deployment.Deployment.Timestamp.Format("2006-01-02 15:04:05"))
	if deployment.Deployment.BroadcastFile != "" {
		fmt.Printf("   Broadcast File: %s\n", deployment.Deployment.BroadcastFile)
	}
	fmt.Println()
	
	// Metadata
	fmt.Printf("📋 Contract Metadata:\n")
	if deployment.Metadata.ContractPath != "" {
		fmt.Printf("   Path: %s\n", deployment.Metadata.ContractPath)
	}
	fmt.Printf("   Version: %s\n", deployment.Metadata.ContractVersion)
	fmt.Printf("   Compiler: %s\n", deployment.Metadata.Compiler)
	if deployment.Metadata.SourceCommit != "" {
		fmt.Printf("   Source Commit: %s\n", deployment.Metadata.SourceCommit)
	}
	if deployment.Metadata.SourceHash != "" {
		fmt.Printf("   Source Hash: %s\n", deployment.Metadata.SourceHash)
	}
	fmt.Println()
	
	// Verification status
	fmt.Printf("🔍 Verification:\n")
	fmt.Printf("   Status: %s\n", deployment.Verification.Status)
	if deployment.Verification.ExplorerUrl != "" {
		fmt.Printf("   Explorer: %s\n", deployment.Verification.ExplorerUrl)
	}
	if deployment.Verification.Reason != "" {
		fmt.Printf("   Reason: %s\n", deployment.Verification.Reason)
	}

	return nil
}

func syncRegistry() error {
	// Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	// TODO: Implement sync from broadcast files
	fmt.Println("Syncing from broadcast files...")
	
	// For now, just save the registry
	return registryManager.Save()
}

func listDeployments() error {
	// Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	deployments := registryManager.GetAllDeployments()
	if len(deployments) == 0 {
		fmt.Println("No deployments found")
		return nil
	}

	fmt.Printf("Deployments (%d total):\n\n", len(deployments))
	
	// Group by (network, env)
	type groupKey struct {
		network string
		env     string
	}
	groups := make(map[groupKey][]*registry.DeploymentInfo)
	
	// First pass: collect all environments per network
	networkEnvs := make(map[string]map[string]bool)
	
	for _, deployment := range deployments {
		env := deployment.Entry.Environment
		
		// Track environments per network
		if networkEnvs[deployment.NetworkName] == nil {
			networkEnvs[deployment.NetworkName] = make(map[string]bool)
		}
		networkEnvs[deployment.NetworkName][env] = true
		
		gk := groupKey{
			network: deployment.NetworkName,
			env:     env,
		}
		groups[gk] = append(groups[gk], deployment)
	}
	
	// Sort group keys
	var groupKeys []groupKey
	for gk := range groups {
		groupKeys = append(groupKeys, gk)
	}
	sort.Slice(groupKeys, func(i, j int) bool {
		if groupKeys[i].network != groupKeys[j].network {
			return groupKeys[i].network < groupKeys[j].network
		}
		return groupKeys[i].env < groupKeys[j].env
	})
	
	// Display groups
	for _, gk := range groupKeys {
		// Check if we should show environment
		multipleEnvs := len(networkEnvs[gk.network]) > 1
		
		if multipleEnvs {
			fmt.Printf("▶ %s (%s)\n\n", gk.network, gk.env)
		} else {
			fmt.Printf("▶ %s\n\n", gk.network)
		}
		
		// Sort deployments within group by timestamp (most recent first)
		deploymentList := groups[gk]
		sort.Slice(deploymentList, func(i, j int) bool {
			return deploymentList[i].Entry.Deployment.Timestamp.After(deploymentList[j].Entry.Deployment.Timestamp)
		})
		
		// Find longest display name for alignment
		maxNameLen := 0
		for _, deployment := range deploymentList {
			displayName := deployment.Entry.GetDisplayName()
			if len(displayName) > maxNameLen {
				maxNameLen = len(displayName)
			}
		}
		
		// Display with aligned columns
		for _, deployment := range deploymentList {
			displayName := deployment.Entry.GetDisplayName()
			timestamp := deployment.Entry.Deployment.Timestamp.Format("2006-01-02 15:04:05")
			fmt.Printf("  %-*s  %s  %s\n", maxNameLen, displayName, deployment.Address.Hex(), timestamp)
		}
		fmt.Println()
	}

	return nil
}

func showNetworks() error {
	// Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	networks := registryManager.GetNetworkSummary()
	if len(networks) == 0 {
		fmt.Println("No networks found")
		return nil
	}

	fmt.Println("Deployment Networks:\n")
	
	// Sort networks by chain ID for consistent output
	var chainIDs []string
	for chainID := range networks {
		chainIDs = append(chainIDs, chainID)
	}
	sort.Strings(chainIDs)
	
	totalDeployments := 0
	for _, chainID := range chainIDs {
		networkInfo := networks[chainID]
		fmt.Printf("📡 %s (Chain ID: %s)\n", networkInfo.Name, chainID)
		fmt.Printf("   Deployments: %d\n", networkInfo.DeploymentCount)
		
		if len(networkInfo.Contracts) > 0 {
			// Sort contracts alphabetically
			sort.Strings(networkInfo.Contracts)
			fmt.Printf("   Contracts: %s\n", strings.Join(networkInfo.Contracts, ", "))
		}
		fmt.Println()
		
		totalDeployments += networkInfo.DeploymentCount
	}
	
	fmt.Printf("Total: %d networks, %d deployments\n", len(networks), totalDeployments)
	return nil
}

func showRegistryStatus() error {
	// Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	status := registryManager.GetStatus()
	
	fmt.Println("Deployments Status:\n")
	
	// Project info
	fmt.Printf("📋 Project: %s (v%s)\n", status.ProjectName, status.ProjectVersion)
	if status.LastUpdated != "" {
		fmt.Printf("🕒 Last Updated: %s\n", status.LastUpdated)
	}
	fmt.Println()
	
	// Statistics
	fmt.Printf("📊 Statistics:\n")
	fmt.Printf("   Networks: %d\n", status.NetworkCount)
	fmt.Printf("   Total Deployments: %d\n", status.TotalDeployments)
	fmt.Printf("   Verified: %d\n", status.VerifiedCount)
	fmt.Printf("   Pending Verification: %d\n", status.PendingVerification)
	fmt.Println()
	
	// Recent deployments
	if len(status.RecentDeployments) > 0 {
		fmt.Printf("🕒 Recent Deployments:\n")
		for _, recent := range status.RecentDeployments {
			// Build display name
			displayName := recent.Contract
			if recent.Type == "proxy" {
				displayName = recent.Contract + "Proxy"
			}
			if recent.ProxyLabel != "" {
				displayName = displayName + ":" + recent.ProxyLabel
			}
			fmt.Printf("   %s (%s): %s on %s\n", displayName, recent.Environment, recent.Address, recent.Network)
		}
		fmt.Println()
	}
	
	return nil
}

func cleanRegistry() error {
	// Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	fmt.Println("Cleaning deployments...")
	
	cleaned := registryManager.CleanInvalidEntries()
	
	if cleaned > 0 {
		fmt.Printf("Removed %d invalid entries\n", cleaned)
		err := registryManager.Save()
		if err != nil {
			return fmt.Errorf("failed to save cleaned registry: %w", err)
		}
		fmt.Println("Registry cleaned and saved")
	} else {
		fmt.Println("No invalid entries found")
	}
	
	return nil
}