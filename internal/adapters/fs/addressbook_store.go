package fs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// AddressbookStoreAdapter implements AddressbookRepository using the file system.
type AddressbookStoreAdapter struct {
	path string
}

// NewAddressbookStoreAdapter creates a new AddressbookStoreAdapter.
func NewAddressbookStoreAdapter(cfg *config.RuntimeConfig) *AddressbookStoreAdapter {
	return &AddressbookStoreAdapter{
		path: filepath.Join(cfg.DataDir, "addressbook.json"),
	}
}

// Load reads the addressbook from disk. Returns an empty map when the file doesn't exist.
func (s *AddressbookStoreAdapter) Load(_ context.Context) (domain.Addressbook, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(domain.Addressbook), nil
		}
		return nil, fmt.Errorf("failed to read addressbook: %w", err)
	}

	var ab domain.Addressbook
	if err := json.Unmarshal(data, &ab); err != nil {
		return nil, fmt.Errorf("failed to parse addressbook: %w", err)
	}

	return ab, nil
}

// Save writes the addressbook to disk.
func (s *AddressbookStoreAdapter) Save(_ context.Context, ab domain.Addressbook) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create addressbook directory: %w", err)
	}

	data, err := json.MarshalIndent(ab, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal addressbook: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write addressbook: %w", err)
	}

	return nil
}

// GetPath returns the path to the addressbook file.
func (s *AddressbookStoreAdapter) GetPath() string {
	return s.path
}

var _ usecase.AddressbookRepository = (*AddressbookStoreAdapter)(nil)
