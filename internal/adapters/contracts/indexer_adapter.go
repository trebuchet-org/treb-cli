package contracts

import (
	"context"
	"fmt"

	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// IndexerAdapter adapts the internal indexer to the ContractIndexer interface
type IndexerAdapter struct {
	indexer *InternalIndexer
}

// NewIndexerAdapter creates a new contract indexer adapter
func NewIndexerAdapter(cfg *config.RuntimeConfig) (*IndexerAdapter, error) {
	indexer := NewInternalIndexer(cfg.ProjectRoot)
	
	// Index contracts
	if err := indexer.Index(); err != nil {
		return nil, fmt.Errorf("failed to index contracts: %w", err)
	}

	return &IndexerAdapter{
		indexer: indexer,
	}, nil
}

// GetContract retrieves a contract by key
func (a *IndexerAdapter) GetContract(ctx context.Context, key string) (*domain.ContractInfo, error) {
	return a.indexer.GetContract(key)
}

// SearchContracts searches for contracts matching a pattern
func (a *IndexerAdapter) SearchContracts(ctx context.Context, pattern string) []*domain.ContractInfo {
	return a.indexer.SearchContracts(pattern)
}

// GetContractByArtifact retrieves a contract by its artifact path
func (a *IndexerAdapter) GetContractByArtifact(ctx context.Context, artifact string) *domain.ContractInfo {
	return a.indexer.GetContractByArtifact(artifact)
}

// RefreshIndex refreshes the contract index
func (a *IndexerAdapter) RefreshIndex(ctx context.Context) error {
	return a.indexer.Index()
}