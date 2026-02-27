package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
)

// This test verifies that VerifyDeployment properly integrates with DeploymentResolver
func TestVerifyDeployment_UsesDeploymentResolver(t *testing.T) {
	t.Run("constructor accepts deployment resolver", func(t *testing.T) {
		// Verify that NewVerifyDeployment accepts a deployment resolver as the fourth parameter
		// This ensures the verify command can use interactive selection when multiple deployments match
		var (
			repo               DeploymentRepository
			contractVerifier   ContractVerifier
			networkResolver    NetworkResolver
			deploymentResolver DeploymentResolver
		)

		var progress ProgressSink

		uc := NewVerifyDeployment(repo, contractVerifier, networkResolver, deploymentResolver, progress)
		assert.NotNil(t, uc)
	})

	t.Run("verify specific constructs proper deployment query", func(t *testing.T) {
		// Create a simple mock that captures the query
		var capturedQuery domain.DeploymentQuery
		mockResolver := &mockDeploymentResolver{
			resolveFunc: func(ctx context.Context, query domain.DeploymentQuery) (*models.Deployment, error) {
				capturedQuery = query
				return nil, domain.ErrNotFound
			},
		}

		uc := NewVerifyDeployment(nil, nil, nil, mockResolver, NopProgress{})

		// Test that the filter parameters are properly passed to the query
		filter := domain.DeploymentFilter{
			ChainID:   31337,
			Namespace: "production",
		}

		_, err := uc.VerifySpecific(context.Background(), "Counter:v2", filter, VerifyOptions{})

		// Error is expected since mock returns not found
		assert.Error(t, err)

		// Verify the deployment resolver was called with correct query
		assert.Equal(t, "Counter:v2", capturedQuery.Reference)
		assert.Equal(t, uint64(31337), capturedQuery.ChainID)
		assert.Equal(t, "production", capturedQuery.Namespace)
	})
}

// Simple mock implementation for testing
type mockDeploymentResolver struct {
	resolveFunc func(context.Context, domain.DeploymentQuery) (*models.Deployment, error)
}

func (m *mockDeploymentResolver) ResolveDeployment(ctx context.Context, query domain.DeploymentQuery) (*models.Deployment, error) {
	if m.resolveFunc != nil {
		return m.resolveFunc(ctx, query)
	}
	return nil, domain.ErrNotFound
}
