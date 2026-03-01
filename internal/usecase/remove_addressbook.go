package usecase

import (
	"context"
	"fmt"

	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

// RemoveAddressbookParams contains parameters for removing an addressbook entry.
type RemoveAddressbookParams struct {
	Name string
}

// RemoveAddressbookResult contains the result of removing an addressbook entry.
type RemoveAddressbookResult struct {
	Name    string
	ChainID uint64
}

// RemoveAddressbook is the use case for removing an addressbook entry.
type RemoveAddressbook struct {
	config *config.RuntimeConfig
	repo   AddressbookRepository
}

// NewRemoveAddressbook creates a new RemoveAddressbook use case.
func NewRemoveAddressbook(cfg *config.RuntimeConfig, repo AddressbookRepository) *RemoveAddressbook {
	return &RemoveAddressbook{
		config: cfg,
		repo:   repo,
	}
}

// Run executes the remove addressbook use case.
func (uc *RemoveAddressbook) Run(ctx context.Context, params RemoveAddressbookParams) (*RemoveAddressbookResult, error) {
	if uc.config.Network == nil {
		return nil, fmt.Errorf("no network configured; set one with --network or `treb config set network <name>`")
	}

	chainKey := fmt.Sprintf("%d", uc.config.Network.ChainID)

	ab, err := uc.repo.Load(ctx)
	if err != nil {
		return nil, err
	}

	chain, ok := ab[chainKey]
	if !ok || chain[params.Name] == "" {
		return nil, fmt.Errorf("addressbook entry %q not found on chain %d", params.Name, uc.config.Network.ChainID)
	}

	delete(chain, params.Name)

	// Clean up empty chain map
	if len(chain) == 0 {
		delete(ab, chainKey)
	}

	if err := uc.repo.Save(ctx, ab); err != nil {
		return nil, err
	}

	return &RemoveAddressbookResult{
		Name:    params.Name,
		ChainID: uc.config.Network.ChainID,
	}, nil
}
