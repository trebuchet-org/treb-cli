package deployment

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// ForgeJSONOutput represents the JSON output from forge script
type ForgeJSONOutput struct {
	Logs        []ForgeLogEntry `json:"logs"`
	Traces      []interface{}   `json:"traces"`
	Success     bool            `json:"success"`
	GasUsed     string          `json:"gas_used"`
	Deployments []interface{}   `json:"deployments"`
}

// ForgeLogEntry represents a log entry in forge JSON output
type ForgeLogEntry struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
}

// Event signatures (keccak256 hashes)
var (
	// DeploymentCompleted(address,string,string,bytes32,bytes32,uint8)
	DeploymentCompletedSig = "0x..." // TODO: Calculate actual signature
	// TransactionExecuted(address,address,string,uint8,bytes32)
	TransactionExecutedSig = "0x..." // TODO: Calculate actual signature
	// SafeTransactionQueued(address,address,bytes32,uint256)
	SafeTransactionQueuedSig = "0x..." // TODO: Calculate actual signature
	// ProxyDeployed(address,address,string,bytes)
	ProxyDeployedSig = "0x..." // TODO: Calculate actual signature
	// LibraryDeployed(address,string,bytes32)
	LibraryDeployedSig = "0x..." // TODO: Calculate actual signature
)

// parseJSONOutput attempts to parse JSON output and extract deployment information from events
func parseJSONOutput(output string) (DeploymentOutput, error) {
	// First check if this is JSON output
	if !strings.HasPrefix(strings.TrimSpace(output), "{") {
		// Not JSON, fall back to legacy parser
		return parseDeploymentResult(output)
	}

	var jsonOutput ForgeJSONOutput
	if err := json.Unmarshal([]byte(output), &jsonOutput); err != nil {
		// Failed to parse as JSON, fall back to legacy parser
		return parseDeploymentResult(output)
	}

	// Look for deployment events in logs
	result := DeploymentOutput{}
	
	for _, log := range jsonOutput.Logs {
		if len(log.Topics) == 0 {
			continue
		}
		
		switch log.Topics[0] {
		case DeploymentCompletedSig:
			// Parse DeploymentCompleted event
			if err := parseDeploymentCompletedEvent(log, &result); err != nil {
				return result, err
			}
		case ProxyDeployedSig:
			// Parse ProxyDeployed event
			if err := parseProxyDeployedEvent(log, &result); err != nil {
				return result, err
			}
		case LibraryDeployedSig:
			// Parse LibraryDeployed event
			if err := parseLibraryDeployedEvent(log, &result); err != nil {
				return result, err
			}
		case SafeTransactionQueuedSig:
			// Parse SafeTransactionQueued event
			if err := parseSafeTransactionQueuedEvent(log, &result); err != nil {
				return result, err
			}
		}
	}

	// If we didn't find deployment events, fall back to legacy parser
	if result.Address == "" && result.PredictedAddress == "" {
		return parseDeploymentResult(output)
	}

	return result, nil
}

func parseDeploymentCompletedEvent(log ForgeLogEntry, result *DeploymentOutput) error {
	// TODO: Implement actual event parsing based on ABI
	// For now, we'll fall back to legacy parsing
	return nil
}

func parseProxyDeployedEvent(log ForgeLogEntry, result *DeploymentOutput) error {
	// TODO: Implement actual event parsing based on ABI
	return nil
}

func parseLibraryDeployedEvent(log ForgeLogEntry, result *DeploymentOutput) error {
	// TODO: Implement actual event parsing based on ABI
	return nil
}

func parseSafeTransactionQueuedEvent(log ForgeLogEntry, result *DeploymentOutput) error {
	// TODO: Implement actual event parsing based on ABI
	return nil
}

// tryParseDeploymentOutput tries both JSON and legacy parsing
func tryParseDeploymentOutput(output string) (DeploymentOutput, error) {
	// For now, we'll keep using the legacy parser until we fully implement event parsing
	// This allows us to keep the existing functionality working
	return parseDeploymentResult(output)
	
	// TODO: Once event parsing is implemented, use this:
	// return parseJSONOutput(output)
}