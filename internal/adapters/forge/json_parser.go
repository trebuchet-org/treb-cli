package forge

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// ForgeJSONOutput represents the structure of forge script --json output
type ForgeJSONOutput struct {
	Success          bool                   `json:"success"`
	Logs             []string               `json:"logs"`
	GasUsed          uint64                 `json:"gas_used"`
	RawLogs          []ForgeEventLog        `json:"raw_logs"`
	LabeledAddresses map[string]string      `json:"labeled_addresses"`
	Traces           []interface{}          `json:"traces"`
	Returns          map[string]interface{} `json:"returns"`
}

// ForgeEventLog represents a raw log entry from forge output
type ForgeEventLog struct {
	Address common.Address   `json:"address"`
	Topics  []common.Hash    `json:"topics"`
	Data    string          `json:"data"`
}

// ParseForgeJSONOutput parses the JSON output from forge script
func ParseForgeJSONOutput(output string) (*ForgeJSONOutput, error) {
	// The output may contain multiple JSON objects on separate lines
	// We need to find the main script output object
	scanner := bufio.NewScanner(strings.NewReader(output))
	
	// Set a larger buffer for long JSON lines
	const maxTokenSize = 10 * 1024 * 1024 // 10MB
	buf := make([]byte, maxTokenSize)
	scanner.Buffer(buf, maxTokenSize)
	
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		
		// Skip empty lines
		if line == "" {
			continue
		}
		
		// Try to parse as JSON
		if strings.HasPrefix(line, "{") {
			var forgeOutput ForgeJSONOutput
			if err := json.Unmarshal([]byte(line), &forgeOutput); err == nil {
				// Check if this is the main output (has raw_logs field)
				if forgeOutput.RawLogs != nil {
					return &forgeOutput, nil
				}
			}
		}
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning output: %w", err)
	}
	
	// If we couldn't find JSON output, return nil
	return nil, nil
}