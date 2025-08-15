package registry

import (
	"context"

	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// UpdaterAdapter adapts the registry updater for the usecase layer
type UpdaterAdapter struct {
	updater *InternalUpdater
}

// NewUpdaterAdapter creates a new registry updater adapter
func NewUpdaterAdapter(
	deploymentStore usecase.DeploymentStore,
	transactionStore usecase.TransactionStore,
) *UpdaterAdapter {
	updater := NewInternalUpdater(deploymentStore, transactionStore)
	return &UpdaterAdapter{
		updater: updater,
	}
}

// PrepareUpdates analyzes the execution and prepares registry updates
func (a *UpdaterAdapter) PrepareUpdates(
	ctx context.Context,
	execution *domain.ScriptExecution,
	namespace string,
	network string,
) (*usecase.RegistryChanges, error) {
	return a.updater.PrepareUpdates(ctx, execution, namespace, network)
}

// ApplyUpdates applies the prepared changes to the registry
func (a *UpdaterAdapter) ApplyUpdates(ctx context.Context, changes *usecase.RegistryChanges) error {
	return a.updater.ApplyUpdates(ctx, changes)
}

// HasChanges returns true if there are any changes to apply
func (a *UpdaterAdapter) HasChanges(changes *usecase.RegistryChanges) bool {
	return a.updater.HasChanges(changes)
}