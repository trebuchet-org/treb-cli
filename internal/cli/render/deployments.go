package render

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
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

// DeploymentsRenderer renders deployment lists as formatted tables with tree-style layout
type DeploymentsRenderer struct {
	out   io.Writer
	color bool
}

// NewDeploymentsRenderer creates a new deployments renderer
func NewDeploymentsRenderer(out io.Writer, color bool) *DeploymentsRenderer {
	return &DeploymentsRenderer{
		out:   out,
		color: color,
	}
}

// RenderDeploymentList renders deployments in the tree-style format
func (r *DeploymentsRenderer) RenderDeploymentList(result *usecase.DeploymentListResult) error {
	if len(result.Deployments) == 0 {
		fmt.Fprintln(r.out, "No deployments found")
		return nil
	}

	// Display in tree-style table format
	r.displayTableFormat(result.Deployments)
	return nil
}

// displayTableFormat shows deployments in table format
func (r *DeploymentsRenderer) displayTableFormat(deployments []*models.Deployment) {
	// Group by namespace and chain
	namespaceChainGroups := make(map[string]map[uint64][]*models.Deployment)

	for _, dep := range deployments {
		if namespaceChainGroups[dep.Namespace] == nil {
			namespaceChainGroups[dep.Namespace] = make(map[uint64][]*models.Deployment)
		}
		namespaceChainGroups[dep.Namespace][dep.ChainID] = append(namespaceChainGroups[dep.Namespace][dep.ChainID], dep)
	}

	// Build a map of implementation addresses to determine which singletons are implementations
	implementationAddresses := make(map[string]bool)
	for _, dep := range deployments {
		if dep.Type == models.ProxyDeployment && dep.ProxyInfo != nil {
			implementationAddresses[dep.ProxyInfo.Implementation] = true
		}
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

			// Separate deployments into 4 categories
			proxies := make([]*models.Deployment, 0)
			implementations := make([]*models.Deployment, 0)
			singletons := make([]*models.Deployment, 0)
			libraries := make([]*models.Deployment, 0)

			for _, dep := range chainDeployments {
				switch dep.Type {
				case models.ProxyDeployment:
					proxies = append(proxies, dep)
				case models.LibraryDeployment:
					libraries = append(libraries, dep)
				default:
					// Check if this singleton is an implementation
					if implementationAddresses[dep.Address] {
						implementations = append(implementations, dep)
					} else {
						singletons = append(singletons, dep)
					}
				}
			}

			if len(proxies) > 0 {
				proxyTable := r.buildDeploymentTable(proxies)
				allTables = append(allTables, proxyTable)
			}
			if len(implementations) > 0 {
				implTable := r.buildDeploymentTable(implementations)
				allTables = append(allTables, implTable)
			}
			if len(singletons) > 0 {
				singletonTable := r.buildDeploymentTable(singletons)
				allTables = append(allTables, singletonTable)
			}
			if len(libraries) > 0 {
				libraryTable := r.buildDeploymentTable(libraries)
				allTables = append(allTables, libraryTable)
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
		fmt.Fprintln(r.out, nsHeaderPrefix)

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

			fmt.Fprintln(r.out, chainHeaderRow)
			fmt.Fprintln(r.out, continuationPrefix)

			// Separate deployments into 4 categories
			proxies := make([]*models.Deployment, 0)
			implementations := make([]*models.Deployment, 0)
			singletons := make([]*models.Deployment, 0)
			libraries := make([]*models.Deployment, 0)

			for _, dep := range chainDeployments {
				switch dep.Type {
				case models.ProxyDeployment:
					proxies = append(proxies, dep)
				case models.LibraryDeployment:
					libraries = append(libraries, dep)
				default:
					// Check if this singleton is an implementation
					if implementationAddresses[dep.Address] {
						implementations = append(implementations, dep)
					} else {
						singletons = append(singletons, dep)
					}
				}
			}

			sectionsDisplayed := 0

			// Display proxies section first
			if len(proxies) > 0 {
				if sectionsDisplayed > 0 {
					fmt.Fprintln(r.out, continuationPrefix)
				}
				fmt.Fprintf(r.out, "%s%s\n", continuationPrefix, sectionHeaderStyle.Sprint("PROXIES"))
				proxyTable := r.buildDeploymentTable(proxies)
				fmt.Fprint(r.out, renderTableWithWidths(proxyTable, globalColumnWidths, continuationPrefix))
				fmt.Fprintln(r.out)
				sectionsDisplayed++
			}

			// Display implementations section
			if len(implementations) > 0 {
				if sectionsDisplayed > 0 {
					fmt.Fprintln(r.out, continuationPrefix)
				}
				fmt.Fprintf(r.out, "%s%s\n", continuationPrefix, sectionHeaderStyle.Sprint("IMPLEMENTATIONS"))
				implTable := r.buildDeploymentTable(implementations)
				fmt.Fprint(r.out, renderTableWithWidths(implTable, globalColumnWidths, continuationPrefix))
				fmt.Fprintln(r.out)
				sectionsDisplayed++
			}

			// Display singletons section
			if len(singletons) > 0 {
				if sectionsDisplayed > 0 {
					fmt.Fprintln(r.out, continuationPrefix)
				}
				fmt.Fprintf(r.out, "%s%s\n", continuationPrefix, sectionHeaderStyle.Sprint("SINGLETONS"))
				singletonTable := r.buildDeploymentTable(singletons)
				fmt.Fprint(r.out, renderTableWithWidths(singletonTable, globalColumnWidths, continuationPrefix))
				fmt.Fprintln(r.out)
				sectionsDisplayed++
			}

			// Display libraries section
			if len(libraries) > 0 {
				if sectionsDisplayed > 0 {
					fmt.Fprintln(r.out, continuationPrefix)
				}
				fmt.Fprintf(r.out, "%s%s\n", continuationPrefix, sectionHeaderStyle.Sprint("LIBRARIES"))
				libraryTable := r.buildDeploymentTable(libraries)
				fmt.Fprint(r.out, renderTableWithWidths(libraryTable, globalColumnWidths, continuationPrefix))
				fmt.Fprintln(r.out)
				sectionsDisplayed++
			}

			if !isLastNetwork {
				fmt.Fprintln(r.out, continuationPrefix)
			} else {
				fmt.Fprintln(r.out)
			}
		}
	}

	fmt.Fprintf(r.out, "Total deployments: %d\n", len(deployments))
}

// buildDeploymentTable creates a TableData for a list of deployments
func (r *DeploymentsRenderer) buildDeploymentTable(deployments []*models.Deployment) TableData {
	tableData := make(TableData, 0)

	// Sort deployments by contract name (alphabetically)
	sort.Slice(deployments, func(i, j int) bool {
		// Get display names for comparison
		nameI := deployments[i].ContractDisplayName()
		nameJ := deployments[j].ContractDisplayName()
		
		// If names are the same, sort by timestamp (newest first)
		if nameI == nameJ {
			return deployments[i].CreatedAt.After(deployments[j].CreatedAt)
		}
		
		return nameI < nameJ
	})

	for _, deployment := range deployments {
		// Add main deployment row
		contractCell := r.getColoredDisplayName(deployment)
		if len(deployment.Tags) > 0 {
			contractCell += " " + tagsStyle.Sprintf("(%s)", deployment.Tags[0])
		}

		addressCell := addressStyle.Sprint(deployment.Address)

		verifiedCell := ""
		// For domain types, we check verification status differently
		if deployment.Transaction != nil {
			switch deployment.Transaction.Status {
			case models.TransactionStatusQueued:
				verifiedCell = pendingStyle.Sprint("⏳ queued")
			case models.TransactionStatusSimulated:
				verifiedCell = pendingStyle.Sprint("⏳ simulated")
			default:
				// Show verifier statuses
				verifiedCell = r.getVerifierStatuses(deployment)
			}
		} else {
			// No transaction info, show verifier statuses
			verifiedCell = r.getVerifierStatuses(deployment)
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

			// If we have the implementation deployment loaded, use its name
			if deployment.Implementation != nil {
				implDisplayName = deployment.Implementation.ContractDisplayName()
			}

			implRow := implPrefixStyle.Sprintf("└─ %s", implDisplayName)

			tableData = append(tableData, []string{
				implRow,
				"", // No address shown for implementation line
				"",
				"",
			})
		}
	}

	return tableData
}

// getVerifierStatuses returns the formatted verifier status string
func (r *DeploymentsRenderer) getVerifierStatuses(deployment *models.Deployment) string {
	verifierStatuses := []string{}

	// Check Etherscan status
	etherscanStatus := "?"
	if deployment.Verification.Verifiers != nil {
		if etherscan, exists := deployment.Verification.Verifiers["etherscan"]; exists {
			switch etherscan.Status {
			case "verified":
				etherscanStatus = verifiedStyle.Sprint("✓")
			case "failed":
				etherscanStatus = notVerifiedStyle.Sprint("✗")
			case "pending":
				etherscanStatus = pendingStyle.Sprint("⏳")
			default:
				etherscanStatus = "?"
			}
		}
	}
	verifierStatuses = append(verifierStatuses, fmt.Sprintf("🅔 %s", etherscanStatus))

	// Check Sourcify status
	sourcifyStatus := "?"
	if deployment.Verification.Verifiers != nil {
		if sourcify, exists := deployment.Verification.Verifiers["sourcify"]; exists {
			switch sourcify.Status {
			case "verified":
				sourcifyStatus = verifiedStyle.Sprint("✓")
			case "failed":
				sourcifyStatus = notVerifiedStyle.Sprint("✗")
			case "pending":
				sourcifyStatus = pendingStyle.Sprint("⏳")
			default:
				sourcifyStatus = "?"
			}
		}
	}
	verifierStatuses = append(verifierStatuses, fmt.Sprintf("🅢 %s", sourcifyStatus))

	return strings.Join(verifierStatuses, " ")
}

// getColoredDisplayName returns a colored display name for deployment
func (r *DeploymentsRenderer) getColoredDisplayName(dep *models.Deployment) string {
	name := dep.ContractDisplayName()

	switch dep.Type {
	case models.ProxyDeployment:
		return color.New(color.FgMagenta, color.Bold).Sprint(name)
	case models.LibraryDeployment:
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
