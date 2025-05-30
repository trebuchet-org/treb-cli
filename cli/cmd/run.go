package cmd

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/spf13/cobra"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/cli/pkg/script"
)

var runCmd = &cobra.Command{
	Use:   "run <script-file>",
	Short: "Run a Foundry script with treb infrastructure",
	Long: `Run a Foundry script with automatic sender configuration and event tracking.

This command executes Foundry scripts while:
- Automatically configuring senders based on your treb configuration
- Parsing deployment events from script execution
- Recording deployments in the registry
- Supporting multiple sender types (private key, Safe, hardware wallet)

Examples:
  # Run a deployment script
  treb run script/deploy/DeployCounter.s.sol

  # Run with dry-run to see what would happen
  treb run script/deploy/DeployCounter.s.sol --dry-run

  # Run with debug output
  treb run script/deploy/DeployCounter.s.sol --debug

  # Run with specific network and profile
  treb run script/deploy/DeployCounter.s.sol --network sepolia --profile production`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		scriptPath := args[0]

		// Check if script exists
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			checkError(fmt.Errorf("script file not found: %s", scriptPath))
		}

		// Get flags
		network, _ := cmd.Flags().GetString("network")
		profile, _ := cmd.Flags().GetString("profile")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		debug, _ := cmd.Flags().GetBool("debug")
		debugJSON, _ := cmd.Flags().GetBool("debug-json")
		verbose, _ := cmd.Flags().GetBool("verbose")

		// Default network
		if network == "" {
			network = os.Getenv("DEPLOYMENT_NETWORK")
			if network == "" {
				network = "local"
			}
		}

		// Create script executor
		executor := script.NewExecutor(".")
		
		// Initialize indexer for contract identification
		indexer, err := contracts.GetGlobalIndexer(".")
		if err != nil {
			fmt.Printf("Warning: Could not initialize contract indexer: %v\n", err)
			indexer = nil
		}

		// Run the script
		fmt.Printf("Running script: %s\n", filepath.Base(scriptPath))
		fmt.Printf("Network: %s\n", network)
		fmt.Printf("Profile: %s\n", profile)
		if dryRun {
			fmt.Println("Mode: Dry run (no broadcast)")
		}
		fmt.Println()

		opts := script.RunOptions{
			ScriptPath: scriptPath,
			Network:    network,
			Profile:    profile,
			DryRun:     dryRun,
			Debug:      debug || debugJSON,
			DebugJSON:  debugJSON,
		}

		result, err := executor.Run(opts)
		checkError(err)

		if !result.Success {
			fmt.Println("\nScript execution failed")
			os.Exit(1)
		}

		// In debug mode, the raw output is already saved
		if debug || debugJSON {
			fmt.Printf("\nDebug output saved to: debug-output.json\n")
			fmt.Printf("Raw output size: %d bytes\n", len(result.RawOutput))
		}

		// Report all parsed events
		if len(result.AllEvents) > 0 {
			fmt.Printf("\n🔍 Parsed %d event(s):\n", len(result.AllEvents))
			for i, event := range result.AllEvents {
				fmt.Printf("  %s %s\n", getEventIcon(event.Type()), event.String())
				
				// In verbose mode, print all raw data
				if verbose {
					fmt.Printf("    Event %d Details:\n", i+1)
					printEventDetails(event, indexer)
				}
			}
			
			// Report deployment events specifically
			if len(result.ParsedEvents) > 0 {
				fmt.Printf("\n📦 %d contract(s) deployed:\n", len(result.ParsedEvents))
				for _, event := range result.ParsedEvents {
					fmt.Printf("  - %s (salt: %s)\n", event.Location.Hex(), event.Salt.Hex()[:10]+"...")
				}

				// TODO: Update registry if not dry run
				if !dryRun {
					fmt.Println("\n⚠️  Registry update not yet implemented")
				}
				
				// Report bundle events
				reportBundles(result.AllEvents, indexer)
			}
		} else if !dryRun {
			fmt.Println("\n⚠️  No events detected")
		}

		// Report broadcast file if found
		if result.BroadcastPath != "" {
			fmt.Printf("\nBroadcast file: %s\n", result.BroadcastPath)
		}

		fmt.Println("\nScript execution completed successfully")
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Network flag
	runCmd.Flags().StringP("network", "n", "", "Network to run on (e.g., mainnet, sepolia, local)")

	// Profile flag
	runCmd.Flags().StringP("profile", "p", "default", "Configuration profile to use")

	// Dry run flag
	runCmd.Flags().Bool("dry-run", false, "Perform a dry run without broadcasting transactions")

	// Debug flags
	runCmd.Flags().Bool("debug", false, "Enable debug mode (shows forge output and saves to file)")
	runCmd.Flags().Bool("debug-json", false, "Enable JSON debug mode (shows raw JSON output)")
	runCmd.Flags().BoolP("verbose", "v", false, "Show detailed event data for verification")
}

// getEventIcon returns an icon for the event type
func getEventIcon(eventType script.EventType) string {
	switch eventType {
	case script.EventTypeDeployingContract:
		return "🔨"
	case script.EventTypeContractDeployed:
		return "✅"
	case script.EventTypeSafeTransaction:
		return "🔐"
	case script.EventTypeBundleSent:
		return "📤"
	default:
		return "📝"
	}
}

// printEventDetails prints detailed information about an event for verification
func printEventDetails(event script.ParsedEvent, indexer *contracts.Indexer) {
	fmt.Printf("      Type: %s\n", event.Type())
	
	switch e := event.(type) {
	case *script.ContractDeployedEvent:
		fmt.Printf("      Deployer: %s\n", e.Deployer.Hex())
		fmt.Printf("      Location: %s\n", e.Location.Hex())
		fmt.Printf("      BundleID: %s\n", e.BundleID.Hex())
		fmt.Printf("      Salt: %s\n", e.Salt.Hex())
		fmt.Printf("      BytecodeHash: %s\n", e.BytecodeHash.Hex())
		fmt.Printf("      CreateStrategy: %s\n", e.CreateStrategy)
		fmt.Printf("      ConstructorArgs: 0x%x (%d bytes)\n", e.ConstructorArgs, len(e.ConstructorArgs))
		
		// Try to identify the contract by bytecode hash
		if indexer != nil {
			if contractInfo := indexer.GetContractByBytecodeHash(e.BytecodeHash.Hex()); contractInfo != nil {
				fmt.Printf("      Contract: %s:%s\n", contractInfo.Path, contractInfo.Name)
				if contractInfo.IsLibrary {
					fmt.Printf("      Type: library\n")
				} else if strings.Contains(strings.ToLower(contractInfo.Name), "proxy") {
					fmt.Printf("      Type: proxy\n")
				} else {
					fmt.Printf("      Type: regular\n")
				}
			} else {
				fmt.Printf("      Contract: <no match>\n")
			}
		}
		
	case *script.DeployingContractEvent:
		fmt.Printf("      What: %s\n", e.What)
		fmt.Printf("      Label: %s\n", e.Label)
		fmt.Printf("      InitCodeHash: %s\n", e.InitCodeHash.Hex())
		
	case *script.SafeTransactionQueuedEvent:
		fmt.Printf("      Safe: %s\n", e.Safe.Hex())
		fmt.Printf("      Proposer: %s\n", e.Proposer.Hex())
		fmt.Printf("      BundleID: %s\n", e.BundleID.Hex())
		fmt.Printf("      SafeTxHash: %s\n", e.SafeTxHash.Hex())
		fmt.Printf("      Nonce: %d\n", e.Nonce)
		
	case *script.BundleSentEvent:
		fmt.Printf("      Sender: %s\n", e.Sender.Hex())
		fmt.Printf("      BundleID: %s\n", e.BundleID.Hex())
		fmt.Printf("      Status: %d\n", e.Status)
		fmt.Printf("      TransactionCount: %d\n", e.TransactionCount)
		
	default:
		fmt.Printf("      Unknown event type\n")
	}
	fmt.Println()
}

// reportBundles reports bundle events and groups related transactions
func reportBundles(events []script.ParsedEvent, indexer *contracts.Indexer) {
	// Group events by bundle ID
	bundleMap := make(map[string]*BundleInfo)
	
	for _, event := range events {
		switch e := event.(type) {
		case *script.BundleSentEvent:
			bundleID := e.BundleID.Hex()
			if bundle, exists := bundleMap[bundleID]; exists {
				bundle.BundleSent = e
			} else {
				bundleMap[bundleID] = &BundleInfo{
					BundleSent: e,
					Deployments: []*script.ContractDeployedEvent{},
				}
			}
		case *script.ContractDeployedEvent:
			bundleID := e.BundleID.Hex()
			if bundle, exists := bundleMap[bundleID]; exists {
				bundle.Deployments = append(bundle.Deployments, e)
			} else {
				bundleMap[bundleID] = &BundleInfo{
					Deployments: []*script.ContractDeployedEvent{e},
				}
			}
		}
	}
	
	// Report bundles
	if len(bundleMap) > 0 {
		fmt.Printf("\n📤 %d bundle(s) captured:\n", len(bundleMap))
		for bundleID, bundle := range bundleMap {
			if bundleID == "0x0000000000000000000000000000000000000000000000000000000000000000" {
				// Skip zero bundle ID (non-broadcast transactions)
				continue
			}
			
			fmt.Printf("\n  Bundle: %s\n", bundleID[:10]+"...")
			
			if bundle.BundleSent != nil {
				status := "QUEUED"
				if bundle.BundleSent.Status == 1 {
					status = "EXECUTED"
				}
				fmt.Printf("    Sender: %s\n", bundle.BundleSent.Sender.Hex())
				fmt.Printf("    Status: %s\n", status)
			}
			
			if len(bundle.Deployments) > 0 {
				fmt.Printf("    Transactions:\n")
				for i, deployment := range bundle.Deployments {
					// Decode the deployment transaction
					txDesc := decodeDeploymentTransaction(deployment, indexer)
					fmt.Printf("      %d. %s\n", i+1, txDesc)
				}
			}
		}
	}
}

// BundleInfo groups related bundle events
type BundleInfo struct {
	BundleSent  *script.BundleSentEvent
	Deployments []*script.ContractDeployedEvent
}

// decodeDeploymentTransaction creates a human-friendly description of a deployment transaction
func decodeDeploymentTransaction(deployment *script.ContractDeployedEvent, indexer *contracts.Indexer) string {
	var contractName string
	if indexer != nil {
		if contractInfo := indexer.GetContractByBytecodeHash(deployment.BytecodeHash.Hex()); contractInfo != nil {
			contractName = contractInfo.Name
		} else {
			contractName = "Unknown"
		}
	} else {
		contractName = "Unknown"
	}
	
	// Try to decode constructor args if present
	var constructorDesc string
	if len(deployment.ConstructorArgs) > 0 {
		// Try to decode constructor arguments
		decodedArgs := tryDecodeConstructorArgs(contractName, deployment.ConstructorArgs)
		if decodedArgs != "" {
			constructorDesc = fmt.Sprintf("(%s)", decodedArgs)
		} else {
			constructorDesc = fmt.Sprintf("(%d bytes args)", len(deployment.ConstructorArgs))
		}
	} else {
		constructorDesc = "()"
	}
	
	// Format as: create3(new Counter()) or create3(new SampleToken(224 bytes args))
	strategy := strings.ToLower(deployment.CreateStrategy)
	return fmt.Sprintf("%s(new %s%s) → %s", strategy, contractName, constructorDesc, deployment.Location.Hex()[:10]+"...")
}

// tryDecodeConstructorArgs attempts to decode constructor arguments for known patterns
func tryDecodeConstructorArgs(contractName string, args []byte) string {
	// Common token pattern: string name, string symbol, uint256 totalSupply
	if strings.Contains(strings.ToLower(contractName), "token") && len(args) == 224 {
		// Try to decode as (string, string, uint256)
		stringType, _ := abi.NewType("string", "", nil)
		uint256Type, _ := abi.NewType("uint256", "", nil)
		
		arguments := abi.Arguments{
			{Type: stringType, Name: "name"},
			{Type: stringType, Name: "symbol"},
			{Type: uint256Type, Name: "totalSupply"},
		}
		
		values, err := arguments.Unpack(args)
		if err == nil && len(values) == 3 {
			name, nameOk := values[0].(string)
			symbol, symbolOk := values[1].(string)
			totalSupply, supplyOk := values[2].(*big.Int)
			
			if nameOk && symbolOk && supplyOk {
				// Format the total supply nicely
				supplyStr := formatTokenAmount(totalSupply)
				return fmt.Sprintf(`"%s", "%s", %s`, name, symbol, supplyStr)
			}
		}
	}
	
	// Add more patterns here as needed
	// For example: proxy patterns, upgradeable patterns, etc.
	
	return ""
}

// formatTokenAmount formats a big.Int as a human-readable token amount
func formatTokenAmount(amount *big.Int) string {
	// Assume 18 decimals by default
	decimals := big.NewInt(1e18)
	whole := new(big.Int).Div(amount, decimals)
	
	// If it's a clean whole number, display it nicely
	remainder := new(big.Int).Mod(amount, decimals)
	if remainder.Cmp(big.NewInt(0)) == 0 {
		return fmt.Sprintf("%s * 10^18", whole.String())
	}
	
	// Otherwise show the raw value
	return amount.String()
}