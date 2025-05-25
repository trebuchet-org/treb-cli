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
		if err := e.enrichFromBroadcast(ctx, deployment); err != nil {
			// Log warning but don't fail
			fmt.Printf("Warning: failed to parse broadcast file: %v\n", err)
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
func (e *Executor) buildDeploymentResult(ctx *Context, results map[string]string) *types.DeploymentResult {
	deployment := &types.DeploymentResult{
		Type:           string(ctx.Type),
		DeploymentType: string(ctx.Type),
		Env:            ctx.Env,
		Label:          ctx.Label,
	}

	// Set addresses
	if addr := results["deployedAddress"]; addr != "" {
		deployment.Address = common.HexToAddress(addr)
	}

	// Set salt and init code hash
	if salt := results["salt"]; salt != "" {
		// Convert hex string to [32]byte
		saltBytes, _ := hex.DecodeString(strings.TrimPrefix(salt, "0x"))
		copy(deployment.Salt[:], saltBytes)
	}
	if hash := results["initCodeHash"]; hash != "" {
		// Convert hex string to [32]byte
		hashBytes, _ := hex.DecodeString(strings.TrimPrefix(hash, "0x"))
		copy(deployment.InitCodeHash[:], hashBytes)
	}

	// Set Safe transaction hash if present
	if safeTxHash := results["safeTxHash"]; safeTxHash != "" {
		deployment.SafeTxHash = common.HexToHash(safeTxHash)
		// TODO: SafeAddress would need to be set from config or parsed output
	}

	return deployment
}

// enrichFromBroadcast enriches deployment result with broadcast data
func (e *Executor) enrichFromBroadcast(ctx *Context, deployment *types.DeploymentResult) error {
	// Use broadcast parser
	parser := broadcast.NewParser(e.projectRoot)
	broadcastResult, err := parser.ParseLatestBroadcast(filepath.Base(ctx.ScriptPath), ctx.NetworkInfo.ChainID)
	if err != nil {
		return err
	}

	// Copy relevant data from broadcast result
	if broadcastResult != nil {
		deployment.TxHash = broadcastResult.TxHash
		deployment.BlockNumber = broadcastResult.BlockNumber
		// Copy other relevant fields if available
	}

	return nil
}

// updateRegistry updates the deployment registry
func (e *Executor) updateRegistry(ctx *Context, deployment *types.DeploymentResult, registryManager *registry.Manager) error {
	// TODO: This needs to be updated to work with the new registry manager
	// The registry manager expects DeploymentEntry, not DeploymentResult
	// For now, just return nil
	return nil
}
