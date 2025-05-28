package deployment

import (
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/abi"
	"github.com/trebuchet-org/treb-cli/cli/pkg/broadcast"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// Executor handles deployment execution
func (d *DeploymentContext) Execute() (*ParsedDeploymentResult, error) {
	// Check for existing deployment
	entry := d.registryManager.GetDeployment(d.GetFQID())
	if entry != nil {
		d.printExistingDeployment(entry)
		return nil, fmt.Errorf("entry already exists")
	}

	output, err := d.runScript()
	if err != nil {
		return nil, err
	}

	// Parse deployment results - supports both JSON events and legacy logs
	result, err := d.parseDeploymentOutput(string(output))
	if err != nil {
		if d.Params.Debug {
			fmt.Println("======== SCRIPT OUTPUT =========")
			fmt.Println(string(output))
			fmt.Println("================================")
		}
		return nil, fmt.Errorf("failed to parse deployment output: %w", err)
	}

	// Parse broadcast file if not predicting to get tx hash and block number
	if !d.Params.Predict && result.ParsedStatus == types.StatusExecuted && result.BroadcastFile != "" {
		parser := broadcast.NewParser(d.projectRoot)
		if broadcastData, err := parser.ParseBroadcastFile(result.BroadcastFile); err != nil {
			// Log warning but don't fail
			fmt.Printf("Warning: failed to load broadcast file: %v\n", err)
		} else {
			// Extract transaction details using the helper method
			if txHash, blockNum, err := broadcastData.GetTransactionHashForAddress(result.Deployed); err == nil {
				result.TxHash = txHash
				result.BlockNumber = blockNum
			}
		}
	}

	// Update registry if deployment was executed
	if !d.Params.Predict {
		entry, err := d.BuildDeploymentEntry(&result)
		if err != nil {
			return &result, fmt.Errorf("failed to build deployment entry: %w", err)
		}
		
		// Verify contract exists at the deployed address for executed deployments
		if result.ParsedStatus == types.StatusExecuted {
			if err := d.verifyDeployment(result.Deployed); err != nil {
				return &result, fmt.Errorf("deployment verification failed: %w", err)
			}
		}
		
		if err := d.registryManager.RecordDeployment(d.networkInfo.ChainID(), entry); err != nil {
			return &result, fmt.Errorf("could not record deployed contract: %w", err)
		}
	}

	return &result, nil
}

func (d *DeploymentContext) runScript() (string, error) {
	flags := []string{
		"--rpc-url", d.networkInfo.RpcUrl,
		"-vvvv",
		"--non-interactive", // Prevent TTY-related errors
	}

	if !d.Params.Predict {
		flags = append(flags, "--broadcast")
	}

	// Always use JSON output for parsing events
	if !d.Params.Predict {
		flags = append(flags, "--json")
	}

	// Add library flags if any
	if len(d.resolvedLibraries) > 0 {
		libFlags := generateLibraryFlags(d.resolvedLibraries)
		flags = append(flags, libFlags...)
	}

	// Try to prepare script arguments for the new method
	functionArgs, err := d.prepareScriptArguments()
	if err != nil {
		// Log warning but continue - might be using legacy run() method
		if d.Params.Debug {
			fmt.Printf("Warning: Could not prepare script arguments: %v\n", err)
		}
	}

	// Always use the new method with encoded config
	if len(functionArgs) == 0 {
		return "", fmt.Errorf("deployment script must accept DeploymentConfig parameter")
	}

	output, err := d.forge.RunScriptWithArgs(d.ScriptPath, flags, nil, functionArgs)

	if err != nil {
		// If script failed with JSON output, rerun without JSON and broadcast to get readable error
		if !d.Params.Predict && strings.Contains(output, "script failed") {
			// Remove --json and --broadcast flags for error diagnosis
			debugFlags := []string{
				"--rpc-url", d.networkInfo.RpcUrl,
				"-vvvv",
				"--non-interactive",
			}

			// Add library flags if any
			if len(d.resolvedLibraries) > 0 {
				libFlags := generateLibraryFlags(d.resolvedLibraries)
				debugFlags = append(debugFlags, libFlags...)
			}

			// Rerun without broadcast/json to get full error output
			debugOutput, _ := d.forge.RunScriptWithArgs(d.ScriptPath, debugFlags, nil, functionArgs)

			// Show the full error to the user
			fmt.Printf("\n=== Script Error Details ===\n")
			fmt.Printf("%s\n", debugOutput)
			fmt.Printf("=== End of Error ===\n\n")
		}

		return output, err
	}

	if d.Params.Debug {
		// Even in debug mode, we're using JSON, so only show full output if needed
		fmt.Printf("\n=== Script executed successfully ===\n")
	}

	return output, nil
}

// prepareScriptArguments prepares the encoded arguments for the script
func (d *DeploymentContext) prepareScriptArguments() ([]string, error) {
	// Try to read the script ABI
	var calldata []byte
	switch d.Params.DeploymentType {
	case types.ProxyDeployment:
		if d.proxyDeploymentConfig == nil {
			return nil, fmt.Errorf("proxy deployment config not initialized")
		}
		calldata = abi.EncodeProxyDeploymentRun(d.proxyDeploymentConfig)

	case types.LibraryDeployment:
		if d.libraryDeploymentConfig == nil {
			return nil, fmt.Errorf("library deployment config not initialized")
		}
		calldata = abi.EncodeLibraryDeploymentRun(d.libraryDeploymentConfig)

	default:
		if d.deploymentConfig == nil {
			return nil, fmt.Errorf("deployment config not initialized")
		}
		calldata = abi.EncodeDeploymentRun(d.deploymentConfig)
	}

	// Return the full calldata as script argument
	return []string{"--sig", "0x" + hex.EncodeToString(calldata)}, nil
}

// verifyDeployment checks that a contract exists at the deployed address
func (d *DeploymentContext) verifyDeployment(address common.Address) error {
	// Use cast to get the bytecode at the address
	cmd := exec.Command("cast", "code", address.Hex(), "--rpc-url", d.networkInfo.RpcUrl)
	cmd.Dir = d.projectRoot
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to check contract code: %w\nOutput: %s", err, output)
	}
	
	// Check if the output is empty or just "0x"
	code := strings.TrimSpace(string(output))
	if code == "" || code == "0x" {
		return fmt.Errorf("no contract found at address %s - deployment may have failed", address.Hex())
	}
	
	// Contract exists if we have any bytecode
	return nil
}
