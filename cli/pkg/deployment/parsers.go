package deployment

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

type DeploymentOutput struct {
	Address               string `json:"address"`
	PredictedAddress      string `json:"predicted_address"`
	Salt                  string `json:"salt"`
	InitCodeHash          string `json:"init_code_hash"`
	Status                string `json:"status"`
	DeploymentType        string `json:"deployment_type"`
	Strategy              string `json:"strategy"`
	BlockNumber           string `json:"block_number"`
	ConstructorArgs       string `json:"constructor_args"`
	SafeTxHash            string `json:"safe_tx_hash"`
	ImplementationAddress string `json:"implementation_address"`
	LibraryAddress        string `json:"library_address"`
	ProxyInitializer      string `json:"proxy_initializer"`
}

// parseDeploymentResult parses deployment results from script output
func parseDeploymentResult(output string) (DeploymentOutput, error) {
	result := DeploymentOutput{}

	// Parse structured output between === DEPLOYMENT_RESULT === and === END_DEPLOYMENT ===
	startPattern := "  === DEPLOYMENT_RESULT ===\n"
	endPattern := "  === END_DEPLOYMENT ===\n"

	startIdx := strings.Index(output, startPattern)
	endIdx := strings.Index(output, endPattern)

	// Extract the section between markers
	deploymentSection := output[startIdx+len(startPattern) : endIdx]

	// Parse key:value pairs
	lines := strings.Split(deploymentSection, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split on first colon
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Map to expected field names
			switch key {
			case "ADDRESS":
				result.Address = value
			case "PREDICTED":
				result.PredictedAddress = value
			case "SALT":
				result.Salt = value
			case "INIT_CODE_HASH":
				result.InitCodeHash = value
			case "STATUS":
				result.Status = value
			case "DEPLOYMENT_TYPE":
				result.DeploymentType = value
			case "STRATEGY":
				result.Strategy = value
			case "BLOCK_NUMBER":
				result.BlockNumber = value
			case "CONSTRUCTOR_ARGS":
				result.ConstructorArgs = value
			case "SAFE_TX_HASH":
				result.SafeTxHash = value
			case "IMPLEMENTATION_ADDRESS":
				result.ImplementationAddress = value
			case "LIBRARY_ADDRESS":
				result.LibraryAddress = value
			case "PROXY_INITIALIZER":
				result.ProxyInitializer = value
			}
		}
	}

	return result, nil
}

// ParsePredictionOutput parses prediction output from script
func (d *DeploymentContext) parsePredictionOutput(output string) (*types.PredictResult, error) {
	// For predictions, use the same structured parser since they use the same format
	parsed, err := parseDeploymentResult(output)
	if err != nil {
		return nil, err
	}

	result := &types.PredictResult{}

	// Get predicted address - use ADDRESS or PREDICTED field
	if addr := parsed.Address; addr != "" {
		result.Address = common.HexToAddress(addr)
	} else if addr := parsed.PredictedAddress; addr != "" {
		result.Address = common.HexToAddress(addr)
	}

	// Get salt
	result.Salt = parsed.Salt
	return result, nil
}

// parseLibraryAddress parses library address from deployment output
func parseLibraryAddress(output string) (common.Address, error) {
	// Use structured parser to get library address
	parsed, err := parseDeploymentResult(output)
	if err != nil {
		return common.Address{}, err
	}

	// Check for library address
	if addr := parsed.LibraryAddress; addr != "" {
		return common.HexToAddress(addr), nil
	}

	return common.Address{}, fmt.Errorf("library address not found in output")
}
