package safe

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
)

// ClientAdapter wraps the internal Safe client to implement SafeClient
type SafeClient struct {
	serviceURL string
	httpClient *http.Client
}

// NewClientAdapter creates a new adapter wrapping the internal Safe client
func NewSafeClient(chainId uint64) (*SafeClient, error) {
	serviceURL, ok := TransactionServiceURLs[chainId]
	if !ok {
		return nil, fmt.Errorf("unsupported chain ID: %d", chainId)
	}

	return &SafeClient{
		serviceURL: serviceURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// GetTransactionExecutionInfo checks if a Safe transaction is executed
func (c *SafeClient) GetTransactionExecutionInfo(ctx context.Context, safeTxHash string) (*models.SafeExecutionInfo, error) {
	hash := common.HexToHash(safeTxHash)
	// Check if transaction is executed
	isExecuted, ethTxHash, err := c.IsTransactionExecuted(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to check transaction execution: %w", err)
	}

	info := &models.SafeExecutionInfo{
		IsExecuted: isExecuted,
	}

	if isExecuted && ethTxHash != nil {
		info.TxHash = ethTxHash.Hex()
	}

	// Get transaction details for confirmation info
	tx, err := c.GetTransaction(hash)
	if err == nil && tx != nil {
		info.Confirmations = len(tx.Confirmations)
		info.ConfirmationsRequired = tx.ConfirmationsRequired

		// Convert confirmations
		for _, conf := range tx.Confirmations {
			info.ConfirmationDetails = append(info.ConfirmationDetails, models.Confirmation{
				Signer:    conf.Owner,
				Signature: conf.Signature,
				// Note: Safe API doesn't provide confirmation time
			})
		}
	}

	return info, nil
}
