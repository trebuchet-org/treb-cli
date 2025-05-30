package script

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// ForgeScriptOutput represents the main output structure from forge script --json
type ForgeScriptOutput struct {
	Logs       []string  `json:"logs"`
	Success    bool      `json:"success"`
	RawLogs    []RawLog  `json:"raw_logs"`
	GasUsed    uint64    `json:"gas_used"`
}

// RawLog represents a raw log entry from the script output
type RawLog struct {
	Address common.Address   `json:"address"`
	Topics  []common.Hash    `json:"topics"`
	Data    string           `json:"data"`
}

// GasEstimate represents gas estimation output
type GasEstimate struct {
	Chain                   uint64 `json:"chain"`
	EstimatedGasPrice       string `json:"estimated_gas_price"`
	EstimatedTotalGasUsed   uint64 `json:"estimated_total_gas_used"`
	EstimatedAmountRequired string `json:"estimated_amount_required"`
}

// StatusOutput represents the status output from forge
type StatusOutput struct {
	Status       string `json:"status"`
	Transactions string `json:"transactions"`
	Sensitive    string `json:"sensitive"`
}

// ParseForgeOutput parses the JSON output from forge script
func ParseForgeOutput(output []byte) (*ForgeScriptOutput, error) {
	// First try to parse the entire output as a single JSON object
	var mainOutput ForgeScriptOutput
	if err := json.Unmarshal(output, &mainOutput); err == nil {
		// Check if this looks like the main output (has raw_logs)
		if mainOutput.RawLogs != nil {
			return &mainOutput, nil
		}
	}
	
	// If that fails, fall back to line-by-line parsing for multi-line output
	scanner := bufio.NewScanner(bytes.NewReader(output))
	
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		
		// Skip non-JSON lines (like error logs, ANSI codes, etc)
		if !bytes.HasPrefix(line, []byte("{")) {
			continue
		}
		
		// Try to parse as main output
		var lineOutput ForgeScriptOutput
		if err := json.Unmarshal(line, &lineOutput); err == nil {
			// Check if this looks like the main output (has raw_logs)
			if lineOutput.RawLogs != nil {
				return &lineOutput, nil
			}
		}
	}
	
	return nil, fmt.Errorf("no valid forge script output found")
}

// ParseAllEvents extracts all types of events from forge output
func ParseAllEvents(output *ForgeScriptOutput) ([]ParsedEvent, error) {
	var events []ParsedEvent
	
	for _, rawLog := range output.RawLogs {
		if len(rawLog.Topics) == 0 {
			continue
		}
		
		log := Log{
			Address: rawLog.Address,
			Topics:  rawLog.Topics,
			Data:    rawLog.Data,
		}
		
		// Check event type by topic
		switch rawLog.Topics[0] {
		case DeployingContractTopic:
			event, err := parseDeployingContractEvent(log)
			if err != nil {
				fmt.Printf("Warning: failed to parse DeployingContract event: %v\n", err)
				continue
			}
			events = append(events, event)
			
		case ContractDeployedTopic:
			event, err := parseContractDeployedEvent(log)
			if err != nil {
				fmt.Printf("Warning: failed to parse ContractDeployed event: %v\n", err)
				continue
			}
			events = append(events, event)
			
		case SafeTransactionQueuedTopic:
			event, err := parseSafeTransactionQueuedEvent(log)
			if err != nil {
				fmt.Printf("Warning: failed to parse SafeTransactionQueued event: %v\n", err)
				continue
			}
			events = append(events, event)
			
		case BundleSentTopic:
			event, err := parseBundleSentEvent(log)
			if err != nil {
				fmt.Printf("Warning: failed to parse BundleSent event: %v\n", err)
				continue
			}
			events = append(events, event)
		}
	}
	
	return events, nil
}

// ParseEvents extracts deployment events from forge output (legacy compatibility)
func ParseEvents(output *ForgeScriptOutput) ([]DeploymentEvent, error) {
	allEvents, err := ParseAllEvents(output)
	if err != nil {
		return nil, err
	}
	
	// Filter only deployment events
	var deploymentEvents []DeploymentEvent
	for _, event := range allEvents {
		if deployEvent, ok := event.(*ContractDeployedEvent); ok {
			deploymentEvents = append(deploymentEvents, *deployEvent)
		}
	}
	
	return deploymentEvents, nil
}