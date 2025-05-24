package cmd

import (
	"crypto/ecdsa"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/internal/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
	"github.com/trebuchet-org/treb-cli/cli/pkg/safe"
)

var (
	showContract  string
	fromBroadcast bool
	debugSync     bool
)

var deploymentsCmd = &cobra.Command{
	Use:   "deployments",
	Short: "Deployment management commands",
	Long: `Manage deployments including showing deployment details,
listing all deployments, and tracking verification status.`,
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
	Args: cobra.ExactArgs(1),
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

var deploymentsTagCmd = &cobra.Command{
	Use:   "tag [identifier] --tag <tag>",
	Short: "Add or remove tags from deployments",
	Long: `Add or remove tags from deployments for versioning and organization.

The identifier can be:
- A contract address (0x...)
- A display name (e.g., "Counter" or "Counter:v2")
- Part of a display name (will show matches)

Use --all to tag all deployments that match the identifier.
Use --remove to remove tags instead of adding them.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		identifier := args[0]
		tag, _ := cmd.Flags().GetString("tag")
		all, _ := cmd.Flags().GetBool("all")
		remove, _ := cmd.Flags().GetBool("remove")

		if tag == "" {
			checkError(fmt.Errorf("--tag flag is required"))
		}

		if err := tagDeployments(identifier, tag, all, remove); err != nil {
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
	deploymentsCmd.AddCommand(deploymentsTagCmd)

	// Create flags for sync command
	deploymentsSyncCmd.Flags().BoolVar(&fromBroadcast, "from-broadcast", true, "Sync from broadcast files")
	deploymentsSyncCmd.Flags().BoolVar(&debugSync, "debug", false, "Show debug information during sync")

	// Add flags for tag command
	deploymentsTagCmd.Flags().String("tag", "", "Tag to add or remove from deployments")
	deploymentsTagCmd.Flags().Bool("all", false, "Tag all deployments that match the identifier")
	deploymentsTagCmd.Flags().Bool("remove", false, "Remove the tag instead of adding it")
	deploymentsTagCmd.MarkFlagRequired("tag")
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
	fmt.Printf("Deployment: %s\n", displayName)
	fmt.Printf("Environment: %s\n\n", deployment.Entry.Environment)

	// Basic deployment info
	fmt.Printf("Contract Information:\n")
	fmt.Printf("   Address: %s\n", deployment.Address.Hex())
	fmt.Printf("   Type: %s\n", deployment.Entry.Type)
	fmt.Printf("   Salt: %s\n", deployment.Entry.Salt)
	if deployment.Entry.InitCodeHash != "" {
		fmt.Printf("   Init Code Hash: %s\n", deployment.Entry.InitCodeHash)
	}
	fmt.Println()

	// Deployment details
	fmt.Printf("Deployment Details:\n")

	// Show deployment status
	status := deployment.Entry.Deployment.Status
	if status == "" {
		status = "deployed"
	}
	fmt.Printf("   Status: %s\n", status)

	// Safe-specific information for pending deployments
	if status == "pending_safe" {
		if deployment.Entry.Deployment.SafeAddress != "" {
			fmt.Printf("   Safe Address: %s\n", deployment.Entry.Deployment.SafeAddress)
		}
		if deployment.Entry.Deployment.SafeNonce > 0 {
			fmt.Printf("   Safe Nonce: %d\n", deployment.Entry.Deployment.SafeNonce)
		}
		if deployment.Entry.Deployment.SafeTxHash != nil {
			fmt.Printf("   Safe Tx Hash: %s\n", deployment.Entry.Deployment.SafeTxHash.Hex())

			// Try to get current confirmation status
			chainIDUint, err := strconv.ParseUint(deployment.ChainID, 10, 64)
			if err == nil {
				if safeClient, err := safe.NewClient(chainIDUint); err == nil {
					if tx, err := safeClient.GetTransaction(*deployment.Entry.Deployment.SafeTxHash); err == nil {
						fmt.Printf("   Confirmations: %d/%d\n", len(tx.Confirmations), tx.ConfirmationsRequired)
					}
				}
			}
		}
		fmt.Printf("   This deployment is pending execution in the Safe UI\n")
	} else {
		// Regular deployment info
		if deployment.Entry.Deployment.TxHash != nil && deployment.Entry.Deployment.TxHash.Hex() != "0x0000000000000000000000000000000000000000000000000000000000000000" {
			fmt.Printf("   Transaction: %s\n", deployment.Entry.Deployment.TxHash.Hex())
		}
		if deployment.Entry.Deployment.BlockNumber > 0 {
			fmt.Printf("   Block: %d\n", deployment.Entry.Deployment.BlockNumber)
		}
	}

	fmt.Printf("   Network: %s (Chain ID: %s)\n", networkInfo.Name, deployment.ChainID)
	fmt.Printf("   Timestamp: %s\n", deployment.Entry.Deployment.Timestamp.Format("2006-01-02 15:04:05"))
	if deployment.Entry.Deployment.BroadcastFile != "" {
		fmt.Printf("   Broadcast File: %s\n", deployment.Entry.Deployment.BroadcastFile)
	}
	fmt.Println()

	// Metadata
	fmt.Printf("Contract Metadata:\n")
	if deployment.Entry.Metadata.ContractPath != "" {
		fmt.Printf("   Path: %s\n", deployment.Entry.Metadata.ContractPath)
	}
	fmt.Printf("   Compiler: %s\n", deployment.Entry.Metadata.Compiler)
	if deployment.Entry.Metadata.SourceCommit != "" {
		fmt.Printf("   Source Commit: %s\n", deployment.Entry.Metadata.SourceCommit)
	}
	if deployment.Entry.Metadata.SourceHash != "" {
		fmt.Printf("   Source Hash: %s\n", deployment.Entry.Metadata.SourceHash)
	}
	if len(deployment.Entry.Tags) > 0 {
		fmt.Printf("   Tags: %s\n", strings.Join(deployment.Entry.Tags, ", "))
	}
	fmt.Println()

	// Verification status
	fmt.Printf("Verification:\n")
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
	fmt.Printf("Deployment: %s\n", displayName)
	fmt.Printf("Environment: %s\n\n", deployment.Environment)

	// Basic deployment info
	fmt.Printf("Contract Information:\n")
	fmt.Printf("   Address: %s\n", deployment.Address.Hex())
	fmt.Printf("   Type: %s\n", deployment.Type)
	fmt.Printf("   Salt: %s\n", deployment.Salt)
	if deployment.InitCodeHash != "" {
		fmt.Printf("   Init Code Hash: %s\n", deployment.InitCodeHash)
	}
	fmt.Println()

	// Deployment details
	fmt.Printf("Deployment Details:\n")
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
	fmt.Printf("Contract Metadata:\n")
	if deployment.Metadata.ContractPath != "" {
		fmt.Printf("   Path: %s\n", deployment.Metadata.ContractPath)
	}
	fmt.Printf("   Compiler: %s\n", deployment.Metadata.Compiler)
	if deployment.Metadata.SourceCommit != "" {
		fmt.Printf("   Source Commit: %s\n", deployment.Metadata.SourceCommit)
	}
	if deployment.Metadata.SourceHash != "" {
		fmt.Printf("   Source Hash: %s\n", deployment.Metadata.SourceHash)
	}
	if len(deployment.Tags) > 0 {
		fmt.Printf("   Tags: %s\n", strings.Join(deployment.Tags, ", "))
	}
	fmt.Println()

	// Verification status
	fmt.Printf("Verification:\n")
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

	fmt.Println("Syncing registry...")

	// Check and update pending Safe transactions
	if err := syncPendingSafeTransactions(registryManager); err != nil {
		fmt.Printf("Warning: Failed to sync Safe transactions: %v\n", err)
	}

	return registryManager.Save()
}

func listDeployments() error {
	// Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	deployConfig, err := config.LoadDeployConfig(".")
	if err != nil {
		return fmt.Errorf("failed to load deploy config: %w", err)
	}
	deployments := registryManager.GetAllDeployments()
	if len(deployments) == 0 {
		fmt.Println("No deployments found")
		return nil
	}

	// Create color styles
	envHeader := color.New(color.BgGreen, color.FgBlack)
	envHeaderBold := color.New(color.BgGreen, color.FgBlack, color.Bold)
	chainHeader := color.New(color.BgCyan, color.FgWhite)
	chainHeaderBold := color.New(color.BgCyan, color.FgWhite, color.Bold)
	// deployerStyle := color.New(color.Faint)
	contractNameStyle := color.New(color.Bold)
	addressStyle := color.New(color.Bold, color.FgHiWhite)
	timestampStyle := color.New(color.Faint)
	pendingStyle := color.New(color.FgYellow)
	tagsStyle := color.New(color.FgCyan)

	fmt.Printf("Deployments (%d total):\n\n", len(deployments))

	groups := make(map[string]map[string][]*registry.DeploymentInfo)

	// First pass: collect all environments per network
	envs := make([]string, 0)
	networks := make([]string, 0)
	envsByNetwork := make(map[string][]string)

	for _, deployment := range deployments {
		env := deployment.Entry.Environment
		network := deployment.NetworkName

		if !slices.Contains(networks, network) {
			networks = append(networks, network)
		}

		if !slices.Contains(envs, env) {
			envs = append(envs, env)
		}

		if envsByNetwork[network] == nil {
			envsByNetwork[network] = make([]string, 0)
		}

		if !slices.Contains(envsByNetwork[network], env) {
			envsByNetwork[network] = append(envsByNetwork[network], env)
		}

		if groups[deployment.NetworkName] == nil {
			groups[deployment.NetworkName] = make(map[string][]*registry.DeploymentInfo)
		}

		groups[deployment.NetworkName][env] = append(groups[deployment.NetworkName][env], deployment)
	}

	slices.Sort(envs)
	slices.Sort(networks)

	// Calculate global max name length for alignment
	maxNameLen := 0
	for _, deployment := range deployments {
		displayName := deployment.Entry.GetDisplayName()
		if len(displayName) > maxNameLen {
			maxNameLen = len(displayName)
		}
	}

	// Display groups
	for _, env := range envs {
		envConfig, err := deployConfig.GetEnvironmentConfig(env)
		if err != nil {
			return fmt.Errorf("failed to get environment config: %w", err)
		}

		deployerAddress := "<unknown>"
		if envConfig.Deployer.Type == "safe" {
			deployerAddress = envConfig.Deployer.Safe
		} else if envConfig.Deployer.Type == "private_key" {
			// Convert private key to address for display
			if addr, err := privateKeyToAddress(envConfig.Deployer.PrivateKey); err == nil {
				deployerAddress = addr
			} else {
				deployerAddress = "<invalid>"
			}
		}

		if len(envs) > 1 {
			// Environment header with colored environment name only
			envHeader.Print(" ▶ Environment: ")
			envHeaderBold.Print(env)
			envHeader.Print(" ")
			envHeader.Printf(" (deployer: %s)", deployerAddress)
			fmt.Println() // No extra newline after header
		}
		for _, network := range networks {
			deployments := groups[network][env]
			if len(deployments) == 0 {
				continue
			}
			// Chain header with color starting at text (no whitespace prefix)
			chainHeader.Print(" ⛓ Chain: ")
			chainHeaderBold.Print(network)
			chainHeader.Print(" ")
			fmt.Println() // No extra newline after header
			fmt.Println() // No extra newline after header

			sort.Slice(deployments, func(i, j int) bool {
				return deployments[i].Entry.Deployment.Timestamp.After(deployments[j].Entry.Deployment.Timestamp)
			})
			for _, deployment := range deployments {
				displayName := deployment.Entry.GetDisplayName()
				timestamp := deployment.Entry.Deployment.Timestamp.Format("2006-01-02 15:04:05")

				// Print contract name in bold with extra space for alignment
				fmt.Printf("   ")
				contractNameStyle.Printf("%-*s", maxNameLen, displayName)
				fmt.Print("  ")

				// Print address in bold
				addressStyle.Print(deployment.Address.Hex())
				fmt.Print("  ")

				// Print timestamp in faint
				timestampStyle.Print(timestamp)

				// Add status indicator for pending Safe deployments
				if deployment.Entry.Deployment.Status == "pending_safe" {
					pendingStyle.Print(" ⏳ pending safe execution")
				}

				// Add tags if present
				if len(deployment.Entry.Tags) > 0 {
					fmt.Print(" ")
					tagsStyle.Printf("[%s]", strings.Join(deployment.Entry.Tags, ", "))
				}

				fmt.Println()
			}
			fmt.Println()
		}
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
		fmt.Printf("%s (Chain ID: %s)\n", networkInfo.Name, chainID)
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
	fmt.Printf("Project: %s (v%s)\n", status.ProjectName, status.ProjectVersion)
	if status.LastUpdated != "" {
		fmt.Printf("Last Updated: %s\n", status.LastUpdated)
	}
	fmt.Println()

	// Statistics
	fmt.Printf("Statistics:\n")
	fmt.Printf("   Networks: %d\n", status.NetworkCount)
	fmt.Printf("   Total Deployments: %d\n", status.TotalDeployments)
	fmt.Printf("   Verified: %d\n", status.VerifiedCount)
	fmt.Printf("   Pending Verification: %d\n", status.PendingVerification)
	fmt.Println()

	// Recent deployments
	if len(status.RecentDeployments) > 0 {
		fmt.Printf("Recent Deployments:\n")
		for _, recent := range status.RecentDeployments {
			// Build display name
			displayName := recent.Contract
			if recent.Type == "proxy" {
				displayName = recent.Contract + "Proxy"
			}
			if recent.Label != "" {
				displayName = displayName + ":" + recent.Label
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

// syncPendingSafeTransactions checks pending Safe transactions and updates their status
func syncPendingSafeTransactions(registryManager *registry.Manager) error {
	deployments := registryManager.GetAllDeployments()

	// Group pending deployments by chain ID
	pendingByChain := make(map[uint64][]*registry.DeploymentInfo)

	for _, deployment := range deployments {
		if deployment.Entry.Deployment.Status == "pending_safe" && deployment.Entry.Deployment.SafeTxHash != nil {
			chainID, err := strconv.ParseUint(deployment.ChainID, 10, 64)
			if err != nil {
				fmt.Printf("Warning: Invalid chain ID %s for deployment %s\n", deployment.ChainID, deployment.Address.Hex())
				continue
			}
			pendingByChain[chainID] = append(pendingByChain[chainID], deployment)
		}
	}

	if len(pendingByChain) == 0 {
		fmt.Println("No pending Safe transactions found")
		return nil
	}

	fmt.Printf("Found pending Safe transactions on %d network(s)\n", len(pendingByChain))

	// Check each chain
	for chainID, pendingDeployments := range pendingByChain {
		fmt.Printf("\nChecking %d pending transaction(s) on chain %d...\n", len(pendingDeployments), chainID)

		// Create Safe client for this chain
		safeClient, err := safe.NewClient(chainID)
		if err != nil {
			fmt.Printf("Warning: Cannot create Safe client for chain %d: %v\n", chainID, err)
			continue
		}

		// Enable debug if flag is set
		safeClient.SetDebug(debugSync)

		// Check each pending deployment
		for _, deployment := range pendingDeployments {
			safeTxHash := *deployment.Entry.Deployment.SafeTxHash
			fmt.Printf("  Checking Safe tx %s for %s... \n", safeTxHash.Hex(), deployment.Entry.GetDisplayName())

			// Debug info
			if debugSync {
				fmt.Printf("    [DEBUG] Deployment address: %s\n", deployment.Address.Hex())
				fmt.Printf("    [DEBUG] Safe address: %s\n", deployment.Entry.Deployment.SafeAddress)
				fmt.Printf("    [DEBUG] Environment: %s\n", deployment.Entry.Environment)
			}

			// Check if transaction is executed
			isExecuted, ethTxHash, err := safeClient.IsTransactionExecuted(safeTxHash)
			if err != nil {
				fmt.Printf("    ERROR: %v\n", err)

				// Provide helpful context for common errors
				if strings.Contains(err.Error(), "transaction not found") {
					fmt.Printf("    HINT: This might happen if:\n")
					fmt.Printf("      - The Safe transaction was never created (check if Safe address is correct)\n")
					fmt.Printf("      - The transaction is on a different network\n")
					fmt.Printf("      - The Safe Transaction Service hasn't indexed it yet (try again later)\n")

					if deployment.Entry.Deployment.SafeAddress == "" || deployment.Entry.Deployment.SafeAddress == "0x0000000000000000000000000000000000000000" {
						fmt.Printf("      - WARNING: Safe address is missing! This deployment needs to be re-executed.\n")
					}
				}
				continue
			}

			if isExecuted && ethTxHash != nil {
				fmt.Printf("EXECUTED (tx: %s)\n", ethTxHash.Hex())

				// Update the deployment entry
				deployment.Entry.Deployment.Status = "deployed"
				deployment.Entry.Deployment.TxHash = ethTxHash

				// Update in registry
				key := strings.ToLower(deployment.Address.Hex())
				if err := registryManager.UpdateDeployment(key, deployment.Entry); err != nil {
					fmt.Printf("    Warning: Failed to update registry: %v\n", err)
				}
			} else {
				// Get more details about the pending transaction
				tx, err := safeClient.GetTransaction(safeTxHash)
				if err == nil {
					fmt.Printf("    PENDING (%d/%d confirmations)\n", len(tx.Confirmations), tx.ConfirmationsRequired)
				} else {
					fmt.Printf("    PENDING (couldn't get confirmation details)\n")
				}
			}
		}
	}

	return nil
}

// privateKeyToAddress derives the Ethereum address from a private key
func privateKeyToAddress(privateKeyHex string) (string, error) {
	// Remove 0x prefix if present
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")

	// Convert hex string to ECDSA private key
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return "", fmt.Errorf("invalid private key: %w", err)
	}

	// Get the public key from the private key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("error casting public key to ECDSA")
	}

	// Derive the Ethereum address
	address := crypto.PubkeyToAddress(*publicKeyECDSA)
	return address.Hex(), nil
}
