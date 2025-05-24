package cmd

import (
	"crypto/ecdsa"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/internal/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/config"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all deployments",
	Long: `Display deployments organized by environment and network.
Shows contract addresses, deployment status, and version tags.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := listDeployments(); err != nil {
			checkError(err)
		}
	},
}

func init() {
	// Add flags if needed in the future
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
			envHeader.Print("   ◎ environment ")
			envHeaderBold.Printf(" %-*s ", 33, strings.ToUpper(env))
			envHeader.Printf("  deployer ")
			envHeaderBold.Printf("%s ", deployerAddress)
			fmt.Println() // No extra newline after header
		}
		for _, network := range networks {
			deployments := groups[network][env]
			if len(deployments) == 0 {
				continue
			}
			// Chain header with color starting at text (no whitespace prefix)
			fmt.Print("└─")
			chainHeader.Print(" ⛓ chain       ")
			chainHeaderBold.Printf(" %-*s ", 87, strings.ToUpper(network))
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
