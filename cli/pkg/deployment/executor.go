package deployment

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/abi"
	"github.com/trebuchet-org/treb-cli/cli/pkg/broadcast"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// Executor handles deployment execution
func (d *DeploymentContext) Execute() (*types.DeploymentResult, error) {
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
	results, err := tryParseDeploymentOutput(string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse deployment output: %w", err)
	}

	// Build deployment result
	deployment := d.buildDeploymentResult(results)

	// Parse broadcast file if not predicting
	if !d.Params.Predict && deployment.Status == types.StatusExecuted {
		if broadcastData, err := d.loadBroadcastFile(); err != nil {
			// Log warning but don't fail
			fmt.Printf("Warning: failed to load broadcast file: %v\n", err)
		} else {
			deployment.BroadcastData = broadcastData
			// Extract transaction details using the helper method
			if txHash, blockNum, err := broadcastData.GetTransactionHashForAddress(deployment.Address); err == nil {
				deployment.TxHash = txHash
				deployment.BlockNumber = blockNum
			}
		}
	}

	// Update registry if deployment was executed
	if !d.Params.Predict {
		if err := d.updateRegistry(deployment); err != nil {
			// Log warning but don't fail
			fmt.Printf("Warning: failed to update registry: %v\n", err)
		}
	}

	return deployment, nil
}

// buildDeploymentResult builds deployment result from parsed output
func (d *DeploymentContext) buildDeploymentResult(result DeploymentOutput) *types.DeploymentResult {
	// Determine the address based on deployment type
	var address common.Address
	if d.Params.DeploymentType == types.LibraryDeployment && result.LibraryAddress != "" {
		address = common.HexToAddress(result.LibraryAddress)
	} else if result.Address != "" {
		address = common.HexToAddress(result.Address)
	}

	// Convert hex strings to byte arrays
	deployment := &types.DeploymentResult{
		FQID:                 d.GetFQID(),
		ShortID:              d.GetShortID(),
		TargetDeploymentFQID: d.targetDeploymentFQID,
		DeploymentType:       d.Params.DeploymentType,
		Namespace:            d.Params.Namespace,
		Label:                d.Params.Label,
		NetworkInfo:          d.networkInfo,
		ContractInfo:         d.contractInfo,
		Status:               result.Status,
		Salt:                 result.Salt,
		InitCodeHash:         result.InitCodeHash,
		SafeTxHash:           common.HexToHash(result.SafeTxHash),
		Address:              address,
		ConstructorArgs:      result.ConstructorArgs,
		Metadata: &types.ContractMetadata{
			Compiler:     d.contractInfo.Artifact.Metadata.Compiler.Version,
			ContractPath: d.contractInfo.Path,
			ScriptPath:   d.ScriptPath,
			SourceHash:   d.contractInfo.GetSourceHash(),
		},
	}

	return deployment
}

// loadBroadcastFile loads the broadcast file data
func (d *DeploymentContext) loadBroadcastFile() (*broadcast.BroadcastFile, error) {
	// Use broadcast parser to get the raw broadcast file
	parser := broadcast.NewParser(d.projectRoot)
	return parser.ParseLatestBroadcast(filepath.Base(d.ScriptPath), d.networkInfo.ChainID())
}

// updateRegistry updates the deployment registry
func (d *DeploymentContext) updateRegistry(deployment *types.DeploymentResult) error {
	d.registryManager.RecordDeployment(d.contractInfo, d.Params.Namespace, deployment, d.networkInfo.ChainID())
	return nil
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

	// Add JSON output for event parsing when not in debug mode
	// In debug mode, we want human-readable output
	if !d.Params.Debug && !d.Params.Predict {
		flags = append(flags, "--json")
	}

	// Add library flags if any
	if len(d.resolvedLibraries) > 0 {
		libFlags := generateLibraryFlags(d.resolvedLibraries)
		flags = append(flags, libFlags...)
	}

	// Try to encode deployment config for new run(DeploymentConfig) method
	functionArgs, err := d.prepareScriptArguments()
	if err != nil {
		// Log warning but continue - might be using legacy run() method
		if d.Params.Debug {
			fmt.Printf("Warning: Could not prepare script arguments: %v\n", err)
		}
	}

	var output string
	if len(functionArgs) > 0 {
		// Use new method with encoded config
		output, err = d.forge.RunScriptWithArgs(d.ScriptPath, flags, d.envVars, functionArgs)
	} else {
		// Fall back to legacy method
		output, err = d.forge.RunScript(d.ScriptPath, flags, d.envVars)
	}

	if err != nil {
		if d.Params.Debug {
			fmt.Printf("Full output:\n%s\n", output)
		}
		return output, err
	}

	if d.Params.Debug {
		fmt.Printf("\n=== Full Foundry Script Output ===\n")
		fmt.Printf("%s\n", string(output))
		fmt.Printf("=== End of Output ===\n\n")
	}

	return output, nil
}

// prepareScriptArguments prepares the encoded arguments for the script
func (d *DeploymentContext) prepareScriptArguments() ([]string, error) {
	// Try to read the script ABI
	scriptName := strings.TrimSuffix(filepath.Base(d.ScriptPath), ".s.sol")
	abiPath := filepath.Join(d.projectRoot, "out", filepath.Base(d.ScriptPath), scriptName + ".json")
	
	// Read the ABI file
	abiData, err := os.ReadFile(abiPath)
	if err != nil {
		// ABI file doesn't exist - script might use legacy run() method
		return nil, nil
	}
	
	// Parse the ABI to check if run() method accepts DeploymentConfig
	var contractABI struct {
		ABI json.RawMessage `json:"abi"`
	}
	if err := json.Unmarshal(abiData, &contractABI); err != nil {
		return nil, fmt.Errorf("failed to parse ABI file: %w", err)
	}
	
	// Check if run method accepts DeploymentConfig
	var abiArray []map[string]interface{}
	if err := json.Unmarshal(contractABI.ABI, &abiArray); err != nil {
		return nil, fmt.Errorf("failed to parse ABI array: %w", err)
	}
	
	// Find the run method
	var runMethod map[string]interface{}
	for _, method := range abiArray {
		if method["type"] == "function" && method["name"] == "run" {
			inputs, ok := method["inputs"].([]interface{})
			if ok && len(inputs) > 0 {
				runMethod = method
				break
			}
		}
	}
	
	if runMethod == nil {
		// No run method with parameters found - use legacy method
		return nil, nil
	}
	
	// Create deployment config
	senderAddr := common.HexToAddress(d.envVars["SENDER_ADDRESS"])
	chainId, _ := new(big.Int).SetString(d.networkInfo.ChainID(), 10)
	
	config := abi.CreateDeploymentConfig(
		d.Params.ProjectName,
		d.Params.Namespace,
		d.Params.Label,
		chainId,
		d.networkInfo.Name,
		senderAddr,
		d.envVars["SENDER_TYPE"],
		common.Address{}, // Registry address - not implemented yet
		!d.Params.Predict, // broadcast
		d.Params.Verify,
	)
	
	// Encode the config
	encodedConfig, err := abi.EncodeDeploymentConfig(config, string(contractABI.ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to encode deployment config: %w\nThis might indicate a version mismatch between treb-cli and treb-sol")
	}
	
	if encodedConfig == "" {
		// Legacy run() method
		return nil, nil
	}
	
	// Return the encoded config as script argument
	return []string{"--sig", "run((string,string,string,uint256,string,address,string,address,bool,bool))", encodedConfig}, nil
}
