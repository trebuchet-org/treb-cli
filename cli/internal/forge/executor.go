package forge

import (
	"fmt"
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
	GetLibrary(libraryName string, chainID uint64) *types.DeploymentEntry
	RecordLibraryDeployment(libraryName string, result *types.DeploymentResult, chainID uint64) error
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