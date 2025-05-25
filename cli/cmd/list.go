package cmd

import (
	"crypto/ecdsa"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/internal/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

var (
	filterEnv      string
	filterNetwork  string
	filterContract string
	showLibraries  bool
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all deployments",
	Long: `Display deployments organized by environment and network.
Shows contract addresses, deployment status, and version tags.

Filters:
  --filter-env      Filter by environment (exact match, case-insensitive)
  --filter-network  Filter by network (exact match, case-insensitive)
  --filter-contract Filter by contract name (partial match, case-insensitive)
  --libraries        Show deployed libraries instead of contracts`,
	Run: func(cmd *cobra.Command, args []string) {
		if showLibraries {
			if err := listLibraries(); err != nil {
				checkError(err)
			}
		} else {
			if err := listDeployments(); err != nil {
				checkError(err)
			}
		}
	},
}

func init() {
	listCmd.Flags().StringVar(&filterEnv, "filter-env", "", "Filter by environment")
	listCmd.Flags().StringVar(&filterNetwork, "filter-network", "", "Filter by network")
	listCmd.Flags().StringVar(&filterContract, "filter-contract", "", "Filter by contract name (partial match)")
	listCmd.Flags().BoolVar(&showLibraries, "libraries", false, "Show deployed libraries instead of contracts")
}

// columnWidths stores the calculated column widths for consistent table rendering
type columnWidths struct {
	contract  int
	address   int
	verified  int
	timestamp int
	status    int
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
	allDeployments := registryManager.GetAllDeployments()
	
	// Apply filters
	deployments := filterDeployments(allDeployments)
	
	if len(deployments) == 0 {
		if len(allDeployments) > 0 {
			fmt.Println("No deployments found matching the filters")
		} else {
			fmt.Println("No deployments found")
		}
		return nil
	}

	// Create color styles
	envBg := color.BgYellow
	chainBg := color.BgCyan
	envHeader := color.New(envBg, color.FgBlack)
	envHeaderBold := color.New(envBg, color.FgBlack, color.Bold)
	chainHeader := color.New(chainBg, color.FgBlack)
	chainHeaderBold := color.New(chainBg, color.FgBlack, color.Bold)
	contractNameStyle := color.New(color.Bold)
	addressStyle := color.New(color.Bold, color.FgHiWhite)
	timestampStyle := color.New(color.Faint)
	pendingStyle := color.New(color.FgYellow)
	tagsStyle := color.New(color.FgCyan)
	verifiedStyle := color.New(color.FgGreen)
	notVerifiedStyle := color.New(color.FgRed)

	fmt.Printf("Deployments (%d total):\n\n", len(deployments))

	groups := make(map[string]map[string][]*registry.DeploymentInfo)

	// First pass: collect all environments per network and calculate max widths
	envs := make([]string, 0)
	networks := make([]string, 0)
	envsByNetwork := make(map[string][]string)

	// Calculate max widths across all deployments
	widths := calculateColumnWidths(deployments)

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

		// Always show environment header when there are deployments
		// Environment header with colored environment name only
		envHeader.Print("   ◎ environment ")
		envHeaderBold.Printf(" %-*s ", 35, strings.ToUpper(env))
		envHeader.Printf("  deployer ")
		envHeaderBold.Printf("%s ", deployerAddress)
		fmt.Println()
		// Filter networks to only show those with deployments for this env
		networksWithDeployments := []string{}
		for _, network := range networks {
			if len(groups[network][env]) > 0 {
				networksWithDeployments = append(networksWithDeployments, network)
			}
		}

		for netIdx, network := range networksWithDeployments {
			deployments := groups[network][env]

			// Sort deployments by timestamp (newest first)
			sort.Slice(deployments, func(i, j int) bool {
				return deployments[i].Entry.Deployment.Timestamp.After(deployments[j].Entry.Deployment.Timestamp)
			})

			// Determine if this is the last network for tree drawing
			isLastNetwork := netIdx == len(networksWithDeployments)-1
			treePrefix := "├─"
			continuationPrefix := "│ "
			if isLastNetwork {
				treePrefix = "└─"
				continuationPrefix = "  "
			}

			// Create table with chain header as the first row
			t := table.NewWriter()
			t.SetStyle(table.StyleLight)
			t.Style().Options.SeparateRows = false
			t.Style().Options.DrawBorder = false
			t.Style().Options.SeparateHeader = false
			t.Style().Options.SeparateColumns = false
			t.Style().Box = table.BoxStyle{
				PaddingLeft:  "",
				PaddingRight: "  ",
			}

			// Configure column styles with calculated widths
			t.SetColumnConfigs([]table.ColumnConfig{
				{Number: 1, Align: text.AlignLeft, WidthMin: widths.contract, WidthMax: widths.contract},
				{Number: 2, Align: text.AlignLeft, WidthMin: widths.address, WidthMax: widths.address},
				{Number: 3, Align: text.AlignLeft, WidthMin: widths.verified, WidthMax: widths.verified},
				{Number: 4, Align: text.AlignLeft, WidthMin: widths.timestamp, WidthMax: widths.timestamp},
			})

			// Add chain header with proper tree character
			chainHeaderRow := fmt.Sprintf("%s%s%s",
				treePrefix,
				chainHeader.Sprint(" ⛓ chain       "),
				chainHeaderBold.Sprintf(" %s ", strings.ToUpper(network)))

			// Print the chain header outside the table
			fmt.Println(chainHeaderRow)
			fmt.Println(continuationPrefix) // Continue the tree line

			for _, deployment := range deployments {
				displayName := deployment.Entry.GetDisplayName()
				timestamp := deployment.Entry.Deployment.Timestamp.Format("2006-01-02 15:04:05")

				// Build contract name with tags
				contractCell := contractNameStyle.Sprint(displayName)
				if len(deployment.Entry.Tags) > 0 {
					contractCell += " " + tagsStyle.Sprintf("(%s)", deployment.Entry.Tags[0])
				}

				// Address cell
				addressCell := addressStyle.Sprint(deployment.Address.Hex())

				// Verified status or pending safe status
				verifiedCell := ""
				if deployment.Entry.Deployment.Status == "pending_safe" {
					verifiedCell = pendingStyle.Sprint("⧖ deploy queued")
				} else {
					switch deployment.Entry.Verification.Status {
					case "verified":
						verifiedCell = verifiedStyle.Sprint("✓ verified")
					case "partial":
						verifiedCell = color.New(color.FgYellow).Sprint("⚠ partial")
					case "failed":
						verifiedCell = notVerifiedStyle.Sprint("✗ failed")
					default:
						verifiedCell = notVerifiedStyle.Sprint("✗ not verified")
					}
				}

				// Timestamp
				timestampCell := timestampStyle.Sprint(timestamp)

				t.AppendRow(table.Row{
					continuationPrefix + " " + contractCell, // Add tree continuation and single space
					addressCell,
					verifiedCell,
					timestampCell,
				})
			}

			fmt.Print(t.Render())
			fmt.Println() // Extra newline for spacing between sections
			if !isLastNetwork {
				fmt.Println(continuationPrefix) // Continue the tree line
			} else {
				fmt.Println()
			}
		}
	}

	return nil
}

// calculateColumnWidths calculates the max width needed for each column across all deployments
func calculateColumnWidths(deployments []*registry.DeploymentInfo) columnWidths {
	widths := columnWidths{
		contract:  20, // Minimum width
		address:   42, // Fixed for addresses
		verified:  15, // Fixed for "⧖ deploy queued"
		timestamp: 19, // Fixed for timestamp format
		status:    0,  // Not used anymore
	}

	// Calculate max contract name width (including tags)
	for _, deployment := range deployments {
		displayName := deployment.Entry.GetDisplayName()
		contractLen := len(displayName) + 3 // +3 for tree prefix and space ("│ " or "  ")
		if len(deployment.Entry.Tags) > 0 {
			// Add space for tag like " (v1.0.0)"
			contractLen += len(deployment.Entry.Tags[0]) + 3
		}
		if contractLen > widths.contract {
			widths.contract = contractLen
		}
	}

	// Cap the contract column at a reasonable max
	if widths.contract > 50 {
		widths.contract = 50
	}

	return widths
}

// filterDeployments applies the command-line filters to the deployment list
func filterDeployments(deployments []*registry.DeploymentInfo) []*registry.DeploymentInfo {
	if filterEnv == "" && filterNetwork == "" && filterContract == "" {
		return deployments
	}
	
	filtered := make([]*registry.DeploymentInfo, 0)
	
	for _, deployment := range deployments {
		// Filter by environment (exact match, case-insensitive)
		if filterEnv != "" && !strings.EqualFold(deployment.Entry.Environment, filterEnv) {
			continue
		}
		
		// Filter by network (exact match, case-insensitive)
		if filterNetwork != "" && !strings.EqualFold(deployment.NetworkName, filterNetwork) {
			continue
		}
		
		// Filter by contract (partial match, case-insensitive)
		if filterContract != "" {
			contractName := deployment.Entry.ContractName
			if !strings.Contains(strings.ToLower(contractName), strings.ToLower(filterContract)) {
				continue
			}
		}
		
		filtered = append(filtered, deployment)
	}
	
	return filtered
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

// listLibraries displays all deployed libraries
func listLibraries() error {
	// Initialize registry manager
	registryManager, err := registry.NewManager("deployments.json")
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	libraries := registryManager.GetAllLibraries()
	
	if len(libraries) == 0 {
		fmt.Println("No libraries found")
		return nil
	}

	// Create library info structure
	type LibraryInfo struct {
		Name    string
		Entry   *types.DeploymentEntry
		ChainID uint64
		Address common.Address
	}

	// Group libraries by chain
	librariesByChain := make(map[uint64][]*LibraryInfo)
	chains := make([]uint64, 0)
	allLibraries := make([]*LibraryInfo, 0)
	
	// Parse library keys (format: "chainID-libraryName")
	for key, entry := range libraries {
		parts := strings.Split(key, "-")
		if len(parts) < 2 {
			continue
		}
		
		chainID, err := parseUint64(parts[0])
		if err != nil {
			continue
		}
		
		libraryName := strings.Join(parts[1:], "-") // Handle library names with dashes
		
		libInfo := &LibraryInfo{
			Name:    libraryName,
			Entry:   entry,
			ChainID: chainID,
			Address: entry.Address,
		}
		
		allLibraries = append(allLibraries, libInfo)
		
		if !slices.Contains(chains, chainID) {
			chains = append(chains, chainID)
		}
		librariesByChain[chainID] = append(librariesByChain[chainID], libInfo)
	}
	
	// Sort chains
	sort.Slice(chains, func(i, j int) bool {
		return chains[i] < chains[j]
	})

	// Create color styles
	chainBg := color.BgCyan
	chainHeader := color.New(chainBg, color.FgBlack)
	chainHeaderBold := color.New(chainBg, color.FgBlack, color.Bold)
	libraryNameStyle := color.New(color.Bold)
	addressStyle := color.New(color.Bold, color.FgHiWhite)
	timestampStyle := color.New(color.Faint)
	foundryStyle := color.New(color.FgCyan)

	fmt.Printf("Libraries (%d total):\n\n", len(libraries))

	for _, chainID := range chains {
		chainLibs := librariesByChain[chainID]
		
		// Sort libraries by timestamp (newest first)
		sort.Slice(chainLibs, func(i, j int) bool {
			return chainLibs[i].Entry.Deployment.Timestamp.After(chainLibs[j].Entry.Deployment.Timestamp)
		})
		
		// Create table
		t := table.NewWriter()
		t.SetStyle(table.StyleLight)
		t.Style().Options.SeparateRows = false
		t.Style().Options.DrawBorder = false
		t.Style().Options.SeparateHeader = false
		t.Style().Options.SeparateColumns = false
		t.Style().Box = table.BoxStyle{
			PaddingLeft:  "",
			PaddingRight: "  ",
		}

		// Configure column styles
		t.SetColumnConfigs([]table.ColumnConfig{
			{Number: 1, Align: text.AlignLeft, WidthMin: 20, WidthMax: 30},
			{Number: 2, Align: text.AlignLeft, WidthMin: 42, WidthMax: 42},
			{Number: 3, Align: text.AlignLeft, WidthMin: 50, WidthMax: 80},
			{Number: 4, Align: text.AlignLeft, WidthMin: 19, WidthMax: 19},
		})

		// Add chain header
		chainName := getChainName(chainID)
		chainHeaderRow := fmt.Sprintf("%s%s",
			chainHeader.Sprint(" ⛓ chain       "),
			chainHeaderBold.Sprintf(" %s ", strings.ToUpper(chainName)))
		
		fmt.Println(chainHeaderRow)
		fmt.Println()

		for _, lib := range chainLibs {
			libraryName := lib.Name
			timestamp := lib.Entry.Deployment.Timestamp.Format("2006-01-02 15:04:05")
			
			// Build library name cell
			libraryCell := libraryNameStyle.Sprint(libraryName)
			
			// Address cell
			addressCell := addressStyle.Sprint(lib.Address.Hex())
			
			// Foundry.toml format
			foundryCell := foundryStyle.Sprintf("\"src/%s.sol:%s:%s\"", libraryName, libraryName, lib.Address.Hex())
			
			// Timestamp
			timestampCell := timestampStyle.Sprint(timestamp)
			
			t.AppendRow(table.Row{
				"  " + libraryCell,
				addressCell,
				foundryCell,
				timestampCell,
			})
		}
		
		fmt.Print(t.Render())
		fmt.Println()
		fmt.Println()
	}
	
	// Show foundry.toml configuration tip
	fmt.Println("To use these libraries, add the library entries to your foundry.toml:")
	color.New(color.FgCyan).Println("[profile.default]")
	color.New(color.FgCyan).Println("libraries = [")
	for _, lib := range allLibraries {
		color.New(color.FgCyan).Printf("  \"src/%s.sol:%s:%s\",\n", lib.Name, lib.Name, lib.Address.Hex())
	}
	color.New(color.FgCyan).Println("]")
	
	return nil
}

// getChainName returns the chain name for a given chain ID
func getChainName(chainID uint64) string {
	// Common chain names
	chainNames := map[uint64]string{
		1:        "mainnet",
		5:        "goerli",
		11155111: "sepolia",
		17000:    "holesky",
		10:       "optimism",
		420:      "optimism-goerli",
		11155420: "optimism-sepolia",
		42161:    "arbitrum",
		421613:   "arbitrum-goerli",
		421614:   "arbitrum-sepolia",
		137:      "polygon",
		80001:    "mumbai",
		80002:    "amoy",
		8453:     "base",
		84531:    "base-goerli",
		84532:    "base-sepolia",
		43114:    "avalanche",
		43113:    "fuji",
		56:       "bsc",
		97:       "bsc-testnet",
		100:      "gnosis",
		10200:    "chiado",
		42220:    "celo",
		44787:    "alfajores",
		62320:    "baklava",
		534351:   "scroll-sepolia",
		534352:   "scroll",
	}
	
	if name, ok := chainNames[chainID]; ok {
		return name
	}
	
	return fmt.Sprintf("chain-%d", chainID)
}

// parseUint64 parses a string to uint64
func parseUint64(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}
