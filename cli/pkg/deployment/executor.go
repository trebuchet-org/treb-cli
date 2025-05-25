package deployment

import (
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/internal/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/broadcast"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// Executor handles deployment execution
type Executor struct {
	projectRoot string
	parser      *Parser
}

// NewExecutor creates a new deployment executor
func NewExecutor(projectRoot string) *Executor {
	return &Executor{
		projectRoot: projectRoot,
		parser:      NewParser(),
	}
}

// Execute runs the deployment
func (e *Executor) Execute(ctx *Context) (*types.DeploymentResult, error) {
	registryPath := filepath.Join(e.projectRoot, "deployments.json")
	registryManager, err := registry.NewManager(registryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load registry: %w", err)
	}

	// Check for existing deployment
	if existing, isPending := e.checkExistingDeployment(ctx, registryManager); existing != nil {
		if isPending {
			return nil, fmt.Errorf("deployment is pending Safe execution at %s", existing.Address.Hex())
		}
		return existing, nil
	}

	// Set up environment variables
	envVars := os.Environ()
	for key, value := range ctx.EnvVars {
		envVars = append(envVars, fmt.Sprintf("%s=%s", key, value))
	}

	// Construct forge script command
	cmdArgs := []string{"script", ctx.ScriptPath}

	// Add network configuration
	if ctx.NetworkInfo.RpcUrl != "" {
		cmdArgs = append(cmdArgs, "--rpc-url", ctx.NetworkInfo.RpcUrl)
	}

	// Add broadcast flags for actual deployment
	if !ctx.Predict {
		cmdArgs = append(cmdArgs, "--broadcast")
	}

	// Add verbosity
	cmdArgs = append(cmdArgs, "-vvvv")

	// Execute forge script
	cmd := exec.Command("forge", cmdArgs...)
	cmd.Dir = e.projectRoot
	cmd.Env = envVars

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Debug {
			fmt.Printf("Full output:\n%s\n", string(output))
		}
		return nil, e.parser.HandleForgeError(err, output)
	}

	if ctx.Debug {
		fmt.Printf("\n=== Full Foundry Script Output ===\n")
		fmt.Printf("%s\n", string(output))
		fmt.Printf("=== End of Output ===\n\n")
	}

	// Parse deployment results
	results, err := e.parser.ParseDeploymentResult(string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse deployment output: %w", err)
	}

	// Build deployment result
	deployment := e.buildDeploymentResult(ctx, results)

	// Parse broadcast file if not predicting
	if !ctx.Predict && deployment.SafeTxHash == (common.Hash{}) {
		if broadcastData, err := e.loadBroadcastFile(ctx); err != nil {
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
	if !ctx.Predict {
		if err := e.updateRegistry(ctx, deployment, registryManager); err != nil {
			// Log warning but don't fail
			fmt.Printf("Warning: failed to update registry: %v\n", err)
		}
	}

	return deployment, nil
}

// checkExistingDeployment checks if deployment already exists
func (e *Executor) checkExistingDeployment(ctx *Context, registryManager *registry.Manager) (*types.DeploymentResult, bool) {
	// TODO: This needs to be updated to work with the new registry manager
	// The registry manager returns DeploymentEntry, not DeploymentResult
	// For now, return nil to allow deployment to proceed
	return nil, false
}

// buildDeploymentResult builds deployment result from parsed output
func (e *Executor) buildDeploymentResult(ctx *Context, results DeploymentOutput) *types.DeploymentResult {
	// Convert hex strings to byte arrays
	var salt [32]byte
	var initCodeHash [32]byte
	
	if saltBytes, err := hex.DecodeString(strings.TrimPrefix(results.Salt, "0x")); err == nil && len(saltBytes) == 32 {
		copy(salt[:], saltBytes)
	}
	
	if hashBytes, err := hex.DecodeString(strings.TrimPrefix(results.InitCodeHash, "0x")); err == nil && len(hashBytes) == 32 {
		copy(initCodeHash[:], hashBytes)
	}

	deployment := &types.DeploymentResult{
		DeploymentType:  string(ctx.Type),
		Env:             ctx.Env,
		Label:           ctx.Label,
		NetworkInfo:     ctx.NetworkInfo,
		ContractInfo:    ctx.ContractInfo,
		Status:          results.Status,
		Salt:            salt,
		InitCodeHash:    initCodeHash,
		SafeTxHash:      common.HexToHash(results.SafeTxHash),
		Address:         common.HexToAddress(results.Address),
		ConstructorArgs: results.ConstructorArgs,
		Metadata: &types.ContractMetadata{
			Compiler:     ctx.ContractInfo.Artifact.Metadata.Compiler.Version,
			ContractPath: ctx.ContractInfo.Path,
			ScriptPath:   ctx.ScriptPath,
			SourceHash:   ctx.ContractInfo.GetSourceHash(),
		},
	}

	return deployment
}

// loadBroadcastFile loads the broadcast file data
func (e *Executor) loadBroadcastFile(ctx *Context) (*broadcast.BroadcastFile, error) {
	// Use broadcast parser to get the raw broadcast file
	parser := broadcast.NewParser(e.projectRoot)
	return parser.ParseLatestBroadcast(filepath.Base(ctx.ScriptPath), ctx.NetworkInfo.ChainID)
}

// updateRegistry updates the deployment registry
func (e *Executor) updateRegistry(ctx *Context, deployment *types.DeploymentResult, registryManager *registry.Manager) error {
	registryPath := filepath.Join(e.projectRoot, "deployments.json")
	registryManager, err := registry.NewManager(registryPath)
	if err != nil {
		return nil
	}
	registryManager.RecordDeployment(ctx.ContractInfo, ctx.Env, deployment, ctx.NetworkInfo.ChainID)
	return nil
}
