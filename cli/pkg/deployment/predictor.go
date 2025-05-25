package deployment

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// Predictor handles deployment address prediction
type Predictor struct {
	projectRoot string
	parser      *Parser
}

// NewPredictor creates a new deployment predictor
func NewPredictor(projectRoot string) *Predictor {
	return &Predictor{
		projectRoot: projectRoot,
		parser:      NewParser(),
	}
}

// Predict runs address prediction for the deployment
func (p *Predictor) Predict(ctx *Context) (*types.PredictResult, error) {
	switch ctx.Type {
	case TypeLibrary:
		return p.predictLibrary(ctx)
	default:
		return p.predictScript(ctx)
	}
}

// predictScript runs prediction using deployment script
func (p *Predictor) predictScript(ctx *Context) (*types.PredictResult, error) {
	// Set up environment variables
	envVars := os.Environ()
	for key, value := range ctx.EnvVars {
		envVars = append(envVars, fmt.Sprintf("%s=%s", key, value))
	}

	// Execute script without broadcast
	cmdArgs := []string{"script", ctx.ScriptPath, "-vvvv"}
	if ctx.NetworkInfo.RpcUrl != "" {
		cmdArgs = append(cmdArgs, "--rpc-url", ctx.NetworkInfo.RpcUrl)
	}

	cmd := exec.Command("forge", cmdArgs...)
	cmd.Dir = p.projectRoot
	cmd.Env = envVars

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Debug {
			fmt.Printf("Full output:\n%s\n", string(output))
		}
		return nil, p.parser.HandleForgeError(err, output)
	}

	if ctx.Debug {
		fmt.Printf("\n=== Full Foundry Script Output (Prediction) ===\n")
		fmt.Printf("%s\n", string(output))
		fmt.Printf("=== End of Output ===\n\n")
	}

	// Parse prediction output
	return p.parser.ParsePredictionOutput(string(output))
}

// predictLibrary runs prediction for library deployment
func (p *Predictor) predictLibrary(ctx *Context) (*types.PredictResult, error) {
	// Set up environment variables
	envVars := os.Environ()
	for key, value := range ctx.EnvVars {
		envVars = append(envVars, fmt.Sprintf("%s=%s", key, value))
	}

	// Execute script without broadcast
	cmdArgs := []string{"script", ctx.ScriptPath, "-vvvv"}
	if ctx.NetworkInfo.RpcUrl != "" {
		cmdArgs = append(cmdArgs, "--rpc-url", ctx.NetworkInfo.RpcUrl)
	}

	cmd := exec.Command("forge", cmdArgs...)
	cmd.Dir = p.projectRoot
	cmd.Env = envVars

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Debug {
			fmt.Printf("Full output:\n%s\n", string(output))
		}
		return nil, p.parser.HandleForgeError(err, output)
	}

	if ctx.Debug {
		fmt.Printf("\n=== Full Foundry Script Output (Prediction) ===\n")
		fmt.Printf("%s\n", string(output))
		fmt.Printf("=== End of Output ===\n\n")
	}

	// Parse library address from output
	address, err := p.parser.ParseLibraryAddress(string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse library address: %w", err)
	}

	// Build prediction result
	result := &types.PredictResult{
		Address: address,
		// Libraries don't use salt
	}

	return result, nil
}

// GetExistingAddress checks if deployment already exists and returns its address
func (p *Predictor) GetExistingAddress(ctx *Context, registryManager interface{}) common.Address {
	// Type assertion to avoid circular import
	type RegistryGetter interface {
		GetDeployment(identifier, network string) *types.DeploymentResult
		GetLibraryDeployment(name, network string) *types.DeploymentResult
	}
	
	registry, ok := registryManager.(RegistryGetter)
	if !ok {
		return common.Address{}
	}

	if ctx.Type == TypeLibrary {
		if deployment := registry.GetLibraryDeployment(ctx.ContractName, ctx.NetworkInfo.Name); deployment != nil {
			return deployment.Address
		}
	} else {
		identifier := ctx.GetFullIdentifier()
		if deployment := registry.GetDeployment(identifier, ctx.NetworkInfo.Name); deployment != nil {
			return deployment.Address
		}
	}
	
	return common.Address{}
}