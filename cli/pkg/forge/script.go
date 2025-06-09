package forge

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/creack/pty"
	"github.com/ethereum/go-ethereum/common"
)

// ScriptOptions contains options for running a script
type ScriptOptions struct {
	ScriptPath     string
	FunctionName   string            // Optional function to call
	FunctionArgs   []string          // Arguments for the function
	Network        string            // Network name from foundry.toml
	RpcUrl         string            // Direct RPC URL (overrides network)
	Profile        string            // Foundry profile
	EnvVars        map[string]string // Environment variables
	AdditionalArgs []string          // Additional forge arguments
	DryRun         bool              // Simulate only
	Broadcast      bool              // Broadcast transactions
	VerifyContract bool              // Verify on etherscan
	Debug          bool              // Enable debug output
	JSON           bool              // Output JSON format
}

// ScriptResult contains the parsed result of running a script
type ScriptResult struct {
	Success       bool
	RawOutput     []byte
	ParsedOutput  *ParsedOutput
	BroadcastPath string
	Error         error
}

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
	Address          *string                `json:"address,omitempty"`
	Logs             []string               `json:"logs"`
	Success          bool                   `json:"success"`
	RawLogs          []EventLog             `json:"raw_logs"`
	GasUsed          uint64                 `json:"gas_used"`
	LabeledAddresses map[string]string      `json:"labeled_addresses"`
	Returned         string                 `json:"returned"`
	Returns          map[string]interface{} `json:"returns"`
	Traces           interface{}            `json:"traces"`
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
	Steps                        []interface{}   `json:"steps"`
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

// Run executes a Foundry script with the given options
func (f *Forge) Run(opts ScriptOptions) (*ScriptResult, error) {
	// Build forge command arguments
	args := f.buildArgs(opts)

	// Build environment variables
	env := f.buildEnv(opts)

	if opts.Debug {
		fmt.Printf("Running: forge %s\n", strings.Join(args, " "))
		if len(env) > 0 {
			fmt.Printf("With env vars: %v\n", env)
		}
	}

	// Execute the script
	cmd := exec.Command("forge", args...)
	cmd.Dir = f.projectRoot
	cmd.Env = append(os.Environ(), env...)

	// Start with PTY for proper color handling
	ptyFile, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start pty: %w", err)
	}
	defer ptyFile.Close()

	result := &ScriptResult{
		Success: true, // Will be updated based on command exit
	}

	// Debug mode: direct copy to stdout
	if opts.Debug && !opts.JSON {
		// Simple copy for debug mode
		io.Copy(os.Stdout, ptyFile)

		// Wait for command to finish
		if err := cmd.Wait(); err != nil {
			result.Success = false
			result.Error = fmt.Errorf("forge script failed: %w", err)
		}

		return result, nil
	}

	// Normal mode: process output with scanner
	// Create debug directory for this run
	runUUID := fmt.Sprintf("%d", time.Now().Unix())
	debugDir := filepath.Join(f.projectRoot, "out", ".treb-debug", runUUID)
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		fmt.Printf("Warning: failed to create debug directory: %v\n", err)
		// Continue anyway
		debugDir = ""
	}

	// Create channels for parsed entities
	entityChan := make(chan ParsedEntity, 100)
	errChan := make(chan error, 1)

	// Collect all output for raw output
	var outputBuffer bytes.Buffer
	teeReader := io.TeeReader(ptyFile, &outputBuffer)

	// Start output processor
	processor := NewOutputProcessor(debugDir)

	// Process output in goroutine
	go func() {
		if err := processor.ProcessOutput(teeReader, entityChan); err != nil {
			errChan <- err
		}
		close(entityChan)
		close(errChan)
	}()

	// Collect parsed entities
	parsedOutput := &ParsedOutput{
		ConsoleLogs:  []string{},
		TraceOutputs: []TraceOutput{},
	}

	for entity := range entityChan {
		switch entity.Type {
		case "ScriptOutput":
			if output, ok := entity.Data.(ScriptOutput); ok {
				parsedOutput.ScriptOutput = &output
				// Extract console logs
				parsedOutput.ConsoleLogs = append(parsedOutput.ConsoleLogs, f.extractConsoleLogs(output.Logs)...)
			}
		case "GasEstimate":
			if gas, ok := entity.Data.(GasEstimate); ok {
				parsedOutput.GasEstimate = &gas
			}
		case "StatusOutput":
			if status, ok := entity.Data.(StatusOutput); ok {
				parsedOutput.StatusOutput = &status
				// Extract broadcast path
				if status.Transactions != "" {
					result.BroadcastPath = status.Transactions
				}
			}
		case "TraceOutput":
			if trace, ok := entity.Data.(TraceOutput); ok {
				parsedOutput.TraceOutputs = append(parsedOutput.TraceOutputs, trace)
			}
		case "TextOutput":
			if text, ok := entity.Data.(string); ok {
				parsedOutput.TextOutput = text
			}
		case "Receipt":
			if receipt, ok := entity.Data.(Receipt); ok {
				if parsedOutput.Receipts == nil {
					parsedOutput.Receipts = []Receipt{}
				}
				parsedOutput.Receipts = append(parsedOutput.Receipts, receipt)
			}
		}
	}

	// Check for processing errors
	select {
	case err := <-errChan:
		if err != nil {
			result.Error = fmt.Errorf("output processing error: %w", err)
		}
	default:
	}

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		result.Success = false
		if result.Error == nil {
			result.Error = fmt.Errorf("forge script failed: %w", err)
		}
	}

	// Set results
	result.RawOutput = outputBuffer.Bytes()
	result.ParsedOutput = parsedOutput

	// Print text output if script failed or in debug/verbose mode
	if parsedOutput.TextOutput != "" && (result.Error != nil || opts.Debug) {
		fmt.Println("\n📝 Forge Output:")
		fmt.Println(strings.Repeat("─", 50))
		fmt.Println(parsedOutput.TextOutput)
		fmt.Println(strings.Repeat("─", 50))
	}

	// Save debug output if requested
	if opts.Debug && opts.JSON && len(result.RawOutput) > 0 {
		f.saveDebugOutput(result.RawOutput)
	}

	// Find broadcast file if broadcast was enabled and no dry run
	if opts.Broadcast && !opts.DryRun && result.BroadcastPath == "" {
		result.BroadcastPath = f.findBroadcastFile(opts.ScriptPath, opts.Network)
	}

	return result, nil
}

// buildArgs builds the forge script command arguments
func (f *Forge) buildArgs(opts ScriptOptions) []string {
	args := []string{"script", opts.ScriptPath, "--ffi"}

	// Add function signature if specified
	if opts.FunctionName != "" {
		args = append(args, "--sig", opts.FunctionName)
		if len(opts.FunctionArgs) > 0 {
			args = append(args, opts.FunctionArgs...)
		}
	}

	// Network configuration
	if opts.RpcUrl != "" {
		args = append(args, "--rpc-url", opts.RpcUrl)
	} else if opts.Network != "" {
		args = append(args, "--rpc-url", opts.Network)
	}

	// Broadcast/dry run
	if opts.Broadcast {
		args = append(args, "--broadcast")
		if opts.VerifyContract {
			args = append(args, "--verify")
		}
	}

	// JSON output
	if opts.JSON {
		args = append(args, "--json")
	}

	args = append(args, "-vvvv")

	// Additional arguments
	args = append(args, opts.AdditionalArgs...)

	return args
}

// buildEnv builds environment variable array
func (f *Forge) buildEnv(opts ScriptOptions) []string {
	var env []string
	for k, v := range opts.EnvVars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Profile
	env = append(env, fmt.Sprintf("FOUNDRY_PROFILE=%s", opts.Profile))

	if opts.Debug {
		env = append(env, "QUIET=true")
	}

	return env
}

// ParseOutput parses the JSON output from forge script
func (f *Forge) ParseOutput(output []byte) (*ParsedOutput, error) {
	result := &ParsedOutput{}

	// First try to parse the entire output as a single JSON object
	var mainOutput ScriptOutput
	if err := json.Unmarshal(output, &mainOutput); err == nil {
		// Check if this looks like the main output (has raw_logs)
		if mainOutput.RawLogs != nil {
			result.ScriptOutput = &mainOutput
			result.ConsoleLogs = f.extractConsoleLogs(mainOutput.Logs)
			return result, nil
		}
	}

	// If that fails, parse line by line for multi-object output
	scanner := bufio.NewScanner(bytes.NewReader(output))

	// Increase buffer size to handle large JSON lines
	const maxTokenSize = 200 * 1024 * 1024 // 200MB
	buf := make([]byte, maxTokenSize)
	scanner.Buffer(buf, maxTokenSize)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 || !bytes.HasPrefix(line, []byte("{")) {
			continue
		}

		// Try to parse as main output
		if result.ScriptOutput == nil {
			var lineOutput ScriptOutput
			if err := json.Unmarshal(line, &lineOutput); err == nil {
				if lineOutput.RawLogs != nil {
					result.ScriptOutput = &lineOutput
					result.ConsoleLogs = f.extractConsoleLogs(lineOutput.Logs)
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

		// Try to parse as status output
		if result.StatusOutput == nil {
			var statusOutput StatusOutput
			if err := json.Unmarshal(line, &statusOutput); err == nil {
				if statusOutput.Status != "" {
					result.StatusOutput = &statusOutput
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

// extractConsoleLogs extracts console.log messages from forge logs
func (f *Forge) extractConsoleLogs(logs []string) []string {
	var consoleLogs []string
	for _, log := range logs {
		// Forge prefixes console.log with "Logs:"
		if strings.HasPrefix(log, "Logs:") {
			consoleLogs = append(consoleLogs, strings.TrimSpace(strings.TrimPrefix(log, "Logs:")))
		} else if strings.Contains(log, "console.log") {
			// Sometimes the format is different
			consoleLogs = append(consoleLogs, log)
		}
	}
	return consoleLogs
}

// findBroadcastFile attempts to find the broadcast file for a script
func (f *Forge) findBroadcastFile(scriptPath, network string) string {
	// Extract script name from path
	scriptName := filepath.Base(scriptPath)
	scriptName = strings.TrimSuffix(scriptName, filepath.Ext(scriptName))

	// Try to find the latest broadcast file
	broadcastDir := filepath.Join(f.projectRoot, "broadcast", scriptName)
	if network != "" {
		// Network might be a chain ID or name, try to resolve it
		// For now, just use it as is
		broadcastDir = filepath.Join(broadcastDir, network)
	}

	latestPath := filepath.Join(broadcastDir, "run-latest.json")
	if _, err := os.Stat(latestPath); err == nil {
		return latestPath
	}

	return ""
}

// saveDebugOutput saves raw output for debugging
func (f *Forge) saveDebugOutput(output []byte) {
	runUUID := fmt.Sprintf("%d", time.Now().Unix())
	debugDir := filepath.Join(f.projectRoot, "out", ".treb-debug", runUUID)

	if err := os.MkdirAll(debugDir, 0755); err != nil {
		fmt.Printf("Warning: failed to create debug directory: %v\n", err)
		return
	}

	// Save raw output
	rawPath := filepath.Join(debugDir, "raw-output.json")
	if err := os.WriteFile(rawPath, output, 0644); err != nil {
		fmt.Printf("Warning: failed to write raw output: %v\n", err)
	} else {
		fmt.Printf("Debug output saved to: %s\n", debugDir)
	}
}
