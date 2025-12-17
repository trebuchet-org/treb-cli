package cli

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// NewRegisterCmd creates the register command
func NewRegisterCmd() *cobra.Command {
	var (
		address      string
		contractPath string
		txHash       string
		label        string
		skipVerify   bool
	)

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register an existing contract deployment in the registry",
		Long: `Register a contract that was deployed outside of treb so it can be used with registry lookups.

This command allows you to add existing deployments to the treb registry. You can provide either:
- A transaction hash (and treb will trace the transaction to find all contract creations)
- Explicit parameters (address, contract path, transaction hash)

The command will:
1. Fetch and trace the transaction from the blockchain
2. Extract all contract creations from the transaction trace
3. Interactively prompt for labels and contract paths for each contract (if not provided)
4. Optionally verify the bytecode matches a contract in your workspace
5. Add the deployments to the registry

Examples:
  # Register using transaction hash (treb will trace and find all contracts)
  treb register --tx-hash 0x1234...

  # Register with explicit parameters (single contract)
  treb register --address 0xabcd... --contract src/Counter.sol:Counter --tx-hash 0x1234...

  # Register with a label
  treb register --address 0xabcd... --contract src/Counter.sol:Counter --tx-hash 0x1234... --label v1

  # Skip bytecode verification (useful for third-party contracts)
  treb register --address 0xabcd... --contract src/Counter.sol:Counter --tx-hash 0x1234... --skip-verify`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd)
			if err != nil {
				return err
			}

			if app.Config.Network == nil {
				return fmt.Errorf("no active network set in config, --network flag is required")
			}

			if app.Config.Namespace == "" {
				return fmt.Errorf("namespace must be set in config")
			}

			if txHash == "" {
				return fmt.Errorf("transaction hash is required (--tx-hash)")
			}

			// If address is provided, we're registering a single contract
			// Otherwise, we need to trace the transaction to find all contracts
			var contractsToRegister []usecase.ContractRegistration

			if address != "" {
				// Single contract with explicit address
				contractsToRegister = []usecase.ContractRegistration{
					{
						Address:      address,
						ContractPath: contractPath,
						Label:        label,
					},
				}
			} else {
				// Need to trace transaction to find contracts
				creations, err := app.RegisterDeployment.TraceTransaction(cmd.Context(), txHash)
				if err != nil {
					return fmt.Errorf("failed to trace transaction: %w", err)
				}

				if len(creations) == 0 {
					return fmt.Errorf("no contract creations found in transaction trace")
				}

				// Determine which contracts to register
				var selectedIndices []int
				if len(creations) > 1 && !app.Config.NonInteractive {
					// Show multi-select for multiple contracts
					title := fmt.Sprintf("Select contracts to register (%d found):", len(creations))
					selectedIndices, err = SelectContracts(creations, title)
					if err != nil {
						return fmt.Errorf("contract selection failed: %w", err)
					}
				} else {
					// Single contract or non-interactive mode - select all
					selectedIndices = make([]int, len(creations))
					for i := range creations {
						selectedIndices[i] = i
					}
				}

				// Prompt for details for each selected contract
				contractsToRegister = make([]usecase.ContractRegistration, len(selectedIndices))
				for idx, creationIdx := range selectedIndices {
					creation := creations[creationIdx]
					fmt.Printf(color.New(color.FgCyan, color.Bold).Sprintf("\nContract %d of %d: %s (%s)\n", idx+1, len(selectedIndices), creation.Address, creation.Kind))

					// Prompt for contract path
					currentContractPath := contractPath
					if currentContractPath == "" {
						prompt := promptui.Prompt{
							Label:    "Contract path (path/to/Contract.sol:ContractName)",
							Validate: validateContractPath,
							Default:  "",
						}
						result, err := prompt.Run()
						if err != nil {
							return fmt.Errorf("input cancelled: %w", err)
						}
						currentContractPath = result
					}

					// Prompt for label
					currentLabel := label
					if currentLabel == "" {
						prompt := promptui.Prompt{
							Label:   "Label (optional, e.g., v1, main)",
							Default: "",
						}
						result, err := prompt.Run()
						if err != nil {
							return fmt.Errorf("input cancelled: %w", err)
						}
						currentLabel = result
					}

					contractsToRegister[idx] = usecase.ContractRegistration{
						Address:      creation.Address,
						ContractPath: currentContractPath,
						Label:        currentLabel,
					}
				}
			}

			params := usecase.RegisterDeploymentParams{
				Address:      address,
				ContractPath: contractPath,
				TxHash:       txHash,
				Label:        label,
				SkipVerify:   skipVerify,
				Contracts:    contractsToRegister,
			}

			result, err := app.RegisterDeployment.Run(cmd.Context(), params)
			if err != nil {
				return fmt.Errorf("failed to register deployment: %w", err)
			}

			if app.Config.JSON {
				// JSON output for multiple deployments
				fmt.Printf("{\"deployments\":[")
				for i := range result.DeploymentIDs {
					if i > 0 {
						fmt.Printf(",")
					}
					fmt.Printf("{\"deploymentId\":\"%s\",\"address\":\"%s\",\"contractName\":\"%s\",\"label\":\"%s\"}",
						result.DeploymentIDs[i], result.Addresses[i], result.ContractNames[i], result.Labels[i])
				}
				fmt.Printf("]}\n")
				return nil
			}

			// Human-readable output
			fmt.Printf(color.New(color.FgGreen, color.Bold).Sprintf("✓ Successfully registered %d deployment(s)\n\n", len(result.DeploymentIDs)))
			for i := range result.DeploymentIDs {
				fmt.Printf("  Deployment %d:\n", i+1)
				fmt.Printf("    Deployment ID: %s\n", result.DeploymentIDs[i])
				fmt.Printf("    Address: %s\n", result.Addresses[i])
				fmt.Printf("    Contract: %s\n", result.ContractNames[i])
				if result.Labels[i] != "" {
					fmt.Printf("    Label: %s\n", result.Labels[i])
				}
				if i < len(result.DeploymentIDs)-1 {
					fmt.Println()
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&address, "address", "", "Contract address to register (optional if tracing transaction)")
	cmd.Flags().StringVar(&contractPath, "contract", "", "Contract path in format path/to/Contract.sol:ContractName")
	cmd.Flags().StringVar(&txHash, "tx-hash", "", "Transaction hash of the deployment (required)")
	cmd.Flags().StringVar(&label, "label", "", "Optional label for the deployment (e.g., v1, main)")
	cmd.Flags().BoolVar(&skipVerify, "skip-verify", false, "Skip bytecode verification (useful for third-party contracts)")

	return cmd
}


// validateContractPath validates a contract path input
// Contract path is optional - if empty, the label will be used as the contract name
func validateContractPath(input string) error {
	if input == "" {
		return nil // Optional - label will be used as contract name
	}
	// Check if it contains a colon (for path:ContractName format)
	if !strings.Contains(input, ":") {
		return fmt.Errorf("contract path should be in format 'path/to/Contract.sol:ContractName'")
	}
	return nil
}
