package deployment

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/trebuchet-org/treb-cli/cli/pkg/registry"
	"github.com/trebuchet-org/treb-cli/cli/pkg/types"
)

// Event signatures (keccak256 of event signature)
var (
	// ContractDeployed(bytes32 indexed operationId, address indexed sender, address deployedAddress, bytes32 salt, bytes32 initCodeHash, string createStrategy)
	ContractDeployedTopic = crypto.Keccak256Hash([]byte("ContractDeployed(bytes32,address,address,bytes32,bytes32,string)"))
	
	// OperationSent(bytes32 indexed operationId, address indexed sender, string label)
	OperationSentTopic = crypto.Keccak256Hash([]byte("OperationSent(bytes32,address,string)"))
	
	// SafeTransactionQueued(bytes32 indexed operationId, address indexed safe, uint256 nonce)
	SafeTransactionQueuedTopic = crypto.Keccak256Hash([]byte("SafeTransactionQueued(bytes32,address,uint256)"))
)

// DeploymentEvent represents a deployment event from the script
type DeploymentEvent struct {
	OperationID    common.Hash
	Sender         common.Address
	DeployedAddress common.Address
	Salt           common.Hash
	InitCodeHash   common.Hash
	CreateStrategy string
	TransactionHash common.Hash
	BlockNumber    uint64
}

// OperationEvent represents an operation event
type OperationEvent struct {
	OperationID common.Hash
	Sender      common.Address
	Label       string
}

// ScriptOutput represents the parsed output from forge script --json
type ScriptOutput struct {
	Logs        []Log        `json:"logs"`
	Receipts    []Receipt    `json:"receipts"`
}

// Log represents a log entry from the script output
type Log struct {
	Address common.Address `json:"address"`
	Topics  []common.Hash  `json:"topics"`
	Data    string         `json:"data"`
}

// Receipt represents a transaction receipt
type Receipt struct {
	TransactionHash common.Hash `json:"transactionHash"`
	BlockNumber     uint64      `json:"blockNumber"`
	Logs            []Log       `json:"logs"`
}

// ParseScriptEvents parses deployment events from forge script output
func ParseScriptEvents(output []byte) ([]DeploymentEvent, error) {
	// The output might contain multiple JSON objects, we need to find the receipts
	// Forge outputs JSON in a specific format when using --json flag
	
	var events []DeploymentEvent
	
	// Try to parse as script output
	var scriptOutput ScriptOutput
	if err := json.Unmarshal(output, &scriptOutput); err != nil {
		// Try to find JSON in the output (forge might output other text)
		jsonStart := strings.Index(string(output), "{")
		if jsonStart >= 0 {
			jsonData := output[jsonStart:]
			if err := json.Unmarshal(jsonData, &scriptOutput); err != nil {
				return nil, fmt.Errorf("failed to parse script output: %w", err)
			}
		} else {
			return nil, fmt.Errorf("no JSON found in output")
		}
	}
	
	// Process receipts
	for _, receipt := range scriptOutput.Receipts {
		for _, log := range receipt.Logs {
			if len(log.Topics) == 0 {
				continue
			}
			
			// Check if this is a ContractDeployed event
			if log.Topics[0] == ContractDeployedTopic {
				event, err := parseContractDeployedEvent(log, receipt.TransactionHash, receipt.BlockNumber)
				if err != nil {
					// Log warning but continue processing
					fmt.Printf("Warning: failed to parse ContractDeployed event: %v\n", err)
					continue
				}
				events = append(events, *event)
			}
		}
	}
	
	return events, nil
}

// parseContractDeployedEvent parses a ContractDeployed event from a log
func parseContractDeployedEvent(log Log, txHash common.Hash, blockNumber uint64) (*DeploymentEvent, error) {
	if len(log.Topics) < 3 {
		return nil, fmt.Errorf("invalid number of topics for ContractDeployed event")
	}
	
	// Topics: [eventSig, operationId (indexed), sender (indexed)]
	operationID := log.Topics[1]
	sender := common.HexToAddress(log.Topics[2].Hex())
	
	// Decode non-indexed parameters from data
	// Parameters: address deployedAddress, bytes32 salt, bytes32 initCodeHash, string createStrategy
	data, err := hex.DecodeString(strings.TrimPrefix(log.Data, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode event data: %w", err)
	}
	
	// Create ABI for decoding
	addressType, _ := abi.NewType("address", "", nil)
	bytes32Type, _ := abi.NewType("bytes32", "", nil)
	stringType, _ := abi.NewType("string", "", nil)
	
	args := abi.Arguments{
		{Type: addressType, Name: "deployedAddress"},
		{Type: bytes32Type, Name: "salt"},
		{Type: bytes32Type, Name: "initCodeHash"},
		{Type: stringType, Name: "createStrategy"},
	}
	
	// Unpack the data
	values, err := args.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack event data: %w", err)
	}
	
	if len(values) != 4 {
		return nil, fmt.Errorf("unexpected number of values unpacked")
	}
	
	deployedAddress, ok := values[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("failed to cast deployed address")
	}
	
	salt, ok := values[1].([32]byte)
	if !ok {
		return nil, fmt.Errorf("failed to cast salt")
	}
	
	initCodeHash, ok := values[2].([32]byte)
	if !ok {
		return nil, fmt.Errorf("failed to cast init code hash")
	}
	
	createStrategy, ok := values[3].(string)
	if !ok {
		return nil, fmt.Errorf("failed to cast create strategy")
	}
	
	return &DeploymentEvent{
		OperationID:     operationID,
		Sender:          sender,
		DeployedAddress: deployedAddress,
		Salt:            common.BytesToHash(salt[:]),
		InitCodeHash:    common.BytesToHash(initCodeHash[:]),
		CreateStrategy:  createStrategy,
		TransactionHash: txHash,
		BlockNumber:     blockNumber,
	}, nil
}

// RecordDeploymentEvents updates the registry with deployment events
func RecordDeploymentEvents(events []DeploymentEvent, network string, profile string) error {
	// Load registry
	mgr, err := registry.NewManager("")
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}
	
	// Convert network name to chain ID if needed
	chainID := network
	// TODO: Implement network to chain ID mapping
	
	for _, event := range events {
		// Create deployment entry
		entry := &registry.DeploymentEntry{
			Address:         event.DeployedAddress.Hex(),
			DeploymentType:  types.SingletonDeployment, // Default, could be determined from event
			ImplementedBy:   nil,
			Salt:            event.Salt.Hex(),
			InitCodeHash:    event.InitCodeHash.Hex(),
			CreateStrategy:  event.CreateStrategy,
			DeploymentInfo: &registry.DeploymentInfo{
				TransactionHash: event.TransactionHash.Hex(),
				BlockNumber:     big.NewInt(int64(event.BlockNumber)),
				Timestamp:       nil, // Would need to query block info
				DeployerAddress: event.Sender.Hex(),
			},
			VerificationInfo: nil,
			Metadata: &registry.ContractMetadata{
				CompilerVersion: "",
				SourceHash:      "",
				Args:            "",
			},
		}
		
		// Generate FQID and SID
		// For now, use a generic name based on address
		contractName := fmt.Sprintf("Contract_%s", event.DeployedAddress.Hex()[2:10])
		namespace := profile
		
		fqid := fmt.Sprintf("%s/%s/%s", chainID, namespace, contractName)
		sid := contractName
		
		// Update registry
		err := mgr.UpdateDeployment(chainID, fqid, sid, entry)
		if err != nil {
			return fmt.Errorf("failed to update registry for %s: %w", contractName, err)
		}
	}
	
	// Save registry
	return mgr.Save()
}