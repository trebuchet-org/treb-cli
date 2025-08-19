package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// ForgeScriptOutput represents the complete parsed output from forge script execution
type ForgeScriptOutput struct {
	ScriptOutput *ForgeMainOutput      // Main script output with logs and events
	GasEstimate  *ForgeGasEstimate     // Gas estimation
	StatusOutput *ForgeStatusOutput    // Status with broadcast path
	TraceOutputs []ForgeTraceOutput    // Detailed execution traces (can be multiple)
	Receipts     []ForgeReceipt        // Transaction receipts for broadcast transactions
	ConsoleLogs  []string              // Extracted console.log messages
	TextOutput   string                // Raw text output from forge (non-JSON lines)
	Stages       []ForgeExecutionStage // Execution stages (Simulating, Broadcasting, etc)
}

// ForgeMainOutput represents the main output structure from forge script --json
type ForgeMainOutput struct {
	Address          *string                `json:"address,omitempty"`
	Logs             []string               `json:"logs"`
	Success          bool                   `json:"success"`
	RawLogs          []ForgeEventLog        `json:"raw_logs"`
	GasUsed          uint64                 `json:"gas_used"`
	LabeledAddresses map[string]string      `json:"labeled_addresses"`
	Returned         string                 `json:"returned"`
	Returns          map[string]interface{} `json:"returns"`
	Traces           []ForgeTraceWithLabel  `json:"traces"`
}

// ForgeEventLog represents a raw log entry from the script output
type ForgeEventLog struct {
	Address common.Address `json:"address"`
	Topics  []common.Hash  `json:"topics"`
	Data    string         `json:"data"`
}

// ForgeGasEstimate represents gas estimation output
type ForgeGasEstimate struct {
	Chain                   uint64 `json:"chain"`
	EstimatedGasPrice       string `json:"estimated_gas_price"`
	EstimatedTotalGasUsed   uint64 `json:"estimated_total_gas_used"`
	EstimatedAmountRequired string `json:"estimated_amount_required"`
	TokenSymbol             string `json:"token_symbol"`
}

// ForgeReceipt represents a transaction receipt from forge
type ForgeReceipt struct {
	Chain           string `json:"chain"`
	Status          string `json:"status"`
	TxHash          string `json:"tx_hash"`
	ContractAddress string `json:"contract_address"`
	BlockNumber     uint64 `json:"block_number"`
	GasUsed         uint64 `json:"gas_used"`
	GasPrice        uint64 `json:"gas_price"`
}

// ForgeStatusOutput represents the status output from forge
type ForgeStatusOutput struct {
	Status       string `json:"status"`
	Transactions string `json:"transactions"` // Path to broadcast file
	Sensitive    string `json:"sensitive"`
}

// ForgeTraceOutput represents the trace output from forge script with detailed execution traces
type ForgeTraceOutput struct {
	Arena []ForgeTraceNode `json:"arena"`
}

// ForgeTraceNode represents a single node in the execution trace tree
type ForgeTraceNode struct {
	Parent   *int              `json:"parent"`
	Children []int             `json:"children"`
	Idx      int               `json:"idx"`
	Trace    ForgeTraceInfo    `json:"trace"`
	Logs     []ForgeLogEntry   `json:"logs"`
	Ordering []ForgeOrderItem  `json:"ordering"`
}

// ForgeTraceInfo contains detailed trace information for a single call
type ForgeTraceInfo struct {
	Depth                        int               `json:"depth"`
	Success                      bool              `json:"success"`
	Caller                       common.Address    `json:"caller"`
	Address                      common.Address    `json:"address"`
	MaybePrecompile              *bool             `json:"maybe_precompile"`
	SelfdestructAddress          *common.Address   `json:"selfdestruct_address"`
	SelfdestructRefundTarget     *common.Address   `json:"selfdestruct_refund_target"`
	SelfdestructTransferredValue *string           `json:"selfdestruct_transferred_value"`
	Kind                         string            `json:"kind"`
	Value                        string            `json:"value"`
	Data                         string            `json:"data"`
	Output                       string            `json:"output"`
	GasUsed                      uint64            `json:"gas_used"`
	GasLimit                     uint64            `json:"gas_limit"`
	Status                       string            `json:"status"`
	Steps                        []interface{}     `json:"steps"`
	Decoded                      *ForgeDecodedCall `json:"decoded"`
}

// ForgeDecodedCall represents decoded call data
type ForgeDecodedCall struct {
	Label      string           `json:"label"`
	ReturnData string           `json:"return_data"`
	CallData   *ForgeCallDataInfo `json:"call_data"`
}

// ForgeCallDataInfo contains decoded call data information
type ForgeCallDataInfo struct {
	Signature string   `json:"signature"`
	Args      []string `json:"args"`
}

// ForgeLogEntry represents a log entry in the trace
type ForgeLogEntry struct {
	RawLog   ForgeRawLogEntry `json:"raw_log"`
	Decoded  ForgeDecodedLog  `json:"decoded"`
	Position int              `json:"position"`
}

// ForgeRawLogEntry represents raw log data
type ForgeRawLogEntry struct {
	Topics []common.Hash `json:"topics"`
	Data   string        `json:"data"`
}

// ForgeDecodedLog represents decoded log information
type ForgeDecodedLog struct {
	Name   string     `json:"name"`
	Params [][]string `json:"params"`
}

// ForgeOrderItem represents an item in the execution ordering
type ForgeOrderItem struct {
	Call *int `json:"Call,omitempty"`
	Log  *int `json:"Log,omitempty"`
}

// ForgeTraceWithLabel wraps a trace with a label
type ForgeTraceWithLabel struct {
	Label string
	Trace ForgeTraceOutput
}

// UnmarshalJSON custom unmarshal for ForgeTraceWithLabel
func (f *ForgeTraceWithLabel) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if len(raw) != 2 {
		return fmt.Errorf("expected 2 items, got %d", len(raw))
	}

	if err := json.Unmarshal(raw[0], &f.Label); err != nil {
		return err
	}

	return json.Unmarshal(raw[1], &f.Trace)
}

// ForgeExecutionStage represents a stage of forge execution
type ForgeExecutionStage struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Completed bool
	Skipped   bool
	Duration  time.Duration
}

// ForgeStage represents the current execution stage
type ForgeStage string

const (
	ForgeStageSimulating   ForgeStage = "Simulating"
	ForgeStageBroadcasting ForgeStage = "Broadcasting"
	ForgeStageCompleted    ForgeStage = "Completed"
)