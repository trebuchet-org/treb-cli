package forge

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// ParsedOutput contains all parsed outputs from forge script
type ParsedOutput struct {
	ScriptOutput *ScriptOutput // Main script output with logs and events
	GasEstimate  *GasEstimate  // Gas estimation
	StatusOutput *StatusOutput // Status with broadcast path
	TraceOutputs []TraceOutput // Detailed execution traces (can be multiple)
	Receipts     []Receipt     // Transaction receipts for broadcast transactions
	ConsoleLogs  []string      // Extracted console.log messages
	TextOutput   string        // Raw text output from forge (non-JSON lines)
}

// ScriptOutput represents the main output structure from forge script --json
type ScriptOutput struct {
	Address          *string           `json:"address,omitempty"`
	Logs             []string          `json:"logs"`
	Success          bool              `json:"success"`
	RawLogs          []EventLog        `json:"raw_logs"`
	GasUsed          uint64            `json:"gas_used"`
	LabeledAddresses map[string]string `json:"labeled_addresses"`
	Returned         string            `json:"returned"`
	Returns          map[string]any    `json:"returns"`
	Traces           []TraceWithLabel  `json:"traces"`
}

type TraceWithLabel struct {
	Label string
	Trace TraceOutput
}

// EventLog represents a raw log entry from the script output
type EventLog struct {
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
	TokenSymbol             string `json:"token_symbol"`
}

type Receipt struct {
	Chain           string `json:"chain"`
	Status          string `json:"status"`
	TxHash          string `json:"tx_hash"`
	ContractAddress string `json:"contract_address"`
	BlockNumber     uint64 `json:"block_number"`
	GasUsed         uint64 `json:"gas_used"`
	GasPrice        uint64 `json:"gas_price"`
}

// StatusOutput represents the status output from forge
type StatusOutput struct {
	Status       string `json:"status"`
	Transactions string `json:"transactions"` // Path to broadcast file
	Sensitive    string `json:"sensitive"`
}

// TraceOutput represents the trace output from forge script with detailed execution traces
type TraceOutput struct {
	Arena []TraceNode `json:"arena"`
}

// TraceNode represents a single node in the execution trace tree
type TraceNode struct {
	Parent   *int        `json:"parent"`
	Children []int       `json:"children"`
	Idx      int         `json:"idx"`
	Trace    TraceInfo   `json:"trace"`
	Logs     []LogEntry  `json:"logs"`
	Ordering []OrderItem `json:"ordering"`
}

// TraceInfo contains detailed trace information for a single call
type TraceInfo struct {
	Depth                        int             `json:"depth"`
	Success                      bool            `json:"success"`
	Caller                       common.Address  `json:"caller"`
	Address                      common.Address  `json:"address"`
	MaybePrecompile              *bool           `json:"maybe_precompile"`
	SelfdestructAddress          *common.Address `json:"selfdestruct_address"`
	SelfdestructRefundTarget     *common.Address `json:"selfdestruct_refund_target"`
	SelfdestructTransferredValue *string         `json:"selfdestruct_transferred_value"`
	Kind                         string          `json:"kind"`
	Value                        string          `json:"value"`
	Data                         string          `json:"data"`
	Output                       string          `json:"output"`
	GasUsed                      uint64          `json:"gas_used"`
	GasLimit                     uint64          `json:"gas_limit"`
	Status                       string          `json:"status"`
	Steps                        []any           `json:"steps"`
	Decoded                      *DecodedCall    `json:"decoded"`
}

// DecodedCall represents decoded call data
type DecodedCall struct {
	Label      string        `json:"label"`
	ReturnData string        `json:"return_data"`
	CallData   *CallDataInfo `json:"call_data"`
}

// CallDataInfo contains decoded call data information
type CallDataInfo struct {
	Signature string   `json:"signature"`
	Args      []string `json:"args"`
}

// LogEntry represents a log entry in the trace
type LogEntry struct {
	RawLog   RawLogEntry `json:"raw_log"`
	Decoded  DecodedLog  `json:"decoded"`
	Position int         `json:"position"`
}

// RawLogEntry represents raw log data
type RawLogEntry struct {
	Topics []common.Hash `json:"topics"`
	Data   string        `json:"data"`
}

// DecodedLog represents decoded log information
type DecodedLog struct {
	Name   string     `json:"name"`
	Params [][]string `json:"params"`
}

// OrderItem represents an item in the execution ordering
type OrderItem struct {
	Call *int `json:"Call,omitempty"`
	Log  *int `json:"Log,omitempty"`
}

func (traceWithLabel *TraceWithLabel) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if len(raw) != 2 {
		return fmt.Errorf("expected 2 items, got %d", len(raw))
	}

	if err := json.Unmarshal(raw[0], &traceWithLabel.Label); err != nil {
		return err
	}

	return json.Unmarshal(raw[1], &traceWithLabel.Trace)
}
