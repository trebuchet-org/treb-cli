package usecase

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/trebuchet-org/treb-cli/internal/domain/config"
)

// SetAddressbookParams contains parameters for setting an addressbook entry.
type SetAddressbookParams struct {
	Name    string
	Address string
}

// SetAddressbookResult contains the result of setting an addressbook entry.
type SetAddressbookResult struct {
	Name    string
	Address string
	ChainID uint64
}

// SetAddressbook is the use case for adding/updating an addressbook entry.
type SetAddressbook struct {
	config *config.RuntimeConfig
	repo   AddressbookRepository
}

// NewSetAddressbook creates a new SetAddressbook use case.
func NewSetAddressbook(cfg *config.RuntimeConfig, repo AddressbookRepository) *SetAddressbook {
	return &SetAddressbook{
		config: cfg,
		repo:   repo,
	}
}

var addressRegexp = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

// Run executes the set addressbook use case.
func (uc *SetAddressbook) Run(ctx context.Context, params SetAddressbookParams) (*SetAddressbookResult, error) {
	if uc.config.Network == nil {
		return nil, fmt.Errorf("no network configured; set one with --network or `treb config set network <name>`")
	}

	// Validate address format
	if !addressRegexp.MatchString(params.Address) {
		return nil, fmt.Errorf("invalid address %q: must be a 0x-prefixed 40-character hex string", params.Address)
	}

	chainKey := fmt.Sprintf("%d", uc.config.Network.ChainID)

	ab, err := uc.repo.Load(ctx)
	if err != nil {
		return nil, err
	}

	if ab[chainKey] == nil {
		ab[chainKey] = make(map[string]string)
	}

	// Store with checksummed address (EIP-55 not enforced here, just store as-given)
	ab[chainKey][params.Name] = strings.TrimSpace(params.Address)

	if err := uc.repo.Save(ctx, ab); err != nil {
		return nil, err
	}

	return &SetAddressbookResult{
		Name:    params.Name,
		Address: params.Address,
		ChainID: uc.config.Network.ChainID,
	}, nil
}
