package safe

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ClientAdapter wraps the internal Safe client to implement SafeClient
type ClientAdapter struct {
	client  *Client
	chainID uint64
}

// NewClientAdapter creates a new adapter wrapping the internal Safe client
func NewClientAdapter(cfg *config.RuntimeConfig) (*ClientAdapter, error) {
	// Create with a default chain ID (will be set later)
	client, err := NewClient(1)
	if err != nil {
		return nil, fmt.Errorf("failed to create Safe client: %w", err)
	}

	return &ClientAdapter{
		client:  client,
		chainID: 1,
	}, nil
}

// SetChain configures the client for a specific chain
func (c *ClientAdapter) SetChain(ctx context.Context, chainID uint64) error {
	// Create a new client for the specified chain
	client, err := NewClient(chainID)
	if err != nil {
		return fmt.Errorf("failed to create Safe client for chain %d: %w", chainID, err)
	}

	c.client = client
	c.chainID = chainID
	return nil
}

// GetTransactionExecutionInfo checks if a Safe transaction is executed
func (c *ClientAdapter) GetTransactionExecutionInfo(ctx context.Context, safeTxHash string) (*domain.SafeExecutionInfo, error) {
	// Convert hex string to hash
	hash := common.HexToHash(safeTxHash)

	// Check if transaction is executed
	isExecuted, ethTxHash, err := c.client.IsTransactionExecuted(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to check transaction execution: %w", err)
	}

	info := &domain.SafeExecutionInfo{
		IsExecuted: isExecuted,
	}

	if isExecuted && ethTxHash != nil {
		info.TxHash = ethTxHash.Hex()
	}

	// Get transaction details for confirmation info
	tx, err := c.client.GetTransaction(hash)
	if err == nil && tx != nil {
		info.Confirmations = len(tx.Confirmations)
		info.ConfirmationsRequired = tx.ConfirmationsRequired

		// Convert confirmations
		for _, conf := range tx.Confirmations {
			info.ConfirmationDetails = append(info.ConfirmationDetails, domain.Confirmation{
				Signer:    conf.Owner,
				Signature: conf.Signature,
				// Note: Safe API doesn't provide confirmation time
			})
		}
	}

	return info, nil
}

// GetTransactionDetails retrieves full Safe transaction details
func (c *ClientAdapter) GetTransactionDetails(ctx context.Context, safeTxHash string) (*domain.SafeTransaction, error) {
	// Convert hex string to hash
	hash := common.HexToHash(safeTxHash)

	// Get transaction from Safe API
	tx, err := c.client.GetTransaction(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction details: %w", err)
	}

	// Convert to domain type
	safeTx := &domain.SafeTransaction{
		SafeTxHash:            safeTxHash,
		ChainID:               c.chainID,
		SafeAddress:           tx.Safe,
		Nonce:                 uint64(tx.Nonce),
		To:                    tx.To,
		Value:                 tx.Value,
		Data:                  tx.Data,
		Operation:             tx.Operation,
		ConfirmationsRequired: tx.ConfirmationsRequired,
	}

	// Add confirmations
	for _, conf := range tx.Confirmations {
		safeTx.Confirmations = append(safeTx.Confirmations, domain.Confirmation{
			Signer:    conf.Owner,
			Signature: conf.Signature,
		})
	}

	safeTx.ConfirmationCount = len(safeTx.Confirmations)

	// Check execution status
	if tx.IsExecuted {
		safeTx.Status = domain.SafeTxStatusExecuted
		if tx.TransactionHash != nil && *tx.TransactionHash != "" {
			safeTx.ExecutionTxHash = *tx.TransactionHash
		}
	} else {
		safeTx.Status = domain.SafeTxStatusQueued
	}

	return safeTx, nil
}

// Ensure the adapter implements the interface
var _ usecase.SafeClient = (*ClientAdapter)(nil)

