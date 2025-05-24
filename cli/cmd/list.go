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
	"github.com/trebuchet-org/treb-cli/cli/internal/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
)

var (
	filterEnv      string
	filterNetwork  string
	filterContract string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all deployments",
	Long: `Display deployments organized by environment and network.
Shows contract addresses, deployment status, and version tags.

Filters:
  --filter-env      Filter by environment (exact match, case-insensitive)
  --filter-network  Filter by network (exact match, case-insensitive)
  --filter-contract Filter by contract name (partial match, case-insensitive)`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := listDeployments(); err != nil {
			checkError(err)
		}
	},
}

func init() {
	listCmd.Flags().StringVar(&filterEnv, "filter-env", "", "Filter by environment")
	listCmd.Flags().StringVar(&filterNetwork, "filter-network", "", "Filter by network")
	listCmd.Flags().StringVar(&filterContract, "filter-contract", "", "Filter by contract name (partial match)")
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
				} else if deployment.Entry.Verification.Status == "verified" {
					verifiedCell = verifiedStyle.Sprint("✓ verified")
				} else {
					verifiedCell = notVerifiedStyle.Sprint("✗ not verified")
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
