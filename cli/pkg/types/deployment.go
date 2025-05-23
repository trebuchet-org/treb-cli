package types

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type DeploymentResult struct {
	Address       common.Address `json:"address"`
	TxHash        common.Hash    `json:"transaction_hash"`
	BlockNumber   uint64         `json:"block_number"`
	BroadcastFile string         `json:"broadcast_file"`
	Salt          [32]byte       `json:"salt"`
	InitCodeHash  [32]byte       `json:"init_code_hash"`
}

type PredictResult struct {
	Address      common.Address `json:"address"`
	Salt         [32]byte       `json:"salt"`
	InitCodeHash [32]byte       `json:"init_code_hash"`
}

type DeploymentEntry struct {
	Address       common.Address   `json:"address"`
	Type          string           `json:"type"` // implementation/proxy
	Salt          [32]byte         `json:"salt"`
	InitCodeHash  [32]byte         `json:"init_code_hash"`
	Constructor   []interface{}    `json:"constructor_args,omitempty"`
	
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
}

type ContractMetadata struct {
	ContractVersion string `json:"contract_version"`
	SourceCommit    string `json:"source_commit"`
	Compiler        string `json:"compiler"`
	SourceHash      string `json:"source_hash"`
}