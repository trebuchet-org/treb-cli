package types

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/cli/pkg/broadcast"
	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/cli/pkg/network"
)

type Status string

const (
	StatusExecuted Status = "EXECUTED"
	StatusQueued   Status = "PENDING_SAFE"
	StatusUnknown  Status = "UNKNOWN"
)

func ParseStatus(status string) (Status, error) {
	switch status {
	case "EXECUTED":
		return StatusExecuted, nil
	case "PENDING_SAFE":
		return StatusQueued, nil
	default:
		return StatusUnknown, fmt.Errorf("unknown status: %s", status)
	}
}

type DeploymentResult struct {
	FQID            string         `json:"fqid"` // Fully qualified identifier
	ShortID         string         `json:"sid"`  // Short identifier
	Address         common.Address `json:"address"`
	TxHash          common.Hash    `json:"transaction_hash"`
	BlockNumber     uint64         `json:"block_number"`
	BroadcastFile   string         `json:"broadcast_file"`
	Salt            string         `json:"salt"`           // Keep as bytes for internal use
	InitCodeHash    string         `json:"init_code_hash"` // Keep as bytes for internal use
	AlreadyDeployed bool           `json:"already_deployed"`

	// New deployment type information
	DeploymentType string               `json:"deployment_type"` // "implementation" or "proxy"
	TargetContract string               `json:"target_contract,omitempty"`
	Label          string               `json:"label,omitempty"` // For implementation deployments
	Tags           []string             `json:"tags,omitempty"`
	Env            string               `json:"env,omitempty"`
	NetworkInfo    *network.NetworkInfo `json:"network_info,omitempty"`
	Status         Status               `json:"status"`

	// Safe deployment information
	SafeTxHash  common.Hash    `json:"safe_tx_hash,omitempty"`
	SafeAddress common.Address `json:"safe_address,omitempty"`

	// Constructor arguments for verification
	ConstructorArgs string `json:"constructor_args,omitempty"`

	// Metadata
	Metadata     *ContractMetadata       `json:"metadata,omitempty"`
	ContractInfo *contracts.ContractInfo `json:"contract_info,omitempty"`

	// Broadcast file (linked directly, not parsed into other fields)
	BroadcastData *broadcast.BroadcastFile `json:"broadcast_data,omitempty"` // Will hold *broadcast.BroadcastFile
}

type PredictResult struct {
	Address      common.Address `json:"address"`
	Salt         string         `json:"salt"`
	InitCodeHash string         `json:"init_code_hash"`
}

type DeploymentEntry struct {
	FQID            string         `json:"fqid"` // Fully qualified identifier
	ShortID         string         `json:"sid"`  // Short identifier
	Address         common.Address `json:"address"`
	ContractName    string         `json:"contract_name"`
	Environment     string         `json:"environment"`
	Type            string         `json:"type"`                       // implementation/proxy
	Salt            string         `json:"salt"`                       // hex string
	InitCodeHash    string         `json:"init_code_hash"`             // hex string
	ConstructorArgs string         `json:"constructor_args,omitempty"` // Raw hex-encoded constructor args

	// Label for all deployments (optional for implementations, required for proxies)
	Label string `json:"label,omitempty"`

	// Proxy-specific fields
	TargetContract string `json:"target_contract,omitempty"` // For proxy deployments

	// Version tags (metadata only, not part of salt)
	Tags []string `json:"tags,omitempty"`

	Verification Verification     `json:"verification"`
	Deployment   DeploymentInfo   `json:"deployment"`
	Metadata     ContractMetadata `json:"metadata"`
}

type Verification struct {
	Status      string                    `json:"status"` // verified/pending/failed/partial
	ExplorerUrl string                    `json:"explorer_url,omitempty"`
	Reason      string                    `json:"reason,omitempty"`
	Verifiers   map[string]VerifierStatus `json:"verifiers,omitempty"` // etherscan, sourcify status
}

type VerifierStatus struct {
	Status string `json:"status"` // verified/pending/failed
	URL    string `json:"url,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type DeploymentInfo struct {
	TxHash        *common.Hash `json:"tx_hash,omitempty"`
	SafeTxHash    *common.Hash `json:"safe_tx_hash,omitempty"`
	BlockNumber   uint64       `json:"block_number,omitempty"`
	BroadcastFile string       `json:"broadcast_file"`
	Timestamp     time.Time    `json:"timestamp"`
	Status        Status       `json:"status"` // "deployed", "pending_safe"
	SafeAddress   string       `json:"safe_address,omitempty"`
	SafeNonce     uint64       `json:"safe_nonce,omitempty"`
	Deployer      string       `json:"deployer,omitempty"` // Address that deployed the contract
}

type ContractMetadata struct {
	SourceCommit string                 `json:"source_commit"`
	Compiler     string                 `json:"compiler"`
	SourceHash   string                 `json:"source_hash"`
	ContractPath string                 `json:"contract_path"`   // Full path like ./src/Contract.sol:Contract
	ScriptPath   string                 `json:"script_path"`     // Deployment script path like script/deploy/DeployCounter.s.sol
	Extra        map[string]interface{} `json:"extra,omitempty"` // Additional metadata (e.g., proxy type, implementation address)
}

// GetDisplayName returns a human-friendly name for the deployment
func (d *DeploymentEntry) GetDisplayName() string {
	if d.Type == "proxy" {
		// For proxies: targetContractProxy or targetContractProxy:label
		baseName := d.TargetContract + "Proxy"
		if d.Label != "" {
			return baseName + ":" + d.Label
		}
		return baseName
	}

	// For implementations: contractName or contractName:label
	if d.Label != "" {
		return d.ContractName + ":" + d.Label
	}
	return d.ContractName
}

func (d *DeploymentEntry) GetIdentifier() string {
	switch d.Type {
	case "proxy":
		return d.TargetContract + "Proxy"
	case "library":
		return d.ContractName
	}

	if d.Label != "" {
		return d.ContractName + ":" + d.Label
	}
	return d.ContractName
}

func (d *DeploymentEntry) GetFullIdentifier(networkName string) string {
	return d.Environment + "/" + networkName + "/" + d.GetIdentifier()
}
