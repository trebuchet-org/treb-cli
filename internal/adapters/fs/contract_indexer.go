package fs

import (
	"context"
	"fmt"

	"github.com/trebuchet-org/treb-cli/internal/adapters/contracts"
	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// ContractIndexerAdapter wraps the internal contracts.InternalIndexer to implement ContractIndexer
type ContractIndexerAdapter struct {
	indexer *contracts.InternalIndexer
}

// NewContractIndexerAdapter creates a new adapter wrapping the internal contract indexer
func NewContractIndexerAdapter(cfg *config.RuntimeConfig) (*ContractIndexerAdapter, error) {
	indexer := contracts.NewInternalIndexer(cfg.ProjectRoot)
	
	// Build the initial index
	if err := indexer.Index(); err != nil {
		return nil, fmt.Errorf("failed to build contract index: %w", err)
	}
	
	return &ContractIndexerAdapter{indexer: indexer}, nil
}

// GetContract retrieves a contract by key
func (c *ContractIndexerAdapter) GetContract(ctx context.Context, key string) (*domain.ContractInfo, error) {
	return c.indexer.GetContract(key)
}

// SearchContracts searches for contracts matching a pattern
func (c *ContractIndexerAdapter) SearchContracts(ctx context.Context, pattern string) []*domain.ContractInfo {
	return c.indexer.SearchContracts(pattern)
}

// GetContractByArtifact retrieves a contract by its artifact path
func (c *ContractIndexerAdapter) GetContractByArtifact(ctx context.Context, artifact string) *domain.ContractInfo {
	return c.indexer.GetContractByArtifact(artifact)
}

// RefreshIndex rebuilds the contract index
func (c *ContractIndexerAdapter) RefreshIndex(ctx context.Context) error {
	return c.indexer.Index()
}