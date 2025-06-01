package safe

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		chainID uint64
		wantErr bool
	}{
		{
			name:    "Mainnet",
			chainID: 1,
			wantErr: false,
		},
		{
			name:    "Sepolia",
			chainID: 11155111,
			wantErr: false,
		},
		{
			name:    "Unsupported chain",
			chainID: 999999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(tt.chainID)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSafeTransactionServiceURLs(t *testing.T) {
	// Verify key chains are supported
	supportedChains := []uint64{1, 5, 10, 100, 137, 42161, 11155111, 8453}

	for _, chainID := range supportedChains {
		if _, ok := TransactionServiceURLs[chainID]; !ok {
			t.Errorf("Chain %d should be supported but is not in TransactionServiceURLs", chainID)
		}
	}
}

// Example of how to check a transaction (commented out to avoid external API calls in tests)
func ExampleClient_GetTransaction() {
	// Create client for Sepolia
	client, _ := NewClient(11155111)

	// Check a specific Safe transaction
	safeTxHash := common.HexToHash("0xf8bc36421955315c1635bfb037853fb24aa5a7d0d720f57428f82902687662e5")

	tx, err := client.GetTransaction(safeTxHash)
	if err != nil {
		// Handle error
		return
	}

	// Check if executed
	if tx.IsExecuted {
		// Transaction has been executed
		_ = tx.TransactionHash // This is the Ethereum transaction hash
	}
}
