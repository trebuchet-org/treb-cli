package types

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type DeploymentResult struct {
	Address         common.Address    `json:"address"`
	TxHash          common.Hash       `json:"transaction_hash"`
	BlockNumber     uint64            `json:"block_number"`
	BroadcastFile   string            `json:"broadcast_file"`
	Salt            [32]byte          `json:"salt"`            // Keep as bytes for internal use
	InitCodeHash    [32]byte          `json:"init_code_hash"`  // Keep as bytes for internal use
	AlreadyDeployed bool              `json:"already_deployed"`
	
	// New deployment type information
	Type            string            `json:"type"`            // "implementation" or "proxy" 
	DeploymentType  string            `json:"deployment_type"`  // "implementation" or "proxy"
	TargetContract  string            `json:"target_contract,omitempty"`
	Label           string            `json:"label,omitempty"`  // For implementation deployments
	Tags            []string          `json:"tags,omitempty"`
	Env             string            `json:"env,omitempty"`
	
	// Safe deployment information
	SafeTxHash      common.Hash       `json:"safe_tx_hash,omitempty"`
	SafeAddress     common.Address    `json:"safe_address,omitempty"`
	
	// Metadata
	Metadata        *ContractMetadata `json:"metadata,omitempty"`
}

type PredictResult struct {
	Address      common.Address `json:"address"`
	Salt         [32]byte       `json:"salt"`
	InitCodeHash [32]byte       `json:"init_code_hash"`
}

type DeploymentEntry struct {
	Address       common.Address   `json:"address"`
	ContractName  string           `json:"contract_name"`
	Environment   string           `json:"environment"`
	Type          string           `json:"type"` // implementation/proxy
	Salt          string           `json:"salt"`           // hex string
	InitCodeHash  string           `json:"init_code_hash"` // hex string
	Constructor   []interface{}    `json:"constructor_args,omitempty"`
	
	// Label for all deployments (optional for implementations, required for proxies)
	Label         string           `json:"label,omitempty"`
	
	// Proxy-specific fields
	TargetContract string           `json:"target_contract,omitempty"` // For proxy deployments
	
	// Version tags (metadata only, not part of salt)
	Tags          []string         `json:"tags,omitempty"`
	
	Verification  Verification     `json:"verification"`
	Deployment    DeploymentInfo   `json:"deployment"`
	Metadata      ContractMetadata `json:"metadata"`
}

type Verification struct {
	Status      string `json:"status"`      // verified/pending/failed
	ExplorerUrl string `json:"explorer_url,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

type DeploymentInfo struct {
	TxHash        *common.Hash `json:"tx_hash,omitempty"`
	SafeTxHash    *common.Hash `json:"safe_tx_hash,omitempty"`
	BlockNumber   uint64       `json:"block_number,omitempty"`
	BroadcastFile string       `json:"broadcast_file"`
	Timestamp     time.Time    `json:"timestamp"`
	Status        string       `json:"status"` // "deployed", "pending_safe"
	SafeAddress   string       `json:"safe_address,omitempty"`
	SafeNonce     uint64       `json:"safe_nonce,omitempty"`
	Deployer      string       `json:"deployer,omitempty"` // Address that deployed the contract
}

type ContractMetadata struct {
	SourceCommit    string                 `json:"source_commit"`
	Compiler        string                 `json:"compiler"`
	SourceHash      string                 `json:"source_hash"`
	ContractPath    string                 `json:"contract_path"`  // Full path like ./src/Contract.sol:Contract
	Extra           map[string]interface{} `json:"extra,omitempty"` // Additional metadata (e.g., proxy type, implementation address)
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