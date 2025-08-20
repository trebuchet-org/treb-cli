package deployments

import (
	"context"
	"time"

	"github.com/trebuchet-org/treb-cli/internal/domain/forge"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
)

// PrepareUpdates analyzes the execution and prepares registry updates
func (f *FileRepository) BuildChangesetFromRunResult(ctx context.Context, execution *forge.HydratedRunResult) (*models.Changeset, error) {
	changeset := &models.Changeset{
		Create: &models.ChangesetModels{
			Deployments:      []*models.Deployment{},
			Transactions:     []*models.Transaction{},
			SafeTransactions: []*models.SafeTransaction{},
		},
	}

	// TODO: Implement the actual parsing logic from HydratedRunResult
	// This would involve:
	// 1. Parsing deployment events from execution
	// 2. Creating Deployment models
	// 3. Creating Transaction models
	// 4. Linking deployments to transactions

	return changeset, nil
}

// ApplyBatchUpdate applies all updates in a single transaction with one lock
func (m *FileRepository) ApplyChangeset(ctx context.Context, changeset *models.Changeset) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// Apply all deployment updates
	for _, deployment := range changeset.Create.Deployments {
		// Set timestamps
		if deployment.CreatedAt.IsZero() {
			deployment.CreatedAt = now
		}
		deployment.UpdatedAt = now

		// Save deployment
		m.deployments[deployment.ID] = deployment
	}

	// Apply all transaction updates
	for _, tx := range changeset.Create.Transactions {
		// Set timestamp
		if tx.CreatedAt.IsZero() {
			tx.CreatedAt = now
		}

		// Save transaction
		m.transactions[tx.ID] = tx
	}

	// Apply all safe transaction updates
	for _, tx := range changeset.Create.SafeTransactions {
		// Set timestamp
		if tx.CreatedAt.IsZero() {
			tx.CreatedAt = now
		}

		// Save transaction
		m.safeTransactions[tx.ID] = tx
	}

	// Rebuild lookups once for all changes
	m.rebuildLookups()

	// Save all files once
	return m.save()
}
