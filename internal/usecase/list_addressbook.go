package usecase

import (
	"context"
	"fmt"
	"sort"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

// ListAddressbookResult contains the result of listing addressbook entries.
type ListAddressbookResult struct {
	Entries []domain.AddressbookEntry
	ChainID uint64
}

// ListAddressbook is the use case for listing addressbook entries.
type ListAddressbook struct {
	config *config.RuntimeConfig
	repo   AddressbookRepository
}

// NewListAddressbook creates a new ListAddressbook use case.
func NewListAddressbook(cfg *config.RuntimeConfig, repo AddressbookRepository) *ListAddressbook {
	return &ListAddressbook{
		config: cfg,
		repo:   repo,
	}
}

// Run executes the list addressbook use case.
func (uc *ListAddressbook) Run(ctx context.Context) (*ListAddressbookResult, error) {
	if uc.config.Network == nil {
		return nil, fmt.Errorf("no network configured; set one with --network or `treb config set network <name>`")
	}

	chainKey := fmt.Sprintf("%d", uc.config.Network.ChainID)

	ab, err := uc.repo.Load(ctx)
	if err != nil {
		return nil, err
	}

	chain := ab[chainKey]
	entries := make([]domain.AddressbookEntry, 0, len(chain))
	for name, addr := range chain {
		entries = append(entries, domain.AddressbookEntry{
			Name:    name,
			Address: addr,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	return &ListAddressbookResult{
		Entries: entries,
		ChainID: uc.config.Network.ChainID,
	}, nil
}
