package helpers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type TrebParser struct {
	Deployments      map[string]Deployment
	Transactions     map[string]Transaction
	SafeTransactions map[string]SafeTransaction
}

type Deployment struct {
	ID            string `json:"id"`        // e.g., "production/1/Counter:v1"
	Namespace     string `json:"namespace"` // e.g., "production", "staging", "test"
	ChainID       uint64 `json:"chainId"`
	ContractName  string `json:"contractName"`  // e.g., "Counter"
	Label         string `json:"label"`         // e.g., "v1", "main", "usdc"
	Address       string `json:"address"`       // Contract address
	Type          string `json:"type"`          // SINGLETON, PROXY, LIBRARY
	TransactionID string `json:"transactionId"` // Reference to transaction record
}

type Transaction struct {
	// Identification
	ID      string `json:"id"` // e.g., "tx-0x1234abcd..."
	ChainID uint64 `json:"chainId"`
	Hash    string `json:"hash"` // Transaction hash

	// Transaction details
	Status      string `json:"status"` // PENDING, EXECUTED, FAILED
	BlockNumber uint64 `json:"blockNumber,omitempty"`
	Sender      string `json:"sender"` // From address
	Nonce       uint64 `json:"nonce"`

	// Deployment references
	Deployments []string `json:"deployments"` // Deployment IDs created in this tx
}

type SafeTransaction struct {
	// Identification
	SafeTxHash  string `json:"safeTxHash"`
	SafeAddress string `json:"safeAddress"`
	ChainID     uint64 `json:"chainId"`
	Status      string `json:"status"`
	Nonce       uint64 `json:"nonce"`

	// References to executed transactions
	TransactionIDs []string `json:"transactionIds"`
}

func NewTrebParser(ctx *TestContext) (*TrebParser, error) {
	parser := &TrebParser{}

	var err error
	if parser.Deployments, err = parseTrebFile[Deployment](ctx.WorkDir, "deployments.json"); err != nil {
		return nil, err
	}
	if parser.Transactions, err = parseTrebFile[Transaction](ctx.WorkDir, "transactions.json"); err != nil {
		return nil, err
	}
	if parser.SafeTransactions, err = parseTrebFile[SafeTransaction](ctx.WorkDir, "safe-txs.json"); err != nil {
		return nil, err
	}

	return parser, nil
}

func (tp *TrebParser) Deployment(keyFragment string) (*Deployment, error) {
	for k, v := range tp.Deployments {
		if strings.Contains(k, keyFragment) {
			return &v, nil
		}
	}

	return nil, fmt.Errorf("%s not found in deployments", keyFragment)
}

func parseTrebFile[T Deployment | Transaction | SafeTransaction](workdir, file string) (map[string]T, error) {
	content, err := os.ReadFile(filepath.Join(workdir, ".treb", file))
	if err != nil {
		return nil, err
	}

	items := make(map[string]T)
	if err := json.Unmarshal(content, &items); err != nil {
		return nil, err
	}
	return items, nil
}
