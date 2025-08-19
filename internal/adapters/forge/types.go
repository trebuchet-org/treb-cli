package forge

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// ScriptOptions contains options for running a script
type ScriptOptions struct {
	Script          string            // Script path
	FunctionName    string            // Optional function to call
	FunctionArgs    []string          // Arguments for the function
	Network         string            // Network name from foundry.toml
	RpcUrl          string            // Direct RPC URL (overrides network)
	Profile         string            // Foundry profile
	EnvVars         map[string]string // Environment variables
	AdditionalArgs  []string          // Additional forge arguments
	DryRun          bool              // Simulate only
	Broadcast       bool              // Broadcast transactions
	VerifyContract  bool              // Verify on etherscan
	Debug           bool              // Enable debug output
	JSON            bool              // Output JSON format
	UseLedger       bool              // Use ledger
	UseTrezor       bool              // Use trezor
	DerivationPaths []string          // Derivation paths for hardware wallets
	Libraries       []string          // Library addresses for linking (format: "file:lib:address")
}

// ScriptResult contains the parsed result of running a script
type ScriptResult struct {
	Script        string
	Success       bool
	RawOutput     []byte
	ParsedOutput  *domain.ForgeScriptOutput
	BroadcastPath string
	Error         error
}

// Type aliases for backward compatibility and internal use
type ScriptOutput = domain.ForgeMainOutput
type EventLog = domain.ForgeEventLog
type GasEstimate = domain.ForgeGasEstimate
type Receipt = domain.ForgeReceipt
type StatusOutput = domain.ForgeStatusOutput
type TraceOutput = domain.ForgeTraceOutput
type TraceNode = domain.ForgeTraceNode
type TraceInfo = domain.ForgeTraceInfo
type DecodedCall = domain.ForgeDecodedCall
type CallDataInfo = domain.ForgeCallDataInfo
type LogEntry = domain.ForgeLogEntry
type RawLogEntry = domain.ForgeRawLogEntry
type DecodedLog = domain.ForgeDecodedLog
type OrderItem = domain.ForgeOrderItem
type TraceWithLabel = domain.ForgeTraceWithLabel

// Custom UnmarshalJSON for TraceWithLabel
func UnmarshalTraceWithLabel(data []byte) (*domain.ForgeTraceWithLabel, error) {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	if len(raw) != 2 {
		return nil, fmt.Errorf("expected 2 items, got %d", len(raw))
	}

	var result domain.ForgeTraceWithLabel
	if err := json.Unmarshal(raw[0], &result.Label); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(raw[1], &result.Trace); err != nil {
		return nil, err
	}

	return &result, nil
}

// Stage represents the current execution stage
type Stage = domain.ForgeStage

const (
	StageSimulating   = domain.ForgeStageSimulating
	StageBroadcasting = domain.ForgeStageBroadcasting
	StageCompleted    = domain.ForgeStageCompleted
)

// ParsedEntity represents different types of parsed output
type ParsedEntity struct {
	Type    string
	Data    interface{}
	Stage   Stage
	RawLine string
}

// StageInfo tracks information about each stage
type StageInfo struct {
	Stage     Stage
	StartTime time.Time
	EndTime   time.Time
	Completed bool
	Skipped   bool
	Lines     int
}