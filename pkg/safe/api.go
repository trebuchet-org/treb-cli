package safe

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// TransactionServiceURLs contains the Safe Transaction Service URLs for different networks
var TransactionServiceURLs = map[uint64]string{
	1:        "https://safe-transaction-mainnet.safe.global",
	5:        "https://safe-transaction-goerli.safe.global",
	10:       "https://safe-transaction-optimism.safe.global",
	100:      "https://safe-transaction-gnosis-chain.safe.global",
	137:      "https://safe-transaction-polygon.safe.global",
	42161:    "https://safe-transaction-arbitrum.safe.global",
	11155111: "https://safe-transaction-sepolia.safe.global",
	8453:     "https://safe-transaction-base.safe.global",
	56:       "https://safe-transaction-bsc.safe.global",
	43114:    "https://safe-transaction-avalanche.safe.global",
	324:      "https://safe-transaction-zksync.safe.global",
	42220:    "https://safe-transaction-celo.safe.global",
	11142220: "https://safe-transaction-celo-sepolia.safe.global", // Celo Sepolia testnet
}

// MultisigTransaction represents a Safe multisig transaction
type MultisigTransaction struct {
	Safe                  string         `json:"safe"`
	To                    string         `json:"to"`
	Value                 string         `json:"value"`
	Data                  string         `json:"data"`
	Operation             int            `json:"operation"`
	SafeTxGas             int            `json:"safeTxGas"`
	BaseGas               int            `json:"baseGas"`
	GasPrice              string         `json:"gasPrice"`
	GasToken              string         `json:"gasToken"`
	RefundReceiver        string         `json:"refundReceiver"`
	Nonce                 int            `json:"nonce"`
	ExecutionDate         *time.Time     `json:"executionDate"`
	SubmissionDate        time.Time      `json:"submissionDate"`
	Modified              time.Time      `json:"modified"`
	BlockNumber           *int64         `json:"blockNumber"`
	TransactionHash       *string        `json:"transactionHash"`
	SafeTxHash            string         `json:"safeTxHash"`
	Executor              *string        `json:"executor"`
	IsExecuted            bool           `json:"isExecuted"`
	IsSuccessful          *bool          `json:"isSuccessful"`
	EthGasPrice           *string        `json:"ethGasPrice"`
	MaxFeePerGas          *string        `json:"maxFeePerGas"`
	MaxPriorityFeePerGas  *string        `json:"maxPriorityFeePerGas"`
	GasUsed               *int64         `json:"gasUsed"`
	Fee                   *string        `json:"fee"`
	Origin                string         `json:"origin"`
	DataDecoded           interface{}    `json:"dataDecoded"`
	ConfirmationsRequired int            `json:"confirmationsRequired"`
	Confirmations         []Confirmation `json:"confirmations"`
	Trusted               bool           `json:"trusted"`
	Signatures            *string        `json:"signatures"`
}

// Confirmation represents a confirmation on a Safe transaction
type Confirmation struct {
	Owner           string    `json:"owner"`
	SubmissionDate  time.Time `json:"submissionDate"`
	TransactionHash *string   `json:"transactionHash"`
	Signature       string    `json:"signature"`
	SignatureType   string    `json:"signatureType"`
	Origin          string    `json:"origin,omitempty"`
}

// GetTransaction retrieves a Safe transaction by its hash
func (c *SafeClient) GetTransaction(safeTxHash common.Hash) (*MultisigTransaction, error) {
	url := fmt.Sprintf("%s/api/v1/multisig-transactions/%s/", c.serviceURL, safeTxHash.Hex())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var tx MultisigTransaction
	if err := json.NewDecoder(resp.Body).Decode(&tx); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tx, nil
}

// IsTransactionExecuted checks if a Safe transaction has been executed
func (c *SafeClient) IsTransactionExecuted(safeTxHash common.Hash) (bool, *common.Hash, error) {
	tx, err := c.GetTransaction(safeTxHash)
	if err != nil {
		return false, nil, err
	}

	if tx.IsExecuted && tx.TransactionHash != nil {
		ethTxHash := common.HexToHash(*tx.TransactionHash)
		return true, &ethTxHash, nil
	}

	return false, nil, nil
}

// GetPendingTransactions retrieves pending transactions for a Safe
func (c *SafeClient) GetPendingTransactions(safeAddress common.Address) ([]*MultisigTransaction, error) {
	url := fmt.Sprintf("%s/api/v1/safes/%s/multisig-transactions/?executed=false&ordering=-nonce",
		c.serviceURL, safeAddress.Hex())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Results []*MultisigTransaction `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Results, nil
}
