package script

import (
	"github.com/ethereum/go-ethereum/crypto"
)

// Event signatures (keccak256 of event signature) for Treb events
// NOTE: These are now primarily used for proxy events not covered by our ABI.
// The main Treb events are handled by generated ABI bindings in event_parser.go
var (
	// From Deployer.sol
	// DeployingContract(string what, string label, bytes32 initCodeHash)
	DeployingContractTopic = crypto.Keccak256Hash([]byte("DeployingContract(string,string,bytes32)"))

	// ContractDeployed(address indexed deployer, address indexed location, bytes32 indexed transactionId, (string,string,string,bytes32,bytes32,bytes32,bytes,string) deployment)
	ContractDeployedTopic = crypto.Keccak256Hash([]byte("ContractDeployed(address,address,bytes32,(string,string,string,bytes32,bytes32,bytes32,bytes,string))"))

	// From Senders.sol - Transaction events
	// TransactionSimulated(bytes32 indexed transactionId, address indexed sender, address indexed to, uint256 value, bytes data, string label, bytes returnData)
	TransactionSimulatedTopic = crypto.Keccak256Hash([]byte("TransactionSimulated(bytes32,address,address,uint256,bytes,string,bytes)"))

	// TransactionFailed(bytes32 indexed transactionId, address indexed sender, address indexed to, uint256 value, bytes data, string label)
	TransactionFailedTopic = crypto.Keccak256Hash([]byte("TransactionFailed(bytes32,address,address,uint256,bytes,string)"))

	// TransactionBroadcast(bytes32 indexed transactionId, address indexed sender, address indexed to, uint256 value, bytes data, string label, bytes returnData)
	TransactionBroadcastTopic = crypto.Keccak256Hash([]byte("TransactionBroadcast(bytes32,address,address,uint256,bytes,string,bytes)"))

	// BroadcastStarted() - Marks the start of the broadcast phase
	BroadcastStartedTopic = crypto.Keccak256Hash([]byte("BroadcastStarted()"))

	// From SafeSender.sol
	// SafeTransactionQueued(bytes32 indexed safeTxHash, address indexed safe, address indexed proposer, ((string,address,bytes,uint256),bytes32,bytes32,uint8,bytes,bytes)[] transactions)
	SafeTransactionQueuedTopic = crypto.Keccak256Hash([]byte("SafeTransactionQueued(bytes32,address,address,((string,address,bytes,uint256),bytes32,bytes32,uint8,bytes,bytes)[])"))
)

// NOTE: Type re-exports have been removed as part of the ABI bindings migration.
// All consumers should now import event types directly from the events package:
//   import "github.com/trebuchet-org/treb-cli/cli/pkg/events"
//
// The migration to auto-generated ABI bindings provides:
// - Type safety from generated code
// - Automatic updates when contracts change
// - Elimination of manual parsing code
// - Single source of truth for event structures
