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
		allFlag      bool
		forceFlag    bool
		contractPath string
		debugFlag    bool
		chainID      uint64
		namespace    string
		dumpCmd      bool
	)

	cmd := &cobra.Command{
		Use:   "verify [deployment-id|address]",
		Short: "Verify contracts on block explorers",
		Long: `Verify contracts on block explorers (Etherscan and Sourcify) and update registry status.

Examples:
  treb verify Counter                      # Verify specific contract
  treb verify Counter:v2                   # Verify specific deployment by label
  treb verify staging/Counter              # Verify by namespace/contract
  treb verify 11155111/Counter             # Verify by chain/contract
  treb verify staging/11155111/Counter     # Verify by namespace/chain/contract
  treb verify 0x1234...                    # Verify by address (requires --chain)
  treb verify --all                        # Verify all unverified contracts (skip local)
  treb verify --all --force                # Re-verify all contracts including verified
  treb verify Counter --force              # Re-verify even if already verified
  treb verify Counter --chain 11155111 --namespace staging  # Verify with filters
  treb verify CounterProxy --contract-path "./src/Counter.sol:Counter"  # Manual contract path`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			// Create options
			options := usecase.VerifyOptions{
				Force:        forceFlag,
				ContractPath: contractPath,
				Debug:        debugFlag,
				DumpCommand:  dumpCmd,
			}

			// Create filter
			filter := domain.DeploymentFilter{
				ChainID:   chainID,
				Namespace: namespace,
			}

			ctx := cmd.Context()

			if allFlag {
				// Verify all unverified contracts
				result, err := app.VerifyDeployment.VerifyAll(ctx, options)
				if err != nil {
					return fmt.Errorf("failed to verify contracts: %w", err)
				}

				if dumpCmd {
					for _, r := range result.Results {
						if len(r.DumpedCommands) > 0 {
							fmt.Fprintf(cmd.OutOrStdout(), "# %s\n", r.Deployment.ContractName)
							for _, c := range r.DumpedCommands {
								fmt.Fprintln(cmd.OutOrStdout(), c)
							}
						}
					}
					return nil
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

			if dumpCmd {
				for _, c := range result.DumpedCommands {
					fmt.Fprintln(cmd.OutOrStdout(), c)
				}
				return nil
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
	cmd.Flags().Uint64VarP(&chainID, "chain", "c", 0, "Filter by chain ID")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Filter by namespace")
	cmd.Flags().BoolVar(&dumpCmd, "dump-command", false, "Print the underlying forge verify-contract commands without executing")

	return cmd
}

// isNonInteractive checks if the environment is non-interactive
func isNonInteractive() bool {
	return os.Getenv("TREB_NON_INTERACTIVE") == "true" ||
		os.Getenv("CI") == "true" ||
		os.Getenv("NO_COLOR") != ""
}
