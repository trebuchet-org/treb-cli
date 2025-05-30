package deployment

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/abi"
	deploymentABI "github.com/trebuchet-org/treb-cli/cli/pkg/abi/deployment"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// ForgeJSONOutput represents the JSON output from forge script
type ForgeExecutionDetails struct {
	ReturnData string   `json:"returned"`
	Logs       []string `json:"logs"`
}

type ForgeTransactionsSummary struct {
	Status       string `json:"status"`
	Transactions string `json:"transactions"`
	Sensitive    string `json:"sensitive"`
}

// ParsedDeploymentResult extends the generated DeploymentResult with additional fields
type ParsedDeploymentResult struct {
	// Embed the generated DeploymentResult
	abi.DeploymentResult
	// Additional fields from forge output
	TxHash        common.Hash
	BlockNumber   uint64
	BroadcastFile string
	// Converted enums
	ParsedStatus         types.Status
	ParsedDeploymentType types.DeploymentType
	ParsedStrategy       types.DeployStrategy
}

// ExecutionStatus enum values from Solidity
const (
	ExecutionStatusPendingSafe = 0
	ExecutionStatusExecuted    = 1
)

// decodeDeploymentResult decodes the hex-encoded DeploymentResult struct using the generated bindings
func (ctx *DeploymentContext) decodeDeploymentResult(hexValue string) (ParsedDeploymentResult, error) {
	// Remove 0x prefix if present
	hexValue = strings.TrimPrefix(hexValue, "0x")
	data, err := hex.DecodeString(hexValue)
	if err != nil {
		return ParsedDeploymentResult{}, fmt.Errorf("failed to decode hex value: %w", err)
	}

	// Use the generated bindings to unpack the result
	contract := deploymentABI.NewDeployment()
	deploymentResult, err := contract.UnpackRun(data)
	if err != nil {
		return ParsedDeploymentResult{}, fmt.Errorf("failed to unpack return data: %w", err)
	}

	// Convert to our ParsedDeploymentResult
	result := ParsedDeploymentResult{
		DeploymentResult: deploymentResult,
		// Parse the string enums into our types
		ParsedStatus:   abi.StatusFromString(deploymentResult.Status),
		ParsedStrategy: abi.DeployStrategyFromString(deploymentResult.Strategy),
	}

	// Parse deployment type separately since it returns an error
	parsedType, err := types.ParseDeploymentType(deploymentResult.DeploymentType)
	if err != nil {
		result.ParsedDeploymentType = types.UnknownDeployment
	} else {
		result.ParsedDeploymentType = parsedType
	}

	return result, nil
}

func (ctx *DeploymentContext) buildDeploymentResult(
	executionDetails *ForgeExecutionDetails,
	transactionsSummary *ForgeTransactionsSummary,
) (ParsedDeploymentResult, error) {
	deploymentResult, err := ctx.decodeDeploymentResult(executionDetails.ReturnData)
	if err != nil {
		return ParsedDeploymentResult{}, fmt.Errorf("failed to decode deployment result: %w", err)
	}

	deploymentResult.BroadcastFile = transactionsSummary.Transactions

	return deploymentResult, nil
}

// parseDeploymentOutput tries to parse deployment output
func (ctx *DeploymentContext) parseDeploymentOutput(output string) (ParsedDeploymentResult, error) {
	var executionDetails *ForgeExecutionDetails
	var transactionsSummary *ForgeTransactionsSummary

	lines := strings.Split(output, "\n")

	if ctx.Params.Debug {
		fmt.Printf("Total lines: %d\n", len(lines))
		for i, line := range lines {
			if strings.TrimSpace(line) != "" {
				fmt.Printf("Line %d: %s\n", i, line[:min(100, len(line))])
			}
		}
	}

	// Find lines containing JSON data
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Try to parse as ForgeExecutionDetails (contains "returned" field)
		if strings.Contains(line, `"returned"`) && executionDetails == nil {
			if err := json.Unmarshal([]byte(line), &executionDetails); err == nil {
				if ctx.Params.Debug {
					fmt.Printf("Found execution details on line %d\n", i)
				}
				continue
			}
		}

		// Try to parse as ForgeTransactionsSummary (contains "transactions" field)
		if strings.Contains(line, `"transactions"`) && transactionsSummary == nil {
			if err := json.Unmarshal([]byte(line), &transactionsSummary); err == nil {
				if ctx.Params.Debug {
					fmt.Printf("Found transactions summary on line %d\n", i)
				}
				continue
			}
		}
	}

	if executionDetails == nil {
		return ParsedDeploymentResult{}, fmt.Errorf("no execution details found in output")
	}

	if transactionsSummary == nil {
		return ParsedDeploymentResult{}, fmt.Errorf("no transactions summary found in output")
	}

	if ctx.Params.Debug {
		fmt.Printf("executionDetails: %+v\n", executionDetails)
		fmt.Printf("transactionsSummary: %+v\n", transactionsSummary)
	}

	return ctx.buildDeploymentResult(executionDetails, transactionsSummary)
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
