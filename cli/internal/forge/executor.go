package forge

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/bogdan/fdeploy/cli/pkg/broadcast"
	"github.com/bogdan/fdeploy/cli/pkg/types"
)

type RegistryManager interface {
	GetDeployment(contract, env string) *types.DeploymentEntry
	RecordDeployment(contract, env string, result *types.DeploymentResult) error
}

type ScriptExecutor struct {
	foundryProfile string
	projectRoot    string
	registry       RegistryManager
	parser         *broadcast.Parser
}

type DeployArgs struct {
	RpcUrl          string
	EtherscanApiKey string
	DeployerPK      string
	ChainID         uint64
	Verify          bool
}

func NewScriptExecutor(foundryProfile, projectRoot string, registry RegistryManager) *ScriptExecutor {
	return &ScriptExecutor{
		foundryProfile: foundryProfile,
		projectRoot:    projectRoot,
		registry:       registry,
		parser:         broadcast.NewParser(projectRoot),
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
	
	cmdArgs := []string{"script", scriptPath, "-vvvv"} // High verbosity for better error messages
	
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

	fmt.Printf("🚀 Executing: forge %s\n", strings.Join(cmdArgs, " "))
	
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
		fmt.Printf("❌ Script execution failed:\n")
		fmt.Printf("Command: forge %s\n", strings.Join(cmdArgs, " "))
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("Full output:\n%s\n", string(output))
		return nil, fmt.Errorf("forge script failed: %w", err)
	}
	
	// Parse output for key information
	se.parseScriptOutput(string(output))

	// 4. Parse broadcast file
	scriptName := fmt.Sprintf("Deploy%s.s.sol", contract)
	result, err := se.parser.ParseLatestBroadcast(scriptName, args.ChainID)
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

// parseScriptOutput extracts key information from forge script output
func (se *ScriptExecutor) parseScriptOutput(output string) {
	lines := strings.Split(output, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Look for transaction hash
		if strings.Contains(line, "Transaction hash:") {
			fmt.Printf("🔍 %s\n", line)
		}
		
		// Look for contract address
		if strings.Contains(line, "Contract Address:") {
			fmt.Printf("📍 %s\n", line)
		}
		
		// Look for gas used
		if strings.Contains(line, "Gas used:") {
			fmt.Printf("⛽ %s\n", line)
		}
		
		// Look for block number
		if strings.Contains(line, "Block:") {
			fmt.Printf("📊 %s\n", line)
		}
	}
}

func (se *ScriptExecutor) PredictAddress(contract string, env string, args DeployArgs) (*types.PredictResult, error) {
	// Use the library's PredictAddress script 
	scriptPath := "lib/forge-deploy/script/PredictAddress.s.sol:PredictAddress"
	
	cmdArgs := []string{"script", scriptPath, "--sig", "predict(string,string)", contract, env, "-vvvv"}
	
	// Add RPC URL if provided to ensure prediction uses same chain as deployment
	if args.RpcUrl != "" {
		cmdArgs = append(cmdArgs, "--rpc-url", args.RpcUrl)
	}
	
	fmt.Printf("🔮 Predicting address: forge %s\n", strings.Join(cmdArgs, " "))
	
	cmd := exec.Command("forge", cmdArgs...)
	cmd.Dir = se.projectRoot

	cmd.Env = append(os.Environ(),
		fmt.Sprintf("DEPLOYMENT_ENV=%s", env),
		fmt.Sprintf("CONTRACT_VERSION=%s", "v1.0.0"), // TODO: Get from config
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("❌ Address prediction failed:\n")
		fmt.Printf("Command: forge %s\n", strings.Join(cmdArgs, " "))
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("Full output:\n%s\n", string(output))
		return nil, fmt.Errorf("forge script failed: %w", err)
	}

	// Parse output for predicted address
	return se.parser.ParsePredictionOutput(output)
}

