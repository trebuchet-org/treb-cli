package deployment

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// Predict runs address prediction for the deployment
func (d *DeploymentContext) Predict() (*types.PredictResult, error) {
	switch d.Params.DeploymentType {
	case TypeLibrary:
		return d.predictLibrary()
	default:
		return d.predictScript()
	}
}

// predictScript runs prediction using deployment script
func (d *DeploymentContext) predictScript() (*types.PredictResult, error) {
	// Execute script without broadcast
	output, err := d.runScript()
	if err != nil {
		return nil, err
	}
	// Parse prediction output
	result, err := d.parsePredictionOutput(output)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// predictLibrary runs prediction for library deployment
func (d *DeploymentContext) predictLibrary() (*types.PredictResult, error) {
	// Execute script without broadcast
	output, err := d.runScript()
	if err != nil {
		return nil, err
	}
	// Parse library address from output
	address, err := parseLibraryAddress(output)
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
func (d *DeploymentContext) GetExistingAddress() common.Address {
	if deployment := d.registryManager.GetDeployment(d.GetFQID()); deployment != nil {
		return deployment.Address
	}
	return common.Address{}
}
