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
	44787:    "https://safe-transaction-alfajores.safe.global", // Celo Alfajores testnet
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
	GasUsed               *int           `json:"gasUsed"`
	Fee                   *string        `json:"fee"`
	Origin                *string        `json:"origin"`
	DataDecoded           interface{}    `json:"dataDecoded"`
	ConfirmationsRequired int            `json:"confirmationsRequired"`
	Confirmations         []Confirmation `json:"confirmations"`
	Trusted               bool           `json:"trusted"`
	Signatures            *string        `json:"signatures"`
}

// Confirmation represents a transaction confirmation
type Confirmation struct {
	Owner           string    `json:"owner"`
	SubmissionDate  time.Time `json:"submissionDate"`
	TransactionHash *string   `json:"transactionHash"`
	Signature       string    `json:"signature"`
	SignatureType   string    `json:"signatureType"`
}

// Client represents a Safe Transaction Service client
type Client struct {
	baseURL    string
	httpClient *http.Client
	debug      bool
}

// NewClient creates a new Safe Transaction Service client
func NewClient(chainID uint64) (*Client, error) {
	baseURL, ok := TransactionServiceURLs[chainID]
	if !ok {
		return nil, fmt.Errorf("unsupported chain ID: %d", chainID)
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		debug: false,
	}, nil
}

// SetDebug enables or disables debug output
func (c *Client) SetDebug(debug bool) {
	c.debug = debug
}

// GetTransaction retrieves a transaction by its Safe transaction hash
func (c *Client) GetTransaction(safeTxHash common.Hash) (*MultisigTransaction, error) {
	url := fmt.Sprintf("%s/api/v1/multisig-transactions/%s/", c.baseURL, safeTxHash.Hex())

	// Debug logging
	if c.debug {
		fmt.Printf("    [DEBUG] GET %s\n", url)
	}

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	defer resp.Body.Close()

	// Read body for debugging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if c.debug {
		fmt.Printf("    [DEBUG] Response status: %d\n", resp.StatusCode)
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("    [DEBUG] Response body: %s\n", string(body))
		}
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("transaction not found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var tx MultisigTransaction
	if err := json.Unmarshal(body, &tx); err != nil {
		if c.debug {
			fmt.Printf("    [DEBUG] Failed to parse JSON: %s\n", string(body))
		}
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tx, nil
}

// GetPendingTransactions retrieves all pending transactions for a Safe
func (c *Client) GetPendingTransactions(safeAddress common.Address) ([]MultisigTransaction, error) {
	url := fmt.Sprintf("%s/api/v1/safes/%s/multisig-transactions/?executed=false", c.baseURL, safeAddress.Hex())

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending transactions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Results []MultisigTransaction `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Results, nil
}

// IsTransactionExecuted checks if a transaction has been executed
func (c *Client) IsTransactionExecuted(safeTxHash common.Hash) (bool, *common.Hash, error) {
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
