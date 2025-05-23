package forge

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/bogdan/fdeploy/cli/pkg/types"
	"github.com/ethereum/go-ethereum/common"
)

type RegistryManager interface {
	GetDeployment(contract, env string) *types.DeploymentEntry
	RecordDeployment(contract, env string, result *types.DeploymentResult) error
}

type ScriptExecutor struct {
	foundryProfile string
	projectRoot    string
	registry       RegistryManager
}

type DeployArgs struct {
	RpcUrl           string
	EtherscanApiKey  string
	DeployerPK       string
	ChainID          uint64
	Verify           bool
	Networks         []string
}

func NewScriptExecutor(foundryProfile, projectRoot string, registry RegistryManager) *ScriptExecutor {
	return &ScriptExecutor{
		foundryProfile: foundryProfile,
		projectRoot:    projectRoot,
		registry:       registry,
	}
}

func (se *ScriptExecutor) Deploy(contract string, env string, args DeployArgs) (*types.DeploymentResult, error) {
	// 1. Predict address first
	predictResult, err := se.PredictAddress(contract, env, args)
	if err != nil {
		return nil, fmt.Errorf("address prediction failed: %w", err)
	}

	// 2. Check if already deployed
	existing := se.registry.GetDeployment(contract, env)
	if existing != nil && existing.Address == predictResult.Address {
		return &types.DeploymentResult{
			Address:      existing.Address,
			Salt:         existing.Salt,
			InitCodeHash: existing.InitCodeHash,
		}, nil
	}

	// 3. Execute forge script
	scriptPath := fmt.Sprintf("script/Deploy%s.s.sol", contract)
	
	cmdArgs := []string{"script", scriptPath}
	
	if args.RpcUrl != "" {
		cmdArgs = append(cmdArgs, "--rpc-url", args.RpcUrl)
	}
	
	cmdArgs = append(cmdArgs, "--broadcast")
	
	if args.Verify {
		cmdArgs = append(cmdArgs, "--verify")
		if args.EtherscanApiKey != "" {
			cmdArgs = append(cmdArgs, fmt.Sprintf("--etherscan-api-key=%s", args.EtherscanApiKey))
		}
	}

	cmd := exec.Command("forge", cmdArgs...)
	cmd.Dir = se.projectRoot

	// Set environment variables
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("DEPLOYMENT_ENV=%s", env),
	)
	
	if args.DeployerPK != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("DEPLOYER_PRIVATE_KEY=%s", args.DeployerPK))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("forge script failed: %w\nOutput: %s", err, output)
	}

	// 4. Parse broadcast file
	result, err := se.parseBroadcastFile(contract, args.ChainID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse broadcast: %w", err)
	}

	// 5. Update registry
	err = se.registry.RecordDeployment(contract, env, result)
	if err != nil {
		return nil, fmt.Errorf("failed to record deployment: %w", err)
	}

	return result, nil
}

func (se *ScriptExecutor) PredictAddress(contract string, env string, args DeployArgs) (*types.PredictResult, error) {
	scriptPath := "script/PredictAddress.s.sol"
	
	cmd := exec.Command("forge", "script", scriptPath,
		"--sig", fmt.Sprintf("predict(string,string)"),
		contract, env,
	)
	cmd.Dir = se.projectRoot

	cmd.Env = append(os.Environ(),
		fmt.Sprintf("DEPLOYMENT_ENV=%s", env),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("forge script failed: %w\nOutput: %s", err, output)
	}

	// Parse output for predicted address
	return se.parseAddressPrediction(output)
}

func (se *ScriptExecutor) parseBroadcastFile(contract string, chainID uint64) (*types.DeploymentResult, error) {
	// Find the latest broadcast file
	broadcastDir := filepath.Join(se.projectRoot, "broadcast", fmt.Sprintf("Deploy%s.s.sol", contract), fmt.Sprintf("%d", chainID))
	
	// Look for run-latest.json
	broadcastFile := filepath.Join(broadcastDir, "run-latest.json")
	
	if _, err := os.Stat(broadcastFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("broadcast file not found: %s", broadcastFile)
	}

	// Parse the broadcast file
	data, err := os.ReadFile(broadcastFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read broadcast file: %w", err)
	}

	var broadcast struct {
		Transactions []struct {
			Hash             string `json:"hash"`
			TransactionType  string `json:"transactionType"`
			ContractAddress  string `json:"contractAddress"`
		} `json:"transactions"`
		Receipts []struct {
			BlockNumber string `json:"blockNumber"`
		} `json:"receipts"`
	}

	if err := json.Unmarshal(data, &broadcast); err != nil {
		return nil, fmt.Errorf("failed to parse broadcast file: %w", err)
	}

	// Find CREATE transaction
	for i, tx := range broadcast.Transactions {
		if tx.TransactionType == "CREATE" || tx.TransactionType == "CREATE2" {
			result := &types.DeploymentResult{
				Address:       common.HexToAddress(tx.ContractAddress),
				TxHash:        common.HexToHash(tx.Hash),
				BroadcastFile: broadcastFile,
			}
			
			if i < len(broadcast.Receipts) {
				// Parse block number (it's a hex string)
				// TODO: Proper hex parsing
				result.BlockNumber = 0 // Placeholder
			}
			
			return result, nil
		}
	}

	return nil, fmt.Errorf("no CREATE transaction found in broadcast file")
}

func (se *ScriptExecutor) parseAddressPrediction(output []byte) (*types.PredictResult, error) {
	// TODO: Parse forge script output for predicted address
	// This would need to parse console.log output from the script
	
	return &types.PredictResult{
		Address:      common.Address{},
		Salt:         [32]byte{},
		InitCodeHash: [32]byte{},
	}, fmt.Errorf("address prediction parsing not implemented")
}