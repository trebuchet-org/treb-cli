package deployment

import (
	"fmt"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/broadcast"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// Executor handles deployment execution
func (d *DeploymentContext) Execute() (*types.DeploymentResult, error) {
	// Check for existing deployment
	deployment := d.registryManager.GetDeployment(d.GetFQID())
	if deployment != nil {
		d.printExistingDeployment(deployment)
		return nil, fmt.Errorf("deployment already exists")
	}

	output, err := d.runScript()
	if err != nil {
		return nil, err
	}

	// Parse deployment results
	results, err := parseDeploymentResult(string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse deployment output: %w", err)
	}

	// Build deployment result
	d.Deployment = d.buildDeploymentResult(results)

	// Parse broadcast file if not predicting
	if !d.Params.Predict && d.Deployment.Status == types.StatusExecuted {
		if broadcastData, err := d.loadBroadcastFile(); err != nil {
			// Log warning but don't fail
			fmt.Printf("Warning: failed to load broadcast file: %v\n", err)
		} else {
			d.Deployment.BroadcastData = broadcastData
			// Extract transaction details using the helper method
			if txHash, blockNum, err := broadcastData.GetTransactionHashForAddress(d.Deployment.Address); err == nil {
				d.Deployment.TxHash = txHash
				d.Deployment.BlockNumber = blockNum
			}
		}
	}

	// Update registry if deployment was executed
	if !d.Params.Predict {
		if err := d.updateRegistry(); err != nil {
			// Log warning but don't fail
			fmt.Printf("Warning: failed to update registry: %v\n", err)
		}
	}

	return d.Deployment, nil
}

// buildDeploymentResult builds deployment result from parsed output
func (d *DeploymentContext) buildDeploymentResult(result DeploymentOutput) *types.DeploymentResult {
	// Convert hex strings to byte arrays
	deployment := &types.DeploymentResult{
		FQID:            d.GetFQID(),
		ShortID:         d.GetShortID(),
		DeploymentType:  string(d.Params.DeploymentType),
		Env:             d.Params.Env,
		Label:           d.Params.Label,
		NetworkInfo:     d.networkInfo,
		ContractInfo:    d.contractInfo,
		Status:          result.Status,
		Salt:            result.Salt,
		InitCodeHash:    result.InitCodeHash,
		SafeTxHash:      common.HexToHash(result.SafeTxHash),
		Address:         common.HexToAddress(result.Address),
		ConstructorArgs: result.ConstructorArgs,
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
func (d *DeploymentContext) updateRegistry() error {
	d.registryManager.RecordDeployment(d.contractInfo, d.Params.Env, d.Deployment, d.networkInfo.ChainID())
	return nil
}

func (d *DeploymentContext) runScript() (string, error) {
	flags := []string{
		"--rpc-url", d.networkInfo.RpcUrl,
		"-vvvv",
	}

	if !d.Params.Predict {
		flags = append(flags, "--broadcast")
	}

	output, err := d.forge.RunScript(d.ScriptPath, flags, d.envVars)
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
