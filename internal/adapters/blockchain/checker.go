package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// CheckerAdapter implements the BlockchainChecker interface using ethclient
type CheckerAdapter struct {
	client      *ethclient.Client
	rpc         *rpc.Client
	rpcURL      string // Store RPC URL for cast run
	chainID     uint64
	forgeAdapter CastTracer // For cast run tracing
	projectRoot string
}

// CastTracer interface for tracing transactions with cast run
type CastTracer interface {
	RunCastTrace(ctx context.Context, rpcURL string, txHash string) ([]byte, error)
}

// NewCheckerAdapter creates a new blockchain checker adapter
func NewCheckerAdapter(forgeAdapter CastTracer, projectRoot string) *CheckerAdapter {
	return &CheckerAdapter{
		forgeAdapter: forgeAdapter,
		projectRoot:  projectRoot,
	}
}

// Connect establishes connection to the blockchain
// It's idempotent - if already connected to the same RPC URL, it reuses the connection
func (c *CheckerAdapter) Connect(ctx context.Context, rpcURL string, chainID uint64) error {
	// If already connected to the same RPC URL, reuse the connection
	if c.client != nil && c.rpcURL == rpcURL {
		// Verify chain ID still matches (quick check)
		if chainID != 0 && c.chainID != 0 && c.chainID != chainID {
			return fmt.Errorf("chain ID mismatch: already connected to chain %d, requested %d", c.chainID, chainID)
		}
		return nil
	}

	c.rpcURL = rpcURL // Store for cast run
	rpcClient, err := rpc.DialContext(ctx, rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RPC: %w", err)
	}
	c.rpc = rpcClient
	c.client = ethclient.NewClient(rpcClient)

	// Verify chain ID matches
	networkChainID, err := c.client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %w", err)
	}

	// If chainID was 0, use the network's chain ID
	if chainID == 0 {
		c.chainID = networkChainID.Uint64()
	} else if networkChainID.Uint64() != chainID {
		return fmt.Errorf("chain ID mismatch: expected %d, got %d", chainID, networkChainID.Uint64())
	} else {
		c.chainID = chainID
	}

	return nil
}

// CheckDeploymentExists checks if a contract exists at the given address
func (c *CheckerAdapter) CheckDeploymentExists(ctx context.Context, address string) (exists bool, reason string, err error) {
	if c.client == nil {
		return false, "", fmt.Errorf("not connected to blockchain")
	}

	addr := common.HexToAddress(address)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	code, err := c.client.CodeAt(ctx, addr, nil)
	if err != nil {
		return false, fmt.Sprintf("failed to check code: %v", err), nil
	}

	// If no code at address, contract doesn't exist
	if len(code) == 0 {
		return false, "no code at address", nil
	}

	return true, "", nil
}

// CheckTransactionExists checks if a transaction exists on-chain
func (c *CheckerAdapter) CheckTransactionExists(ctx context.Context, txHash string) (exists bool, blockNumber uint64, reason string, err error) {
	if c.client == nil {
		return false, 0, "", fmt.Errorf("not connected to blockchain")
	}

	hash := common.HexToHash(txHash)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	receipt, err := c.client.TransactionReceipt(ctx, hash)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return false, 0, "transaction not found on-chain", nil
		}
		// For other errors, return the error
		return false, 0, "", fmt.Errorf("failed to get transaction receipt: %w", err)
	}

	// Transaction exists
	if receipt.BlockNumber != nil {
		return true, receipt.BlockNumber.Uint64(), "", nil
	}

	return true, 0, "", nil
}

// CheckSafeContract checks if a Safe contract exists at the given address
func (c *CheckerAdapter) CheckSafeContract(ctx context.Context, safeAddress string) (exists bool, reason string, err error) {
	// For Safe contracts, we just check if code exists at the address
	// More sophisticated checks could verify it's actually a Safe contract
	return c.CheckDeploymentExists(ctx, safeAddress)
}

// GetTransaction fetches a transaction and its receipt
func (c *CheckerAdapter) GetTransaction(ctx context.Context, txHash string) (*types.Transaction, *types.Receipt, error) {
	if c.client == nil {
		return nil, nil, fmt.Errorf("not connected to blockchain")
	}

	hash := common.HexToHash(txHash)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Get transaction
	tx, isPending, err := c.client.TransactionByHash(ctx, hash)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	if isPending {
		return nil, nil, fmt.Errorf("transaction is still pending")
	}

	// Get receipt
	receipt, err := c.client.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get transaction receipt: %w", err)
	}

	return tx, receipt, nil
}

// GetCode fetches the bytecode at an address
func (c *CheckerAdapter) GetCode(ctx context.Context, address string) ([]byte, error) {
	if c.client == nil {
		return nil, fmt.Errorf("not connected to blockchain")
	}

	addr := common.HexToAddress(address)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	code, err := c.client.CodeAt(ctx, addr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get code: %w", err)
	}

	return code, nil
}

// TraceTransaction traces a transaction to find contract creations using cast run
func (c *CheckerAdapter) TraceTransaction(ctx context.Context, txHash string) ([]models.ContractCreation, error) {
	// We need RPC URL to run cast run
	// This should be set when Connect is called, but we need to store it
	if c.rpcURL == "" {
		return nil, fmt.Errorf("RPC URL not set - must call Connect first")
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Use cast run --json to get the trace
	jsonOutput, err := c.forgeAdapter.RunCastTrace(ctx, c.rpcURL, txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to trace transaction with cast run: %w", err)
	}

	// Parse the JSON output from cast run
	var traceResult interface{}
	if err := json.Unmarshal(jsonOutput, &traceResult); err != nil {
		return nil, fmt.Errorf("failed to parse cast run JSON output: %w", err)
	}

	// Parse the trace result to find contract creations
	creations := parseCastRunTrace(traceResult)
	return creations, nil
}


// parseCastRunTrace parses the JSON output from cast run to find contract creations
// cast run --json returns a trace structure that can have different formats:
// 1. {"arena": [...]} - array of trace entries (newer format)
// 2. {"traces": [...]} - array of trace entries (older format)
// 3. Direct array of trace entries
// 4. Single trace entry object
func parseCastRunTrace(trace interface{}) []models.ContractCreation {
	var creations []models.ContractCreation

	switch t := trace.(type) {
	case map[string]interface{}:
		// Check for "arena" field (newer cast run format)
		if arena, ok := t["arena"].([]interface{}); ok {
			for _, arenaEntry := range arena {
				creations = append(creations, parseCastRunTraceEntry(arenaEntry)...)
			}
		}
		// Check for "traces" field (older format)
		if traces, ok := t["traces"].([]interface{}); ok {
			for _, traceEntry := range traces {
				creations = append(creations, parseCastRunTraceEntry(traceEntry)...)
			}
		}
		// Also check if it's directly a trace entry (single entry format)
		creations = append(creations, parseCastRunTraceEntry(trace)...)
	case []interface{}:
		// Direct array of trace entries
		for _, item := range t {
			creations = append(creations, parseCastRunTrace(item)...)
		}
	}

	return creations
}

// parseCastRunTraceEntry parses a single trace entry from cast run
// Newer format: {"trace": {"kind": "CREATE"|"CREATE2", "address": "0x..."}, ...}
// Older format: {"action": {"callType": "create", "to": "0x..."}, "result": {"address": "0x..."}}
func parseCastRunTraceEntry(entry interface{}) []models.ContractCreation {
	var creations []models.ContractCreation

	entryMap, ok := entry.(map[string]interface{})
	if !ok {
		return creations
	}

	// Check for newer format with "trace" field (arena format)
	if trace, ok := entryMap["trace"].(map[string]interface{}); ok {
		kind, hasKind := trace["kind"].(string)
		if hasKind {
			kindUpper := strings.ToUpper(kind)
			if kindUpper == "CREATE" || kindUpper == "CREATE2" {
				// Extract address from trace.address
				var address string
				if addr, ok := trace["address"].(string); ok && addr != "" && addr != "0x" {
					address = addr
				}

				if address != "" {
					addr := common.HexToAddress(address)
					if addr != (common.Address{}) {
						// Check if this is CREATE3 (through CreateX)
						// CreateX factory address: 0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed
						if caller, ok := trace["caller"].(string); ok {
							callerAddr := common.HexToAddress(caller)
							createXAddr := common.HexToAddress("0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed")
							if callerAddr == createXAddr {
								kindUpper = "CREATE3"
							}
						}

						creations = append(creations, models.ContractCreation{
							Address:      strings.ToLower(addr.Hex()),
							ContractName: "", // Will be filled in later if we can determine it
							Kind:         kindUpper,
						})
					}
				}
			}
		}
	}

	// Check for older format with "action" field
	if action, ok := entryMap["action"].(map[string]interface{}); ok {
		// Look for CREATE operations
		// In cast run format, CREATE operations have "callType": "create" or "create2"
		callType, hasCallType := action["callType"].(string)
		if !hasCallType {
			// Also check "type" field (alternative format)
			callType, hasCallType = entryMap["type"].(string)
		}

		if hasCallType {
			callTypeLower := strings.ToLower(callType)
			if callTypeLower == "create" || callTypeLower == "create2" {
				// Extract address from result (this is where the created contract address is)
				var address string
				if result, ok := entryMap["result"].(map[string]interface{}); ok {
					if addr, ok := result["address"].(string); ok && addr != "" && addr != "0x" {
						address = addr
					}
				}
				// Fallback: try to get from action.to (some formats put it here)
				if address == "" {
					if to, ok := action["to"].(string); ok && to != "" && to != "0x" {
						address = to
					}
				}
				// Another fallback: check if there's a "contractAddress" field
				if address == "" {
					if addr, ok := entryMap["contractAddress"].(string); ok && addr != "" && addr != "0x" {
						address = addr
					}
				}

				if address != "" {
					addr := common.HexToAddress(address)
					if addr != (common.Address{}) {
						kind := strings.ToUpper(callTypeLower)
						// Check if this is CREATE3 (through CreateX)
						if from, ok := action["from"].(string); ok {
							fromAddr := common.HexToAddress(from)
							createXAddr := common.HexToAddress("0xba5Ed099633D3B313e4D5F7bdc1305d3c28ba5Ed")
							if fromAddr == createXAddr {
								kind = "CREATE3"
							}
						}

						creations = append(creations, models.ContractCreation{
							Address:      strings.ToLower(addr.Hex()),
							ContractName: "", // Will be filled in later if we can determine it
							Kind:         kind,
						})
					}
				}
			}
		}
	}

	// Recursively check nested traces (subtraces)
	// cast run uses "subtraces" field for nested calls
	if subtraces, ok := entryMap["subtraces"].([]interface{}); ok {
		for _, subtrace := range subtraces {
			creations = append(creations, parseCastRunTraceEntry(subtrace)...)
		}
	}
	// Also check for "trace" field (alternative format)
	if trace, ok := entryMap["trace"].(map[string]interface{}); ok {
		creations = append(creations, parseCastRunTraceEntry(trace)...)
	}
	// Check for nested "calls" (some trace formats use this)
	if calls, ok := entryMap["calls"].([]interface{}); ok {
		for _, call := range calls {
			creations = append(creations, parseCastRunTraceEntry(call)...)
		}
	}

	return creations
}

// Ensure the adapter implements the interface
var _ usecase.BlockchainChecker = (*CheckerAdapter)(nil)
