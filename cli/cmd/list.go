package cmd

import (
	"crypto/ecdsa"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

var (
	filterEnv      string
	filterNetwork  string
	filterContract string
	showLibraries  bool
)

// Package-level color styles
var (
	envBg              = color.BgYellow
	chainBg            = color.BgCyan
	envHeader          = color.New(envBg, color.FgBlack)
	envHeaderBold      = color.New(envBg, color.FgBlack, color.Bold)
	chainHeader        = color.New(chainBg, color.FgBlack)
	chainHeaderBold    = color.New(chainBg, color.FgBlack, color.Bold)
	addressStyle       = color.New(color.FgWhite)
	timestampStyle     = color.New(color.Faint)
	pendingStyle       = color.New(color.FgYellow)
	tagsStyle          = color.New(color.FgCyan)
	verifiedStyle      = color.New(color.FgGreen)
	notVerifiedStyle   = color.New(color.FgRed)
	sectionHeaderStyle = color.New(color.Bold, color.FgHiWhite)
	implPrefixStyle    = color.New(color.Faint)

	// Library-specific styles
	libraryNameStyle    = color.New(color.Bold)
	libraryAddressStyle = color.New(color.Bold, color.FgHiWhite)
	foundryStyle        = color.New(color.FgCyan)
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

	fmt.Printf("Deployments (%d total):\n\n", len(deployments))

	// Calculate global column widths for all deployments
	globalWidths := calculateColumnWidthsForRows(deployments)

	// Group deployments and display them
	if err := displayDeployments(deployments, deployConfig, globalWidths); err != nil {
		return err
	}

	return nil
}

func displayDeployments(deployments []*registry.DeploymentInfo, deployConfig *config.DeployConfig, globalWidths columnWidths) error {

	groups := make(map[string]map[string][]*registry.DeploymentInfo)
	envs := make([]string, 0)
	networks := make([]string, 0)

	for _, deployment := range deployments {
		env := deployment.Entry.Environment
		network := deployment.NetworkName

		if !slices.Contains(networks, network) {
			networks = append(networks, network)
		}

		if !slices.Contains(envs, env) {
			envs = append(envs, env)
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

		// Environment header
		tableWidth := globalWidths.contract + globalWidths.address + globalWidths.verified + globalWidths.timestamp + 2
		envHeaderPrefix := envHeader.Sprintf("   ◎ environment  %s", envHeaderBold.Sprint(strings.ToUpper(env)))
		envHeader.Printf(
			"%s%s%s\n",
			envHeaderPrefix,
			envHeader.Sprintf("%*s", tableWidth-len(envHeaderPrefix), "deployer:"),
			envHeaderBold.Sprint(deployerAddress),
		)

		// Filter networks to only show those with deployments for this env
		networksWithDeployments := []string{}
		for _, network := range networks {
			if len(groups[network][env]) > 0 {
				networksWithDeployments = append(networksWithDeployments, network)
			}
		}

		for netIdx, network := range networksWithDeployments {
			deployments := groups[network][env]

			// Determine if this is the last network for tree drawing
			isLastNetwork := netIdx == len(networksWithDeployments)-1
			treePrefix := "├─"
			continuationPrefix := "│ "
			if isLastNetwork {
				treePrefix = "└─"
				continuationPrefix = "  "
			}

			// Chain header
			chainHeaderRow := fmt.Sprintf("%s%s%s",
				treePrefix,
				chainHeader.Sprint(" ⛓ chain       "),
				chainHeaderBold.Sprintf(" %s ", strings.ToUpper(network)))

			fmt.Println(chainHeaderRow)
			fmt.Println(continuationPrefix)

			// Separate proxies and singletons within this env/network group
			proxies := make([]*registry.DeploymentInfo, 0)
			singletons := make([]*registry.DeploymentInfo, 0)

			for _, deployment := range deployments {
				if deployment.Entry.Type == "proxy" {
					proxies = append(proxies, deployment)
				} else {
					singletons = append(singletons, deployment)
				}
			}

			// Display proxies section first
			if len(proxies) > 0 {
				fmt.Printf("%s%s\n", continuationPrefix, sectionHeaderStyle.Sprint("PROXIES"))
				displayDeploymentRows(proxies, globalWidths, continuationPrefix)
			}

			// Display singletons section
			if len(singletons) > 0 {
				if len(proxies) > 0 {
					fmt.Println(continuationPrefix)
				}
				fmt.Printf("%s%s\n", continuationPrefix, sectionHeaderStyle.Sprint("SINGLETONS"))
				displayDeploymentRows(singletons, globalWidths, continuationPrefix)
			}

			if !isLastNetwork {
				fmt.Println(continuationPrefix)
			} else {
				fmt.Println()
			}
		}
	}

	return nil
}

func displayDeploymentRows(deployments []*registry.DeploymentInfo, widths columnWidths, continuationPrefix string) {

	// Sort deployments by timestamp (newest first)
	sort.Slice(deployments, func(i, j int) bool {
		return deployments[i].Entry.Deployment.Timestamp.After(deployments[j].Entry.Deployment.Timestamp)
	})

	// Create table
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.Style().Options.SeparateRows = false
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateHeader = false
	t.Style().Options.SeparateColumns = false
	t.Style().Box = table.BoxStyle{
		PaddingRight: "   ",
	}

	// Configure column styles with calculated widths
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, Align: text.AlignLeft, WidthMin: widths.contract, WidthMax: widths.contract},
		{Number: 2, Align: text.AlignLeft, WidthMin: widths.address, WidthMax: widths.address},
		{Number: 3, Align: text.AlignLeft, WidthMin: widths.verified, WidthMax: widths.verified},
		{Number: 4, Align: text.AlignLeft, WidthMin: widths.timestamp, WidthMax: widths.timestamp},
	})

	for _, deployment := range deployments {
		// Add main deployment row
		coloredDisplayName := deployment.Entry.GetColoredDisplayName()
		timestamp := deployment.Entry.Deployment.Timestamp.Format("2006-01-02 15:04:05")

		contractCell := coloredDisplayName
		if len(deployment.Entry.Tags) > 0 {
			contractCell += " " + tagsStyle.Sprintf("(%s)", deployment.Entry.Tags[0])
		}

		addressCell := addressStyle.Sprint(deployment.Address.Hex())

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

		timestampCell := timestampStyle.Sprint(timestamp)

		t.AppendRow(table.Row{
			continuationPrefix + contractCell, // 6 spaces for alignment
			addressCell,
			verifiedCell,
			timestampCell,
		})

		// If this is a proxy and we're showing implementations, add implementation row
		if deployment.Entry.Type == "proxy" && deployment.Entry.Target != nil {
			implDisplayName := deployment.Entry.Target.GetColoredDisplayName()
			implTimestamp := deployment.Entry.Target.Deployment.Timestamp.Format("2006-01-02 15:04:05")

			// Build implementation row with └─ prefix
			implContractCell := implPrefixStyle.Sprint("└─ ") + implDisplayName
			if len(deployment.Entry.Target.Tags) > 0 {
				implContractCell += " " + tagsStyle.Sprintf("(%s)", deployment.Entry.Target.Tags[0])
			}

			implAddressCell := addressStyle.Sprint(deployment.Entry.Target.Address.Hex())

			implVerifiedCell := ""
			switch deployment.Entry.Target.Verification.Status {
			case "verified":
				implVerifiedCell = verifiedStyle.Sprint("✓ verified")
			case "partial":
				implVerifiedCell = color.New(color.FgYellow).Sprint("⚠ partial")
			case "failed":
				implVerifiedCell = notVerifiedStyle.Sprint("✗ failed")
			default:
				implVerifiedCell = notVerifiedStyle.Sprint("✗ not verified")
			}

			implTimestampCell := timestampStyle.Sprint(implTimestamp)

			t.AppendRow(table.Row{
				continuationPrefix + " " + implContractCell,
				implAddressCell,
				implVerifiedCell,
				implTimestampCell,
			})
		}
	}

	fmt.Print(t.Render())
	fmt.Println()
}

// calculateColumnWidthsForRows calculates widths based on actual rendered content
func calculateColumnWidthsForRows(deployments []*registry.DeploymentInfo) columnWidths {
	widths := columnWidths{
		contract:  20, // Minimum width
		address:   42, // Fixed for addresses
		verified:  15, // Fixed for "⧖ deploy queued"
		timestamp: 19, // Fixed for timestamp format
	}

	// Calculate max contract name width based on actual rendered content
	for _, deployment := range deployments {
		// Main deployment row: "      " + displayName + optional tag
		displayName := deployment.Entry.GetDisplayName()
		contractLen := len(displayName) // 6 spaces + display name
		if len(deployment.Entry.Tags) > 0 {
			contractLen += len(deployment.Entry.Tags[0]) + 3 // " (tag)"
		}
		if contractLen > widths.contract {
			widths.contract = contractLen
		}

		if deployment.Entry.Type == "proxy" && deployment.Entry.Target != nil {
			// Implementation row: "        └─ " + displayName + optional tag
			implContractLen := 0

			// Try to get implementation display name
			if deployment.Entry.Target != nil {
				implDisplayName := deployment.Entry.Target.GetDisplayName()
				implContractLen += len(implDisplayName)
				if len(deployment.Entry.Target.Tags) > 0 {
					implContractLen += len(deployment.Entry.Target.Tags[0]) + 3 // " (tag)"
				}
				if implContractLen > widths.contract {
					widths.contract = implContractLen
				}
			}
		}
	}

	// Cap the contract column at a reasonable max
	if widths.contract > 70 {
		widths.contract = 70
	}

	return widths
}

// filterDeployments applies the command-line filters to the deployment list
func filterDeployments(deployments []*registry.DeploymentInfo) []*registry.DeploymentInfo {
	filtered := make([]*registry.DeploymentInfo, 0)

	for _, deployment := range deployments {
		// Always filter out libraries from regular deployment list
		if deployment.Entry.Type == types.LibraryDeployment {
			continue
		}

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

	// Get all deployments and filter for libraries
	allDeployments := registryManager.GetAllDeployments()
	libraries := make([]*registry.DeploymentInfo, 0)
	
	for _, deployment := range allDeployments {
		if deployment.Entry.Type == types.LibraryDeployment {
			libraries = append(libraries, deployment)
		}
	}

	if len(libraries) == 0 {
		fmt.Println("No libraries found")
		return nil
	}

	// Group libraries by chain
	librariesByChain := make(map[string][]*registry.DeploymentInfo)
	chains := make([]string, 0)

	for _, lib := range libraries {
		chainName := lib.NetworkName
		
		if !slices.Contains(chains, chainName) {
			chains = append(chains, chainName)
		}
		librariesByChain[chainName] = append(librariesByChain[chainName], lib)
	}

	// Sort chains
	sort.Slice(chains, func(i, j int) bool {
		return chains[i] < chains[j]
	})

	// Use package-level color styles

	fmt.Printf("Libraries (%d total):\n\n", len(libraries))

	for _, chainName := range chains {
		chainLibs := librariesByChain[chainName]

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
			{Number: 3, Align: text.AlignLeft, WidthMin: 19, WidthMax: 19},
		})

		// Add chain header
		chainHeaderRow := fmt.Sprintf("%s%s",
			chainHeader.Sprint(" ⛓ chain       "),
			chainHeaderBold.Sprintf(" %s ", strings.ToUpper(chainName)))

		fmt.Println(chainHeaderRow)
		fmt.Println()

		for _, lib := range chainLibs {
			libraryName := lib.Entry.ContractName
			timestamp := lib.Entry.Deployment.Timestamp.Format("2006-01-02 15:04:05")

			// Build library name cell
			libraryCell := libraryNameStyle.Sprint(libraryName)

			// Address cell
			addressCell := libraryAddressStyle.Sprint(lib.Entry.Address.Hex())

			// Timestamp
			timestampCell := timestampStyle.Sprint(timestamp)

			t.AppendRow(table.Row{
				"  " + libraryCell,
				addressCell,
				timestampCell,
			})
		}

		fmt.Print(t.Render())
		fmt.Println()
		fmt.Println()
	}

	// Show library usage information
	fmt.Println("ℹ️  Library Usage Information:")
	fmt.Println("• Treb automatically injects library addresses on demand for contracts that need them")
	fmt.Println("• No manual configuration needed in foundry.toml")
	fmt.Println()
	fmt.Println("⚠️  Warning: Adding libraries to foundry.toml will:")
	fmt.Println("• Include library addresses in metadata of ALL contracts (even those that don't use them)")
	fmt.Println("• Change the compilation bytecode hash")
	fmt.Println("• Potentially cause verification difficulties")
	fmt.Println()
	fmt.Println("If you still need to add them to foundry.toml:")
	color.New(color.FgCyan).Println("[profile.default]")
	color.New(color.FgCyan).Println("libraries = [")
	for _, lib := range libraries {
		color.New(color.FgCyan).Printf("  \"src/%s.sol:%s:%s\",\n", lib.Entry.ContractName, lib.Entry.ContractName, lib.Entry.Address.Hex())
	}
	color.New(color.FgCyan).Println("]")

	return nil
}
