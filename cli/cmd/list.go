package cmd

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

var (
	filterNamespace string
	filterNetwork  string
	filterContract string
	showLibraries  bool
)

// Package-level color styles
var (
	nsBg               = color.BgYellow
	chainBg            = color.BgCyan
	nsHeader           = color.New(nsBg, color.FgBlack)
	nsHeaderBold       = color.New(nsBg, color.FgBlack, color.Bold)
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
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all deployments",
	Long: `Display deployments organized by namespace and network.
Shows contract addresses, deployment status, and version tags.

Filters:
  --filter-ns       Filter by namespace (exact match, case-insensitive)
  --filter-network  Filter by network (exact match, case-insensitive)
  --filter-contract Filter by contract name (partial match, case-insensitive)
  --libraries       Show deployed libraries instead of contracts`,
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
	listCmd.Flags().StringVarP(&filterNamespace, "filter-ns", "n", "", "Filter by namespace")
	listCmd.Flags().StringVar(&filterNetwork, "filter-network", "", "Filter by network")
	listCmd.Flags().StringVar(&filterContract, "filter-contract", "", "Filter by contract name (partial match)")
	listCmd.Flags().BoolVar(&showLibraries, "libraries", false, "Show deployed libraries instead of contracts")
}

// TableData represents a table as a 2D array of strings
type TableData [][]string

// TableSet represents multiple tables that need to be rendered together
type TableSet struct {
	Tables       []TableData
	ColumnWidths []int
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

	// Group deployments and display them
	if err := displayDeployments(deployments, deployConfig); err != nil {
		return err
	}

	return nil
}

func displayDeployments(deployments []*registry.DeploymentInfo, deployConfig *config.DeployConfig) error {

	groups := make(map[string]map[string][]*registry.DeploymentInfo)
	namespaces := make([]string, 0)
	networks := make([]string, 0)

	for _, deployment := range deployments {
		ns := deployment.Entry.Namespace
		network := deployment.NetworkName

		if !slices.Contains(networks, network) {
			networks = append(networks, network)
		}

		if !slices.Contains(namespaces, ns) {
			namespaces = append(namespaces, ns)
		}

		if groups[deployment.NetworkName] == nil {
			groups[deployment.NetworkName] = make(map[string][]*registry.DeploymentInfo)
		}

		groups[deployment.NetworkName][ns] = append(groups[deployment.NetworkName][ns], deployment)
	}

	slices.Sort(namespaces)
	slices.Sort(networks)

	// Build all tables first to calculate consistent column widths
	allTables := make([]TableData, 0)
	for _, ns := range namespaces {
		for _, network := range networks {
			if len(groups[network][ns]) == 0 {
				continue
			}
			deployments := groups[network][ns]

			// Separate proxies and singletons within this namespace/network group
			proxies := make([]*registry.DeploymentInfo, 0)
			singletons := make([]*registry.DeploymentInfo, 0)

			for _, deployment := range deployments {
				if deployment.Entry.Type == types.ProxyDeployment {
					proxies = append(proxies, deployment)
				} else {
					singletons = append(singletons, deployment)
				}
			}

			// Build tables for proxies and singletons
			if len(proxies) > 0 {
				proxyTable := buildDeploymentTable(proxies)
				allTables = append(allTables, proxyTable)
			}
			if len(singletons) > 0 {
				singletonTable := buildDeploymentTable(singletons)
				allTables = append(allTables, singletonTable)
			}
		}
	}

	// Calculate global column widths for all tables
	globalColumnWidths := calculateTableColumnWidths(allTables)

	// Display groups
	for _, ns := range namespaces {

		// Namespace header
		nsLabel := fmt.Sprintf("%-12s", "namespace:")
		nsValue := fmt.Sprintf("%-30s", strings.ToUpper(ns))
		nsHeaderPrefix := nsHeader.Sprintf("   ◎ %s %s", nsLabel, nsHeaderBold.Sprint(nsValue))
		// For now, we won't show sender in the header
		nsHeader.Printf("%s\n", nsHeaderPrefix)

		// Filter networks to only show those with deployments for this namespace
		networksWithDeployments := []string{}
		for _, network := range networks {
			if len(groups[network][ns]) > 0 {
				networksWithDeployments = append(networksWithDeployments, network)
			}
		}

		for netIdx, network := range networksWithDeployments {
			deployments := groups[network][ns]

			// Determine if this is the last network for tree drawing
			isLastNetwork := netIdx == len(networksWithDeployments)-1
			treePrefix := "├─"
			continuationPrefix := "│ "
			if isLastNetwork {
				treePrefix = "└─"
				continuationPrefix = "  "
			}

			// Chain header
			chainLabel := fmt.Sprintf("%-12s", "chain:")
			chainValue := fmt.Sprintf("%-30s", strings.ToUpper(network))
			chainHeaderRow := fmt.Sprintf("%s%s%s",
				treePrefix,
				chainHeader.Sprintf(" ⛓ %s ", chainLabel),
				chainHeaderBold.Sprint(chainValue))

			fmt.Println(chainHeaderRow)
			fmt.Println(continuationPrefix)

			// Separate proxies and singletons within this namespace/network group
			proxies := make([]*registry.DeploymentInfo, 0)
			singletons := make([]*registry.DeploymentInfo, 0)

			for _, deployment := range deployments {
				if deployment.Entry.Type == types.ProxyDeployment {
					proxies = append(proxies, deployment)
				} else {
					singletons = append(singletons, deployment)
				}
			}

			// Display proxies section first
			if len(proxies) > 0 {
				fmt.Printf("%s%s\n", continuationPrefix, sectionHeaderStyle.Sprint("PROXIES"))
				proxyTable := buildDeploymentTable(proxies)
				fmt.Print(renderTableWithWidths(proxyTable, globalColumnWidths, continuationPrefix))
				fmt.Println()
			}

			// Display singletons section
			if len(singletons) > 0 {
				if len(proxies) > 0 {
					fmt.Println(continuationPrefix)
				}
				fmt.Printf("%s%s\n", continuationPrefix, sectionHeaderStyle.Sprint("SINGLETONS"))
				singletonTable := buildDeploymentTable(singletons)
				fmt.Print(renderTableWithWidths(singletonTable, globalColumnWidths, continuationPrefix))
				fmt.Println()
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

// filterDeployments applies the command-line filters to the deployment list
func filterDeployments(deployments []*registry.DeploymentInfo) []*registry.DeploymentInfo {
	filtered := make([]*registry.DeploymentInfo, 0)

	for _, deployment := range deployments {
		// Always filter out libraries from regular deployment list
		if deployment.Entry.Type == types.LibraryDeployment {
			continue
		}

		// Filter by namespace (exact match, case-insensitive)
		if filterNamespace != "" && !strings.EqualFold(deployment.Entry.Namespace, filterNamespace) {
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

	// Build all library tables first to calculate consistent column widths
	allLibraryTables := make([]TableData, 0)
	for _, chainName := range chains {
		chainLibs := librariesByChain[chainName]
		libraryTable := buildLibraryTable(chainLibs)
		allLibraryTables = append(allLibraryTables, libraryTable)
	}

	// Calculate global column widths for all library tables
	globalLibraryColumnWidths := calculateTableColumnWidths(allLibraryTables)

	fmt.Printf("Libraries (%d total):\n\n", len(libraries))

	for _, chainName := range chains {
		chainLibs := librariesByChain[chainName]

		// Add chain header
		chainHeaderRow := fmt.Sprintf("%s%s",
			chainHeader.Sprint(" ⛓ chain       "),
			chainHeaderBold.Sprintf(" %s ", strings.ToUpper(chainName)))

		fmt.Println(chainHeaderRow)
		fmt.Println()

		// Build and render table
		libraryTable := buildLibraryTable(chainLibs)
		fmt.Print(renderTableWithWidths(libraryTable, globalLibraryColumnWidths, ""))
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

// stripAnsiEscapes removes ANSI escape sequences to get the actual string length
func stripAnsiEscapes(s string) string {
	// Regex to match ANSI escape sequences
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[mGKHF]`)
	return ansiRegex.ReplaceAllString(s, "")
}

// calculateTableColumnWidths calculates column widths for multiple tables
func calculateTableColumnWidths(tables []TableData) []int {
	if len(tables) == 0 {
		return nil
	}

	// Find the maximum number of columns across all tables
	maxCols := 0
	for _, table := range tables {
		for _, row := range table {
			if len(row) > maxCols {
				maxCols = len(row)
			}
		}
	}

	widths := make([]int, maxCols)

	// Calculate maximum width for each column across all tables
	for _, table := range tables {
		for _, row := range table {
			for colIdx, cell := range row {
				// Strip ANSI escape sequences and calculate actual length
				actualLength := len(stripAnsiEscapes(cell))
				if actualLength > widths[colIdx] {
					widths[colIdx] = actualLength
				}
			}
		}
	}

	widths[0] += 2
	return widths
}

// buildDeploymentTable creates a TableData for a list of deployments
func buildDeploymentTable(deployments []*registry.DeploymentInfo) TableData {
	tableData := make(TableData, 0)

	// Sort deployments by timestamp (newest first)
	sort.Slice(deployments, func(i, j int) bool {
		return deployments[i].Entry.Deployment.Timestamp.After(deployments[j].Entry.Deployment.Timestamp)
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

		tableData = append(tableData, []string{
			contractCell,
			addressCell,
			verifiedCell,
			timestampCell,
		})

		// If this is a proxy and we're showing implementations, add implementation row
		if deployment.Entry.Type == types.ProxyDeployment && deployment.Entry.Target != nil {
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

			tableData = append(tableData, []string{
				implContractCell,
				implAddressCell,
				implVerifiedCell,
				implTimestampCell,
			})
		}
	}

	return tableData
}

// buildLibraryTable creates a TableData for a list of libraries
func buildLibraryTable(libraries []*registry.DeploymentInfo) TableData {
	tableData := make(TableData, 0)

	// Sort libraries by timestamp (newest first)
	sort.Slice(libraries, func(i, j int) bool {
		return libraries[i].Entry.Deployment.Timestamp.After(libraries[j].Entry.Deployment.Timestamp)
	})

	for _, lib := range libraries {
		libraryName := lib.Entry.ContractName
		timestamp := lib.Entry.Deployment.Timestamp.Format("2006-01-02 15:04:05")

		// Build library name cell
		libraryCell := libraryNameStyle.Sprint(libraryName)

		// Address cell
		addressCell := libraryAddressStyle.Sprint(lib.Entry.Address.Hex())

		// Timestamp
		timestampCell := timestampStyle.Sprint(timestamp)

		tableData = append(tableData, []string{
			"  " + libraryCell,
			addressCell,
			timestampCell,
		})
	}

	return tableData
}

// renderTableWithWidths renders a TableData using the go-pretty library with specified column widths
func renderTableWithWidths(tableData TableData, columnWidths []int, continuationPrefix string) string {
	if len(tableData) == 0 {
		return ""
	}

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
	colConfigs := make([]table.ColumnConfig, len(columnWidths))
	for i, width := range columnWidths {
		colConfigs[i] = table.ColumnConfig{
			Number:   i + 1,
			Align:    text.AlignLeft,
			WidthMin: width,
			WidthMax: width,
		}
	}
	t.SetColumnConfigs(colConfigs)

	// Add all rows to the table
	for _, row := range tableData {
		// Convert []string to table.Row
		tableRow := make(table.Row, len(row))
		for i, cell := range row {
			if i == 0 {
				tableRow[i] = continuationPrefix + cell
			} else {
				tableRow[i] = cell
			}
		}
		t.AppendRow(tableRow)
	}

	return t.Render()
}

