package fs

import (
	"context"
	"fmt"

	"github.com/trebuchet-org/treb-cli/cli/pkg/contracts"
	"github.com/trebuchet-org/treb-cli/internal/config"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// ContractIndexerAdapter wraps the existing contracts.Indexer to implement ContractIndexer
type ContractIndexerAdapter struct {
	indexer *contracts.Indexer
}

// NewContractIndexerAdapter creates a new adapter wrapping the existing contract indexer
func NewContractIndexerAdapter(cfg *config.RuntimeConfig) (*ContractIndexerAdapter, error) {
	indexer := contracts.NewIndexer(cfg.ProjectRoot)
	
	// Build the initial index
	if err := indexer.Index(); err != nil {
		return nil, fmt.Errorf("failed to build contract index: %w", err)
	}
	
	return &ContractIndexerAdapter{indexer: indexer}, nil
}

// GetContract retrieves a contract by key
func (c *ContractIndexerAdapter) GetContract(ctx context.Context, key string) (*domain.ContractInfo, error) {
	contract, err := c.indexer.GetContract(key)
	if err != nil {
		return nil, err
	}
	if contract == nil {
		return nil, domain.ErrContractNotFound
	}
	return convertToDomainContractInfo(contract), nil
}

// SearchContracts searches for contracts matching a pattern
func (c *ContractIndexerAdapter) SearchContracts(ctx context.Context, pattern string) []*domain.ContractInfo {
	contracts := c.indexer.SearchContracts(pattern)
	
	result := make([]*domain.ContractInfo, len(contracts))
	for i, contract := range contracts {
		result[i] = convertToDomainContractInfo(contract)
	}
	
	return result
}

// GetContractByArtifact retrieves a contract by its artifact path
func (c *ContractIndexerAdapter) GetContractByArtifact(ctx context.Context, artifact string) *domain.ContractInfo {
	contract := c.indexer.GetContractByArtifact(artifact)
	if contract == nil {
		return nil
	}
	return convertToDomainContractInfo(contract)
}

// RefreshIndex rebuilds the contract index
func (c *ContractIndexerAdapter) RefreshIndex(ctx context.Context) error {
	return c.indexer.Index()
}

// convertToDomainContractInfo converts from pkg/contracts to domain types
func convertToDomainContractInfo(contract *contracts.ContractInfo) *domain.ContractInfo {
	if contract == nil {
		return nil
	}
	
	return &domain.ContractInfo{
		Name:         contract.Name,
		Path:         contract.Path,
		ArtifactPath: contract.ArtifactPath,
		Version:      contract.Version,
		IsLibrary:    contract.IsLibrary,
		IsInterface:  contract.IsInterface,
		IsAbstract:   contract.IsAbstract,
	}
}

// Ensure the adapter implements the interface
var _ usecase.ContractIndexer = (*ContractIndexerAdapter)(nil)