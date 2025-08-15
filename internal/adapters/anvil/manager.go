package anvil

import (
	"context"
	"io"

	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// Manager is an adapter that wraps the internal anvil manager
type Manager struct {
	internal *InternalManager
}

// NewManager creates a new anvil manager adapter
func NewManager() *Manager {
	return &Manager{
		internal: NewInternalManager(),
	}
}

// Start starts an anvil instance
func (m *Manager) Start(ctx context.Context, instance *domain.AnvilInstance) error {
	return m.internal.Start(ctx, instance)
}

// Stop stops an anvil instance
func (m *Manager) Stop(ctx context.Context, instance *domain.AnvilInstance) error {
	return m.internal.Stop(ctx, instance)
}

// GetStatus gets the status of an anvil instance
func (m *Manager) GetStatus(ctx context.Context, instance *domain.AnvilInstance) (*domain.AnvilStatus, error) {
	return m.internal.GetStatus(ctx, instance)
}

// StreamLogs streams logs from an anvil instance
func (m *Manager) StreamLogs(ctx context.Context, instance *domain.AnvilInstance, writer io.Writer) error {
	return m.internal.StreamLogs(ctx, instance, writer)
}