package forge

import (
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/trebuchet-org/treb-cli/cli/pkg/broadcast"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
	"github.com/ethereum/go-ethereum/common"
)

type RegistryManager interface {
	GetDeployment(contract, env string, chainID uint64) *types.DeploymentEntry
	GetDeploymentWithLabel(contract, env, label string, chainID uint64) *types.DeploymentEntry
	RecordDeployment(contract, env string, result *types.DeploymentResult, chainID uint64) error
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
	Label           string
	EnvVars         map[string]string
	Debug           bool
}

type DeploymentSetup struct {
	Contract    string
	Env         string
	Args        DeployArgs
	ScriptPath  string
	EnvVars     []string
	ProjectRoot string
}

func NewScriptExecutor(foundryProfile, projectRoot string, registry RegistryManager) *ScriptExecutor {
	return &ScriptExecutor{
		foundryProfile: foundryProfile,
		projectRoot:    projectRoot,
		registry:       registry,
		parser:         broadcast.NewParser(projectRoot),
	}
}

func (se *ScriptExecutor) setupDeployment(contract string, env string, args DeployArgs) (*DeploymentSetup, error) {
	scriptPath := fmt.Sprintf("script/deploy/Deploy%s.s.sol", contract)
	
	// Set environment variables
	envVars := os.Environ()
	
	// Add deployment-specific environment variables
	if args.EnvVars != nil {
		for key, value := range args.EnvVars {
			envVars = append(envVars, fmt.Sprintf("%s=%s", key, value))
		}
	}
	
	// Legacy environment variables for backwards compatibility
	envVars = append(envVars, fmt.Sprintf("DEPLOYMENT_ENV=%s", env))
	if args.Label != "" {
		envVars = append(envVars, fmt.Sprintf("DEPLOYMENT_LABEL=%s", args.Label))
	}
	if args.DeployerPK != "" {
		envVars = append(envVars, fmt.Sprintf("DEPLOYER_PRIVATE_KEY=%s", args.DeployerPK))
	}

	return &DeploymentSetup{
		Contract:    contract,
		Env:         env,
		Args:        args,
		ScriptPath:  scriptPath,
		EnvVars:     envVars,
		ProjectRoot: se.projectRoot,
	}, nil
}

func (se *ScriptExecutor) Deploy(contract string, env string, args DeployArgs) (*types.DeploymentResult, error) {
	// 1. Setup deployment configuration
	setup, err := se.setupDeployment(contract, env, args)
	if err != nil {
		return nil, fmt.Errorf("deployment setup failed: %w", err)
	}

	// 2. Predict address first using the new method
	predictResult, err := se.PredictAddress(contract, env, args)
	if err != nil {
		return nil, fmt.Errorf("address prediction failed: %w", err)
	}
	// Address prediction successful - don't print here as caller will handle

	// 3. Check if already deployed
	existing := se.registry.GetDeploymentWithLabel(contract, env, args.Label, args.ChainID)
	if existing != nil && existing.Address == predictResult.Address {
		// Check deployment status
		if existing.Deployment.Status == "pending_safe" {
			fmt.Printf("Contract deployment is pending Safe execution\n")
			fmt.Printf("Address: %s\n", existing.Address.Hex())
			fmt.Printf("Safe: %s\n", existing.Deployment.SafeAddress)
			if existing.Deployment.SafeTxHash != nil {
				fmt.Printf("Safe Tx Hash: %s\n", existing.Deployment.SafeTxHash.Hex())
			}
			fmt.Printf("\nPlease execute the pending Safe transaction before attempting to redeploy\n")
			
			// Return error to prevent proceeding
			return nil, fmt.Errorf("deployment pending Safe execution")
		}
		
		// Contract already deployed - don't print here as caller will handle
		
		// Convert hex strings back to byte arrays
		var salt [32]byte
		var initCodeHash [32]byte
		
		if saltBytes, err := hex.DecodeString(existing.Salt); err == nil && len(saltBytes) == 32 {
			copy(salt[:], saltBytes)
		}
		
		if hashBytes, err := hex.DecodeString(existing.InitCodeHash); err == nil && len(hashBytes) == 32 {
			copy(initCodeHash[:], hashBytes)
		}
		
		return &types.DeploymentResult{
			Address:         existing.Address,
			Salt:            salt,
			InitCodeHash:    initCodeHash,
			AlreadyDeployed: true,
		}, nil
	}

	// 4. Execute forge script for deployment
	cmdArgs := []string{"script", setup.ScriptPath, "-vvvv"} // High verbosity for better error messages
	
	if setup.Args.RpcUrl != "" {
		cmdArgs = append(cmdArgs, "--rpc-url", setup.Args.RpcUrl)
	}
	
	cmdArgs = append(cmdArgs, "--broadcast")
	
	if setup.Args.Verify {
		cmdArgs = append(cmdArgs, "--verify")
		if setup.Args.EtherscanApiKey != "" {
			cmdArgs = append(cmdArgs, fmt.Sprintf("--etherscan-api-key=%s", setup.Args.EtherscanApiKey))
		}
	}

	// Execute deployment script - don't print here as caller shows progress
	
	cmd := exec.Command("forge", cmdArgs...)
	cmd.Dir = setup.ProjectRoot
	cmd.Env = setup.EnvVars

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Script execution failed:\n")
		fmt.Printf("Command: forge %s\n", strings.Join(cmdArgs, " "))
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("Full output:\n%s\n", string(output))
		return nil, fmt.Errorf("forge script failed: %w", err)
	}
	
	// Show full output if debug is enabled
	if setup.Args.Debug {
		fmt.Printf("\n=== Full Foundry Script Output ===\n")
		fmt.Printf("%s\n", string(output))
		fmt.Printf("=== End of Output ===\n\n")
	} else {
		// Parse output for key information
		se.parseScriptOutput(string(output))
	}

	// 5. Try to parse structured output first
	result, err := se.parser.ParseDeploymentOutput(output)
	if err != nil {
		// Fallback to parsing broadcast file
		scriptName := fmt.Sprintf("Deploy%s.s.sol", contract)
		result, err = se.parser.ParseLatestBroadcast(scriptName, args.ChainID)
		if err != nil {
			return nil, fmt.Errorf("failed to parse deployment: %w", err)
		}
	}
	
	// 6. If Safe deployment, capture Safe address from environment variables
	if result.SafeTxHash != (common.Hash{}) {
		// First try args.EnvVars which contains the actual deployment environment
		if args.EnvVars != nil {
			if safeAddr, ok := args.EnvVars["DEPLOYER_SAFE_ADDRESS"]; ok && safeAddr != "" {
				result.SafeAddress = common.HexToAddress(safeAddr)
			}
		}
		
		// Fallback to OS environment (shouldn't normally be needed)
		if result.SafeAddress == (common.Address{}) {
			if safeAddr := os.Getenv("DEPLOYER_SAFE_ADDRESS"); safeAddr != "" {
				result.SafeAddress = common.HexToAddress(safeAddr)
			}
		}
		
		if result.SafeAddress == (common.Address{}) {
			// Don't print warning during deployment - it interrupts the spinner
			// The Safe address will be shown in the final summary if available
		}
	}

	// 7. Update registry
	err = se.registry.RecordDeployment(contract, env, result, args.ChainID)
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
		
		// Look for key deployment information
		if strings.Contains(line, "Transaction hash:") ||
		   strings.Contains(line, "Contract Address:") ||
		   strings.Contains(line, "Gas used:") ||
		   strings.Contains(line, "Block:") {
			fmt.Printf("%s\n", line)
		}
	}
}

func (se *ScriptExecutor) PredictAddress(contract string, env string, args DeployArgs) (*types.PredictResult, error) {
	// 1. Setup deployment configuration
	setup, err := se.setupDeployment(contract, env, args)
	if err != nil {
		return nil, fmt.Errorf("prediction setup failed: %w", err)
	}

	// 2. Execute the deployment script's predictAddress function
	cmdArgs := []string{"script", setup.ScriptPath, "--sig", "predictAddress()", "-vvvv"}
	
	// Add RPC URL if provided to ensure prediction uses same chain as deployment
	if setup.Args.RpcUrl != "" {
		cmdArgs = append(cmdArgs, "--rpc-url", setup.Args.RpcUrl)
	}
	
	// Execute predict address script - don't print here as caller shows progress
	
	cmd := exec.Command("forge", cmdArgs...)
	cmd.Dir = setup.ProjectRoot
	cmd.Env = setup.EnvVars

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Address prediction failed:\n")
		fmt.Printf("Command: forge %s\n", strings.Join(cmdArgs, " "))
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("Full output:\n%s\n", string(output))
		return nil, fmt.Errorf("forge script failed: %w", err)
	}

	// Show full output if debug is enabled
	if setup.Args.Debug {
		fmt.Printf("\n=== Full Foundry Script Output (Prediction) ===\n")
		fmt.Printf("%s\n", string(output))
		fmt.Printf("=== End of Output ===\n\n")
	}

	// 3. Parse output for predicted address using the new parser
	return se.parseScriptPredictionOutput(string(output))
}

// parseScriptPredictionOutput parses the output from deployment script's predictAddress function
func (se *ScriptExecutor) parseScriptPredictionOutput(output string) (*types.PredictResult, error) {
	lines := strings.Split(output, "\n")
	
	// Look for the line containing "Predicted Address:"
	predictedAddressRegex := regexp.MustCompile(`Predicted Address:\s*(0x[a-fA-F0-9]{40})`)
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if matches := predictedAddressRegex.FindStringSubmatch(line); len(matches) > 1 {
			address := common.HexToAddress(matches[1])
			
			// For now, return basic prediction result
			// TODO: Extract salt and init code hash if script provides them
			return &types.PredictResult{
				Address:      address,
				Salt:         [32]byte{}, // Will be filled from deployment
				InitCodeHash: [32]byte{}, // Will be filled from deployment
			}, nil
		}
	}
	
	return nil, fmt.Errorf("failed to parse prediction output: 'Predicted Address:' not found in output")
}

// DeployProxy deploys a proxy contract with special handling for proxy-specific data
func (se *ScriptExecutor) DeployProxy(proxyContract, implementationContract string, env string, args DeployArgs) (*types.DeploymentResult, error) {
	// 1. Setup deployment configuration
	setup, err := se.setupDeployment(proxyContract, env, args)
	if err != nil {
		return nil, fmt.Errorf("proxy deployment setup failed: %w", err)
	}

	// 2. Check for existing proxy deployment
	var existingDeployment *types.DeploymentEntry
	if args.Label != "" {
		existingDeployment = se.registry.GetDeploymentWithLabel(proxyContract, env, args.Label, args.ChainID)
	} else {
		existingDeployment = se.registry.GetDeployment(proxyContract, env, args.ChainID)
	}

	if existingDeployment != nil {
		// Proxy already deployed - don't print here as caller will handle
		
		// Convert hex strings back to byte arrays
		var salt [32]byte
		var initCodeHash [32]byte
		
		if saltBytes, err := hex.DecodeString(existingDeployment.Salt); err == nil && len(saltBytes) == 32 {
			copy(salt[:], saltBytes)
		}
		
		if hashBytes, err := hex.DecodeString(existingDeployment.InitCodeHash); err == nil && len(hashBytes) == 32 {
			copy(initCodeHash[:], hashBytes)
		}
		
		return &types.DeploymentResult{
			Address:         existingDeployment.Address,
			Salt:            salt,
			InitCodeHash:    initCodeHash,
			AlreadyDeployed: true,
		}, nil
	}

	// 3. Execute deployment script - don't print here as caller shows progress
	
	// Execute forge script for deployment
	cmdArgs := []string{"script", setup.ScriptPath, "-vvvv"} // High verbosity for better error messages
	
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

	// Execute deployment script - don't print here as caller shows progress
	
	cmd := exec.Command("forge", cmdArgs...)
	cmd.Dir = setup.ProjectRoot
	cmd.Env = setup.EnvVars

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Script execution failed:\n")
		fmt.Printf("Command: forge %s\n", strings.Join(cmdArgs, " "))
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("Full output:\n%s\n", string(output))
		return nil, fmt.Errorf("forge script failed: %w", err)
	}
	
	// Show full output if debug is enabled
	if args.Debug {
		fmt.Printf("\n=== Full Foundry Script Output ===\n")
		fmt.Printf("%s\n", string(output))
		fmt.Printf("=== End of Output ===\n\n")
	} else {
		// Parse output for key information
		se.parseScriptOutput(string(output))
	}

	// Try to parse structured output first
	result, err := se.parser.ParseDeploymentOutput(output)
	if err != nil {
		// Fallback to parsing broadcast file
		scriptName := fmt.Sprintf("Deploy%s.s.sol", proxyContract)
		result, err = se.parser.ParseLatestBroadcast(scriptName, args.ChainID)
		if err != nil {
			return nil, fmt.Errorf("failed to parse deployment: %w", err)
		}
	}

	// 4. Get implementation address from registry
	var implDeployment *types.DeploymentEntry
	if implLabel, ok := args.EnvVars["IMPLEMENTATION_LABEL"]; ok && implLabel != "" {
		implDeployment = se.registry.GetDeploymentWithLabel(implementationContract, env, implLabel, args.ChainID)
	} else {
		implDeployment = se.registry.GetDeployment(implementationContract, env, args.ChainID)
	}

	// 5. Update result with proxy-specific metadata
	if result.Metadata == nil {
		result.Metadata = &types.ContractMetadata{}
	}
	
	// Set the deployment script path as contract path for proxies
	result.Metadata.ContractPath = fmt.Sprintf("script/deploy/Deploy%s.s.sol", proxyContract)
	
	// Add proxy-specific metadata
	if result.Metadata.Extra == nil {
		result.Metadata.Extra = make(map[string]interface{})
	}
	result.Metadata.Extra["proxyType"] = result.Type // Should be "PROXY" from the script output
	if implDeployment != nil {
		result.Metadata.Extra["implementation"] = implDeployment.Address.Hex()
	}

	// 6. Record deployment with proxy type
	result.Type = "proxy" // Ensure type is set correctly
	result.TargetContract = implementationContract // Set the implementation reference
	if err := se.registry.RecordDeployment(proxyContract, env, result, args.ChainID); err != nil {
		return nil, fmt.Errorf("failed to record proxy deployment: %w", err)
	}

	// 7. Verify if requested (using proxy-specific verification)
	if args.Verify && result.TxHash != (common.Hash{}) {
		fmt.Printf("\nVerifying proxy on Etherscan...\n")
		// TODO: Implement proxy-specific verification
	}

	return result, nil
}

