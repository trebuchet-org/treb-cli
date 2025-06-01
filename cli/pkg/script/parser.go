package script

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// ForgeScriptOutput represents the main output structure from forge script --json
type ForgeScriptOutput struct {
	Logs    []string `json:"logs"`
	Success bool     `json:"success"`
	RawLogs []RawLog `json:"raw_logs"`
	GasUsed uint64   `json:"gas_used"`
}

// RawLog represents a raw log entry from the script output
type RawLog struct {
	Address common.Address `json:"address"`
	Topics  []common.Hash  `json:"topics"`
	Data    string         `json:"data"`
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

// ParsedForgeOutput contains all the parsed outputs from forge script
type ParsedForgeOutput struct {
	ScriptOutput  *ForgeScriptOutput
	GasEstimate   *GasEstimate
	StatusOutput  *StatusOutput
	BroadcastPath string
}

// ParseForgeOutput parses the JSON output from forge script
func ParseForgeOutput(output []byte) (*ForgeScriptOutput, error) {
	parsed, err := ParseCompleteForgeOutput(output)
	if err != nil {
		return nil, err
	}
	return parsed.ScriptOutput, nil
}

// ParseCompleteForgeOutput parses all JSON outputs from forge script
func ParseCompleteForgeOutput(output []byte) (*ParsedForgeOutput, error) {
	result := &ParsedForgeOutput{}

	// First try to parse the entire output as a single JSON object
	var mainOutput ForgeScriptOutput
	if err := json.Unmarshal(output, &mainOutput); err == nil {
		// Check if this looks like the main output (has raw_logs)
		if mainOutput.RawLogs != nil {
			result.ScriptOutput = &mainOutput
			return result, nil
		}
	}

	// If that fails, fall back to line-by-line parsing for multi-line output
	scanner := bufio.NewScanner(bytes.NewReader(output))

	// Increase buffer size to handle large JSON lines from forge
	const maxTokenSize = 200 * 1024 * 1024 // 200MB should be sufficient
	buf := make([]byte, maxTokenSize)
	scanner.Buffer(buf, maxTokenSize)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Skip non-JSON lines (like error logs, ANSI codes, etc)
		if !bytes.HasPrefix(line, []byte("{")) {
			continue
		}

		// Try to parse as main output (has raw_logs)
		if result.ScriptOutput == nil {
			var lineOutput ForgeScriptOutput
			if err := json.Unmarshal(line, &lineOutput); err == nil {
				if lineOutput.RawLogs != nil {
					result.ScriptOutput = &lineOutput
					continue
				}
			}
		}

		// Try to parse as gas estimate
		if result.GasEstimate == nil {
			var gasOutput GasEstimate
			if err := json.Unmarshal(line, &gasOutput); err == nil {
				if gasOutput.Chain != 0 {
					result.GasEstimate = &gasOutput
					continue
				}
			}
		}

		// Try to parse as status output (contains broadcast file path)
		if result.StatusOutput == nil {
			var statusOutput StatusOutput
			if err := json.Unmarshal(line, &statusOutput); err == nil {
				if statusOutput.Status != "" {
					result.StatusOutput = &statusOutput
					result.BroadcastPath = statusOutput.Transactions
					continue
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	if result.ScriptOutput == nil {
		return nil, fmt.Errorf("no valid forge script output found")
	}

	return result, nil
}

// ParseAllEvents extracts all types of events from forge output using generated ABI bindings
func ParseAllEvents(output *ForgeScriptOutput) ([]interface{}, error) {
	var events []interface{}
	parser := NewEventParser()

	for _, rawLog := range output.RawLogs {
		if len(rawLog.Topics) == 0 {
			continue
		}

		// Use the new event parser with RawLog directly
		event, err := parser.ParseEvent(rawLog)
		if err != nil {
			// Only log warnings for actual parsing errors, not unknown events
			if !strings.Contains(err.Error(), "unknown event signature") {
				fmt.Printf("Warning: failed to parse event %s: %v\n", rawLog.Topics[0].Hex(), err)
			}
			continue
		}

		events = append(events, event)
	}

	return events, nil
}

// ParseEvents function removed - deployment events are now extracted directly
// from AllEvents in executor.go using type switches on generated types
