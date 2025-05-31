package cmd

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// Color styles for table format
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
)

type TableData [][]string

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List deployments from registry",
	Long:    `List all deployments in the registry, organized by namespace and chain.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get flags
		namespace, _ := cmd.Flags().GetString("namespace")
		chainID, _ := cmd.Flags().GetUint64("chain")
		contractName, _ := cmd.Flags().GetString("contract")

		// Create registry manager
		manager, err := registry.NewManager(".")
		if err != nil {
			checkError(fmt.Errorf("failed to load registry: %w", err))
		}

		// Get all deployments
		allDeployments := manager.GetAllDeployments()

		// Filter deployments
		var deployments []*types.Deployment
		for _, dep := range allDeployments {
			// Apply filters
			if namespace != "" && dep.Namespace != namespace {
				continue
			}
			if chainID != 0 && dep.ChainID != chainID {
				continue
			}
			if contractName != "" && dep.ContractName != contractName {
				continue
			}
			deployments = append(deployments, dep)
		}

		// Sort by namespace, chain, contract name, label
		sort.Slice(deployments, func(i, j int) bool {
			if deployments[i].Namespace != deployments[j].Namespace {
				return deployments[i].Namespace < deployments[j].Namespace
			}
			if deployments[i].ChainID != deployments[j].ChainID {
				return deployments[i].ChainID < deployments[j].ChainID
			}
			if deployments[i].ContractName != deployments[j].ContractName {
				return deployments[i].ContractName < deployments[j].ContractName
			}
			return deployments[i].Label < deployments[j].Label
		})

		// Display results
		if len(deployments) == 0 {
			fmt.Println("No deployments found")
			return
		}

		displayTableFormat(deployments, manager)
	},
}

// displayTableFormat shows deployments in table format
func displayTableFormat(deployments []*types.Deployment, manager *registry.Manager) {
	// Group by namespace and chain
	namespaceChainGroups := make(map[string]map[uint64][]*types.Deployment)

	for _, dep := range deployments {
		if namespaceChainGroups[dep.Namespace] == nil {
			namespaceChainGroups[dep.Namespace] = make(map[uint64][]*types.Deployment)
		}
		namespaceChainGroups[dep.Namespace][dep.ChainID] = append(namespaceChainGroups[dep.Namespace][dep.ChainID], dep)
	}

	// Get sorted namespaces
	namespaces := make([]string, 0, len(namespaceChainGroups))
	for ns := range namespaceChainGroups {
		namespaces = append(namespaces, ns)
	}
	sort.Strings(namespaces)

	// Build all tables for consistent column width calculation
	var allTables []TableData
	for _, ns := range namespaces {
		chains := namespaceChainGroups[ns]
		chainIDs := make([]uint64, 0, len(chains))
		for chainID := range chains {
			chainIDs = append(chainIDs, chainID)
		}
		sort.Slice(chainIDs, func(i, j int) bool { return chainIDs[i] < chainIDs[j] })

		for _, chainID := range chainIDs {
			chainDeployments := chains[chainID]

			// Separate proxies and singletons
			proxies := make([]*types.Deployment, 0)
			singletons := make([]*types.Deployment, 0)

			for _, dep := range chainDeployments {
				if dep.Type == types.ProxyDeployment {
					proxies = append(proxies, dep)
				} else {
					singletons = append(singletons, dep)
				}
			}

			if len(proxies) > 0 {
				proxyTable := buildDeploymentTable(proxies, manager)
				allTables = append(allTables, proxyTable)
			}
			if len(singletons) > 0 {
				singletonTable := buildDeploymentTable(singletons, manager)
				allTables = append(allTables, singletonTable)
			}
		}
	}

	// Calculate global column widths
	globalColumnWidths := calculateTableColumnWidths(allTables)

	// Display groups
	for _, ns := range namespaces {
		// Namespace header
		nsLabel := fmt.Sprintf("%-12s", "namespace:")
		nsValue := fmt.Sprintf("%-30s", strings.ToUpper(ns))
		nsHeaderPrefix := nsHeader.Sprintf("   ◎ %s %s", nsLabel, nsHeaderBold.Sprint(nsValue))
		nsHeader.Printf("%s\n", nsHeaderPrefix)

		chains := namespaceChainGroups[ns]
		chainIDs := make([]uint64, 0, len(chains))
		for chainID := range chains {
			chainIDs = append(chainIDs, chainID)
		}
		sort.Slice(chainIDs, func(i, j int) bool { return chainIDs[i] < chainIDs[j] })

		for netIdx, chainID := range chainIDs {
			chainDeployments := chains[chainID]

			// Determine if this is the last network for tree drawing
			isLastNetwork := netIdx == len(chainIDs)-1
			treePrefix := "├─"
			continuationPrefix := "│ "
			if isLastNetwork {
				treePrefix = "└─"
				continuationPrefix = "  "
			}

			// Chain header
			chainLabel := fmt.Sprintf("%-12s", "chain:")
			chainValue := fmt.Sprintf("%-30s", fmt.Sprintf("%d", chainID))
			chainHeaderRow := fmt.Sprintf("%s%s%s",
				treePrefix,
				chainHeader.Sprintf(" ⛓ %s ", chainLabel),
				chainHeaderBold.Sprint(chainValue))

			fmt.Println(chainHeaderRow)
			fmt.Println(continuationPrefix)

			// Separate proxies and singletons
			proxies := make([]*types.Deployment, 0)
			singletons := make([]*types.Deployment, 0)

			for _, dep := range chainDeployments {
				if dep.Type == types.ProxyDeployment {
					proxies = append(proxies, dep)
				} else {
					singletons = append(singletons, dep)
				}
			}

			// Display proxies section first
			if len(proxies) > 0 {
				fmt.Printf("%s%s\n", continuationPrefix, sectionHeaderStyle.Sprint("PROXIES"))
				proxyTable := buildDeploymentTable(proxies, manager)
				fmt.Print(renderTableWithWidths(proxyTable, globalColumnWidths, continuationPrefix))
				fmt.Println()
			}

			// Display singletons section
			if len(singletons) > 0 {
				if len(proxies) > 0 {
					fmt.Println(continuationPrefix)
				}
				fmt.Printf("%s%s\n", continuationPrefix, sectionHeaderStyle.Sprint("SINGLETONS"))
				singletonTable := buildDeploymentTable(singletons, manager)
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

	fmt.Printf("Total deployments: %d\n", len(deployments))
}

// buildDeploymentTable creates a TableData for a list of deployments
func buildDeploymentTable(deployments []*types.Deployment, manager *registry.Manager) TableData {
	tableData := make(TableData, 0)

	// Sort deployments by timestamp (newest first)
	sort.Slice(deployments, func(i, j int) bool {
		return deployments[i].CreatedAt.After(deployments[j].CreatedAt)
	})

	for _, deployment := range deployments {
		// Add main deployment row
		contractCell := getColoredDisplayName(deployment)
		if len(deployment.Tags) > 0 {
			contractCell += " " + tagsStyle.Sprintf("(%s)", deployment.Tags[0])
		}

		addressCell := addressStyle.Sprint(deployment.Address)

		verifiedCell := ""
		switch deployment.Verification.Status {
		case types.VerificationStatusVerified:
			verifiedCell = verifiedStyle.Sprint("✓ verified")
		case types.VerificationStatusPending:
			verifiedCell = pendingStyle.Sprint("⏳ not deployed")
		case types.VerificationStatusFailed:
			verifiedCell = notVerifiedStyle.Sprint("✗ failed")
		default:
			verifiedCell = notVerifiedStyle.Sprint("✗ not verified")
		}

		timestampCell := timestampStyle.Sprint(deployment.CreatedAt.Format("2006-01-02 15:04:05"))

		tableData = append(tableData, []string{
			contractCell,
			addressCell,
			verifiedCell,
			timestampCell,
		})

		// If this is a proxy, add implementation row
		if deployment.ProxyInfo != nil {
			implDisplayName := deployment.ProxyInfo.Implementation[:10] + "..." // fallback short address
			if implDep, err := manager.GetDeploymentByAddress(deployment.ChainID, deployment.ProxyInfo.Implementation); err == nil {
				// Show short deployment ID: namespace/contract:label
				shortID := implDep.ContractName
				if implDep.Label != "" {
					shortID += ":" + implDep.Label
				}
				implDisplayName = shortID
			}

			implRow := implPrefixStyle.Sprintf("└─ %s", implDisplayName)
			implAddress := addressStyle.Sprint(deployment.ProxyInfo.Implementation)

			tableData = append(tableData, []string{
				implRow,
				implAddress,
				"",
				"",
			})
		}
	}

	return tableData
}

// getColoredDisplayName returns a colored display name for deployment
func getColoredDisplayName(dep *types.Deployment) string {
	name := dep.ContractName
	if dep.Label != "" {
		name += ":" + dep.Label
	}

	switch dep.Type {
	case types.ProxyDeployment:
		return color.New(color.FgMagenta, color.Bold).Sprint(name)
	case types.LibraryDeployment:
		return color.New(color.FgBlue, color.Bold).Sprint(name)
	default:
		return color.New(color.FgGreen, color.Bold).Sprint(name)
	}
}

// renderTableWithWidths renders a table with specific column widths
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
		if i == 0 {
			width += 2 + len([]rune(continuationPrefix))
		}
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

// stripAnsiCodes removes ANSI escape sequences from a string
func stripAnsiCodes(s string) string {
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
				if colIdx < len(widths) {
					// Strip ANSI codes for width calculation
					cellWidth := len(stripAnsiCodes(cell))
					if cellWidth > widths[colIdx] {
						widths[colIdx] = cellWidth
					}
				}
			}
		}
	}

	return widths
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Filter flags
	listCmd.Flags().StringP("namespace", "n", "", "Filter by namespace")
	listCmd.Flags().Uint64P("chain", "c", 0, "Filter by chain ID")
	listCmd.Flags().StringP("contract", "", "", "Filter by contract name")
}
