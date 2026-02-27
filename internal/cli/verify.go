package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/internal/cli/render"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// NewVerifyCmd creates the verify command using the new architecture
func NewVerifyCmd() *cobra.Command {
	var (
		allFlag               bool
		forceFlag             bool
		contractPath          string
		debugFlag             bool
		namespace             string
		etherscan             bool
		blockscout            bool
		sourcify              bool
		blockscoutVerifierURL string
	)

	cmd := &cobra.Command{
		Use:   "verify [deployment-id|address]",
		Short: "Verify contracts on block explorers",
		Long: `Verify contracts on block explorers (Etherscan, Blockscout, and Sourcify) and update registry status.

Examples:
  treb verify Counter                      # Verify specific contract (all verifiers)
  treb verify Counter -e                   # Verify on Etherscan only
  treb verify Counter -eb                  # Verify on Etherscan and Blockscout
  treb verify Counter --etherscan --sourcify  # Verify on Etherscan and Sourcify
  treb verify Counter:v2                   # Verify specific deployment by label
  treb verify staging/Counter              # Verify by namespace/contract
  treb verify Counter --network sepolia    # Verify by contract on network
  treb verify staging/Counter              # Verify by namespace/contract
  treb verify 0x1234... --network sepolia  # Verify by address (requires --network)
  treb verify --all                        # Verify all unverified contracts (skip local)
  treb verify --all --force                # Re-verify all contracts including verified
  treb verify Counter --force              # Re-verify even if already verified
  treb verify Counter --network sepolia --namespace staging  # Verify with filters
  treb verify CounterProxy --contract-path "./src/Counter.sol:Counter"  # Manual contract path`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			// Block verify in fork mode
			if active, _ := isForkActiveForCurrentNetwork(cmd.Context(), app); active {
				return fmt.Errorf("cannot verify contracts on a fork")
			}

			// Determine which verifiers to use
			// If none specified, use all
			verifiers := []string{}
			if !etherscan && !blockscout && !sourcify {
				verifiers = []string{"etherscan", "blockscout", "sourcify"}
			} else {
				if etherscan {
					verifiers = append(verifiers, "etherscan")
				}
				if blockscout {
					verifiers = append(verifiers, "blockscout")
				}
				if sourcify {
					verifiers = append(verifiers, "sourcify")
				}
			}

			// Create options
			options := usecase.VerifyOptions{
				Force:                 forceFlag,
				ContractPath:          contractPath,
				Debug:                 debugFlag,
				Verifiers:             verifiers,
				BlockscoutVerifierURL: blockscoutVerifierURL,
			}

			// Create filter
			filter := domain.DeploymentFilter{
				Namespace: namespace,
			}
			
			// Get network info from app config if available
			if app.Config.Network != nil {
				filter.ChainID = app.Config.Network.ChainID
			}

			ctx := cmd.Context()

			if allFlag {
				// Verify all unverified contracts
				result, err := app.VerifyDeployment.VerifyAll(ctx, filter, options)
				if err != nil {
					return fmt.Errorf("failed to verify contracts: %w", err)
				}

				// Render the results
				renderer := render.NewVerifyRenderer(cmd.OutOrStdout(), !isNonInteractive())
				return renderer.RenderVerifyAllResult(result, options)
			}

			if len(args) == 0 {
				return fmt.Errorf("please provide a deployment identifier or use --all flag")
			}

			// Verify specific contract
			identifier := args[0]
			result, err := app.VerifyDeployment.VerifySpecific(ctx, identifier, filter, options)
			if err != nil {
				return err
			}

			// Render the result
			renderer := render.NewVerifyRenderer(cmd.OutOrStdout(), !isNonInteractive())
			return renderer.RenderVerifyResult(result, options)
		},
	}

	cmd.Flags().BoolVar(&allFlag, "all", false, "Verify all unverified contracts (pending/failed)")
	cmd.Flags().BoolVar(&forceFlag, "force", false, "Re-verify even if already verified")
	cmd.Flags().StringVar(&contractPath, "contract-path", "", "Manual contract path (e.g., ./src/Contract.sol:Contract)")
	cmd.Flags().BoolVar(&debugFlag, "debug", false, "Show debug information including forge verify commands")
	cmd.Flags().StringP("network", "n", "", "Network to run on (e.g., mainnet, sepolia, local)")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Filter by namespace")

	// Verifier selection flags
	cmd.Flags().BoolVarP(&etherscan, "etherscan", "e", false, "Verify on Etherscan")
	cmd.Flags().BoolVarP(&blockscout, "blockscout", "b", false, "Verify on Blockscout")
	cmd.Flags().BoolVarP(&sourcify, "sourcify", "s", false, "Verify on Sourcify")
	cmd.Flags().StringVar(&blockscoutVerifierURL, "blockscout-verifier-url", "", "Custom Blockscout API URL (e.g., https://explorer.example.com/api)")

	return cmd
}

// isNonInteractive checks if the environment is non-interactive
func isNonInteractive() bool {
	return os.Getenv("TREB_NON_INTERACTIVE") == "true" ||
		os.Getenv("CI") == "true" ||
		os.Getenv("NO_COLOR") != ""
}
