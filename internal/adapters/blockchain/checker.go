package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// CheckerAdapter implements the BlockchainChecker interface using ethclient
type CheckerAdapter struct {
	client       *ethclient.Client
	rpc          *rpc.Client
	rpcURL       string
	chainID      uint64
	forgeAdapter CastTracer
	projectRoot  string
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
func (c *CheckerAdapter) Connect(ctx context.Context, rpcURL string, chainID uint64) error {
	if c.client != nil && c.rpcURL == rpcURL {
		if chainID != 0 && c.chainID != 0 && c.chainID != chainID {
			return fmt.Errorf("chain ID mismatch: already connected to chain %d, requested %d", c.chainID, chainID)
		}
		return nil
	}

	c.rpcURL = rpcURL
	rpcClient, err := rpc.DialContext(ctx, rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RPC: %w", err)
	}
	c.rpc = rpcClient
	c.client = ethclient.NewClient(rpcClient)

	networkChainID, err := c.client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %w", err)
	}

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
		return false, 0, "", fmt.Errorf("failed to get transaction receipt: %w", err)
	}

	if receipt.BlockNumber != nil {
		return true, receipt.BlockNumber.Uint64(), "", nil
	}

	return true, 0, "", nil
}

// CheckSafeContract checks if a Safe contract exists at the given address
func (c *CheckerAdapter) CheckSafeContract(ctx context.Context, safeAddress string) (exists bool, reason string, err error) {
	// TODO: Check that it's actually a Safe contract
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

	tx, isPending, err := c.client.TransactionByHash(ctx, hash)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	if isPending {
		return nil, nil, fmt.Errorf("transaction is still pending")
	}

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
	if c.rpcURL == "" {
		return nil, fmt.Errorf("RPC URL not set - must call Connect first")
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	jsonOutput, err := c.forgeAdapter.RunCastTrace(ctx, c.rpcURL, txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to trace transaction with cast run: %w", err)
	}

	var traceResult interface{}
	if err := json.Unmarshal(jsonOutput, &traceResult); err != nil {
		return nil, fmt.Errorf("failed to parse cast run JSON output: %w", err)
	}

	creations := parseCastRunTrace(traceResult)

	creations = filterIntermediateContracts(traceResult, creations)

	if len(creations) > 0 {
		creations = c.detectProxyRelationships(ctx, creations)
	}

	return creations, nil
}

// parseCastRunTrace parses the JSON output from cast run to find contract creations
func parseCastRunTrace(trace interface{}) []models.ContractCreation {
	var creations []models.ContractCreation
	var traceLogs []interface{}

	switch t := trace.(type) {
	case map[string]interface{}:
		if logs, ok := t["logs"].([]interface{}); ok {
			traceLogs = logs
		}

		if arena, ok := t["arena"].([]interface{}); ok {
			for _, arenaEntry := range arena {
				creations = append(creations, parseCastRunTraceEntry(arenaEntry, traceLogs)...)
			}
		}
		if traces, ok := t["traces"].([]interface{}); ok {
			for _, traceEntry := range traces {
				creations = append(creations, parseCastRunTraceEntry(traceEntry, traceLogs)...)
			}
		}
		creations = append(creations, parseCastRunTraceEntry(trace, traceLogs)...)
	case []interface{}:
		for _, item := range t {
			creations = append(creations, parseCastRunTrace(item)...)
		}
	}

	return creations
}

// parseCastRunTraceEntry parses a single trace entry from cast run
func parseCastRunTraceEntry(entry interface{}, traceLogs []interface{}) []models.ContractCreation {
	var creations []models.ContractCreation

	entryMap, ok := entry.(map[string]interface{})
	if !ok {
		return creations
	}

	if trace, ok := entryMap["trace"].(map[string]interface{}); ok {
		kind, hasKind := trace["kind"].(string)
		if hasKind {
			kindUpper := strings.ToUpper(kind)
			if kindUpper == "CREATE" || kindUpper == "CREATE2" {
				var address string
				if addr, ok := trace["address"].(string); ok && addr != "" && addr != "0x" {
					address = addr
				}

				if address != "" {
					addr := common.HexToAddress(address)
					if addr != (common.Address{}) {
						if kindUpper == "CREATE2" && hasCreate3EventInLogs(traceLogs) {
							kindUpper = "CREATE3"
						}

						creations = append(creations, models.ContractCreation{
							Address:      strings.ToLower(addr.Hex()),
							ContractName: "",
							Kind:         kindUpper,
						})
					}
				}
			}
		}
	}

	// Check for older format with "action" field
	if action, ok := entryMap["action"].(map[string]interface{}); ok {
		callType, hasCallType := action["callType"].(string)
		if !hasCallType {
			callType, hasCallType = entryMap["type"].(string)
		}

		if hasCallType {
			callTypeLower := strings.ToLower(callType)
			if callTypeLower == "create" || callTypeLower == "create2" {
				var address string
				if result, ok := entryMap["result"].(map[string]interface{}); ok {
					if addr, ok := result["address"].(string); ok && addr != "" && addr != "0x" {
						address = addr
					}
				}
				if address == "" {
					if to, ok := action["to"].(string); ok && to != "" && to != "0x" {
						address = to
					}
				}
				if address == "" {
					if addr, ok := entryMap["contractAddress"].(string); ok && addr != "" && addr != "0x" {
						address = addr
					}
				}

				if address != "" {
					addr := common.HexToAddress(address)
					if addr != (common.Address{}) {
						kind := strings.ToUpper(callTypeLower)
						if kind == "CREATE2" && hasCreate3EventInLogs(traceLogs) {
							kind = "CREATE3"
						}

						creations = append(creations, models.ContractCreation{
							Address:      strings.ToLower(addr.Hex()),
							ContractName: "",
							Kind:         kind,
						})
					}
				}
			}
		}
	}

	// Recursively check nested traces (subtraces)
	if subtraces, ok := entryMap["subtraces"].([]interface{}); ok {
		for _, subtrace := range subtraces {
			creations = append(creations, parseCastRunTraceEntry(subtrace, traceLogs)...)
		}
	}
	if trace, ok := entryMap["trace"].(map[string]interface{}); ok {
		creations = append(creations, parseCastRunTraceEntry(trace, traceLogs)...)
	}
	if calls, ok := entryMap["calls"].([]interface{}); ok {
		for _, call := range calls {
			creations = append(creations, parseCastRunTraceEntry(call, traceLogs)...)
		}
	}

	return creations
}

// hasCreate3EventInLogs checks if logs contain a CREATE2 event
func hasCreate3EventInLogs(logs []interface{}) bool {
	create3Topic := "0x2feea65dd4e9f9cbd86b74b7734210c59a1b2981b5b137bd0ee3e208200c9067"
	create3TopicNormalized := strings.ToLower(strings.TrimPrefix(create3Topic, "0x"))

	for _, logEntry := range logs {
		logMap, ok := logEntry.(map[string]interface{})
		if !ok {
			continue
		}

		var topics []interface{}
		if rawLog, ok := logMap["raw_log"].(map[string]interface{}); ok {
			if topicList, ok := rawLog["topics"].([]interface{}); ok {
				topics = topicList
			}
		} else {
			if topicList, ok := logMap["topics"].([]interface{}); ok {
				topics = topicList
			}
		}

		if len(topics) > 0 {
			var topic0 string
			switch v := topics[0].(type) {
			case string:
				topic0 = v
			case common.Hash:
				topic0 = v.Hex()
			case []byte:
				topic0 = common.BytesToHash(v).Hex()
			default:
				topic0Str := fmt.Sprintf("%v", v)
				if strings.HasPrefix(topic0Str, "0x") || len(topic0Str) == 64 {
					topic0 = topic0Str
				} else {
					continue
				}
			}

			topic0Normalized := strings.ToLower(strings.TrimPrefix(topic0, "0x"))

			if topic0Normalized == create3TopicNormalized {
				return true
			}
		}
	}

	return false
}

// filterIntermediateContracts filters out intermediate CREATE2 contracts
// that were only used to deploy another contract. These contracts emit specific log events.
func filterIntermediateContracts(traceResult interface{}, creations []models.ContractCreation) []models.ContractCreation {
	traceMap, ok := traceResult.(map[string]interface{})
	if !ok {
		return creations
	}

	arena, ok := traceMap["arena"].([]interface{})
	if !ok || len(arena) == 0 {
		return creations
	}

	var firstEntry map[string]interface{}
	for _, entry := range arena {
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if idx, ok := entryMap["idx"].(float64); ok && idx == 0 {
			firstEntry = entryMap
			break
		}
	}

	if firstEntry == nil {
		return creations
	}

	// Check if this entry has logs with the specific topics
	// Topic 1: 0x2feea65dd4e9f9cbd86b74b7734210c59a1b2981b5b137bd0ee3e208200c9067 (CREATE3 event)
	// Topic 2: 0x4db17dd5e4732fb6da34a148104a592783ca119a1e7bb8829eba6cbadef0b511 (contract creation event)
	create3Topic := "0x2feea65dd4e9f9cbd86b74b7734210c59a1b2981b5b137bd0ee3e208200c9067"
	deploymentTopic := "0x4db17dd5e4732fb6da34a148104a592783ca119a1e7bb8829eba6cbadef0b511"

	// Get logs from the trace entry
	// In cast run format, logs are at the top level of the entry
	var logs []interface{}
	if logList, ok := firstEntry["logs"].([]interface{}); ok {
		logs = logList
	}

	hasCreate3Log := false
	hasDeploymentLog := false

	for _, logEntry := range logs {
		logMap, ok := logEntry.(map[string]interface{})
		if !ok {
			continue
		}

		// Check for raw_log format (cast run uses this)
		var topics []interface{}
		if rawLog, ok := logMap["raw_log"].(map[string]interface{}); ok {
			if topicList, ok := rawLog["topics"].([]interface{}); ok {
				topics = topicList
			}
		} else {
			// Also check if topics are directly in the log entry
			if topicList, ok := logMap["topics"].([]interface{}); ok {
				topics = topicList
			}
		}

		if len(topics) > 0 {
			var topic0 string
			// Topics can be strings, common.Hash objects, or other formats
			switch v := topics[0].(type) {
			case string:
				topic0 = v
			case common.Hash:
				topic0 = v.Hex()
			case []byte:
				topic0 = common.BytesToHash(v).Hex()
			default:
				// Try to convert to string if it's something else
				topic0Str := fmt.Sprintf("%v", v)
				// Try to parse as hex if it looks like it
				if strings.HasPrefix(topic0Str, "0x") || len(topic0Str) == 64 {
					topic0 = topic0Str
				} else {
					continue
				}
			}

			// Normalize topic strings (remove 0x prefix, lowercase)
			topic0Normalized := strings.ToLower(strings.TrimPrefix(topic0, "0x"))
			create3TopicNormalized := strings.ToLower(strings.TrimPrefix(create3Topic, "0x"))
			deploymentTopicNormalized := strings.ToLower(strings.TrimPrefix(deploymentTopic, "0x"))

			// Compare normalized topics
			if topic0Normalized == create3TopicNormalized {
				hasCreate3Log = true
			}
			if topic0Normalized == deploymentTopicNormalized {
				hasDeploymentLog = true
			}
		}
	}

	// If both logs are present, filter out the CREATE2/CREATE3 intermediate contract
	if hasCreate3Log && hasDeploymentLog {
		// Find the address of the first entry's contract (the CREATE2/CREATE3 intermediate)
		var firstAddress string
		if trace, ok := firstEntry["trace"].(map[string]interface{}); ok {
			// Check if this trace is a CREATE2 or CREATE3
			if kind, ok := trace["kind"].(string); ok {
				kindUpper := strings.ToUpper(kind)
				if kindUpper == "CREATE2" || kindUpper == "CREATE3" {
					if addr, ok := trace["address"].(string); ok && addr != "" && addr != "0x" {
						firstAddress = strings.ToLower(common.HexToAddress(addr).Hex())
					}
				}
			}
		}

		// If we didn't find the address in trace, try to find it from creations
		// by matching the kind (should be the first CREATE2 or CREATE3 we found)
		if firstAddress == "" {
			for _, creation := range creations {
				if creation.Kind == "CREATE2" || creation.Kind == "CREATE3" {
					firstAddress = strings.ToLower(creation.Address)
					break
				}
			}
		}

		// Remove the CREATE2/CREATE3 intermediate contract from creations if it matches
		if firstAddress != "" {
			filtered := make([]models.ContractCreation, 0, len(creations))
			for _, creation := range creations {
				creationAddrLower := strings.ToLower(creation.Address)
				if creationAddrLower != firstAddress {
					filtered = append(filtered, creation)
				}
			}
			return filtered
		}
	}

	return creations
}

// detectProxyRelationships detects proxy/implementation relationships by calling _getImplementation()
// on contracts. If a contract returns an implementation address that matches another contract
// in the list, it's marked as a proxy.
func (c *CheckerAdapter) detectProxyRelationships(
	ctx context.Context,
	creations []models.ContractCreation,
) []models.ContractCreation {

	if c.client == nil {
		return creations // Can't detect if not connected
	}

	// ABI with both proxy access patterns
	proxyABI := `[
		{
			"constant":true,
			"inputs":[],
			"name":"_getImplementation",
			"outputs":[
				{
					"internalType":"address",
					"name":"implementation",
					"type":"address"
				}
			],
			"stateMutability":"view",
			"type":"function"
		},
		{
			"constant":true,
			"inputs":[],
			"name":"implementation",
			"outputs":[
				{
					"internalType":"address",
					"name":"implementation",
					"type":"address"
				}
			],
			"stateMutability":"view",
			"type":"function"
		}
	]`

	parsedABI, err := abi.JSON(strings.NewReader(proxyABI))
	if err != nil {
		return creations
	}

	// Try methods in order
	methodNames := []string{
		"_getImplementation",
		"implementation",
	}

	for i := range creations {
		contractAddr := common.HexToAddress(creations[i].Address)

		var implementationAddr *common.Address

		for _, methodName := range methodNames {
			method, ok := parsedABI.Methods[methodName]
			if !ok {
				continue
			}

			callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			result, err := c.client.CallContract(
				callCtx,
				ethereum.CallMsg{
					To:   &contractAddr,
					Data: method.ID, // 4-byte selector
				},
				nil,
			)
			cancel()

			if err != nil || len(result) == 0 {
				continue
			}

			outputs, err := method.Outputs.Unpack(result)
			if err != nil || len(outputs) == 0 {
				continue
			}

			addr, ok := outputs[0].(common.Address)
			if !ok {
				continue
			}

			implementationAddr = &addr
			break
		}

		if implementationAddr == nil {
			continue
		}

		// Check if the implementation address is valid
		implAddrStr := strings.ToLower(implementationAddr.Hex())
		if implAddrStr == "" || implAddrStr == "0x0000000000000000000000000000000000000000" {
			continue
		}

		creations[i].IsProxy = true
		creations[i].Implementation = implAddrStr
	}

	return creations
}

// Ensure the adapter implements the interface
var _ usecase.BlockchainChecker = (*CheckerAdapter)(nil)
