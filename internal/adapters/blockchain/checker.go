package blockchain

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// CheckerAdapter implements the BlockchainChecker interface using ethclient
type CheckerAdapter struct {
	client  *ethclient.Client
	chainID uint64
}

// NewCheckerAdapter creates a new blockchain checker adapter
func NewCheckerAdapter() *CheckerAdapter {
	return &CheckerAdapter{}
}

// Connect establishes connection to the blockchain
func (c *CheckerAdapter) Connect(ctx context.Context, rpcURL string, chainID uint64) error {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RPC: %w", err)
	}
	c.client = client

	// Verify chain ID matches
	networkChainID, err := c.client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %w", err)
	}

	// If chainID was 0, use the network's chain ID
	if chainID == 0 {
		c.chainID = networkChainID.Uint64()
	} else if networkChainID.Uint64() != chainID {
		return fmt.Errorf("chain ID mismatch: expected %d, got %d", chainID, networkChainID.Uint64())
	} else {
		c.chainID = chainID
	}

	return nil
}

// CheckDeploymentExists checks if a contract exists at the given address
func (c *CheckerAdapter) CheckDeploymentExists(ctx context.Context, address string) (exists bool, reason string, err error) {
	if c.client == nil {
		return false, "", fmt.Errorf("not connected to blockchain")
	}

	addr := common.HexToAddress(address)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	code, err := c.client.CodeAt(ctx, addr, nil)
	if err != nil {
		return false, fmt.Sprintf("failed to check code: %v", err), nil
	}

	// If no code at address, contract doesn't exist
	if len(code) == 0 {
		return false, "no code at address", nil
	}

	return true, "", nil
}

// CheckTransactionExists checks if a transaction exists on-chain
func (c *CheckerAdapter) CheckTransactionExists(ctx context.Context, txHash string) (exists bool, blockNumber uint64, reason string, err error) {
	if c.client == nil {
		return false, 0, "", fmt.Errorf("not connected to blockchain")
	}

	hash := common.HexToHash(txHash)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	receipt, err := c.client.TransactionReceipt(ctx, hash)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return false, 0, "transaction not found on-chain", nil
		}
		// For other errors, return the error
		return false, 0, "", fmt.Errorf("failed to get transaction receipt: %w", err)
	}

	// Transaction exists
	if receipt.BlockNumber != nil {
		return true, receipt.BlockNumber.Uint64(), "", nil
	}

	return true, 0, "", nil
}

// CheckSafeContract checks if a Safe contract exists at the given address
func (c *CheckerAdapter) CheckSafeContract(ctx context.Context, safeAddress string) (exists bool, reason string, err error) {
	// For Safe contracts, we just check if code exists at the address
	// More sophisticated checks could verify it's actually a Safe contract
	return c.CheckDeploymentExists(ctx, safeAddress)
}

// Ensure the adapter implements the interface
var _ usecase.BlockchainChecker = (*CheckerAdapter)(nil)
