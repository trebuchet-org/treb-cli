package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/domain/config"
	"github.com/trebuchet-org/treb-cli/internal/domain/models"
)

// mockDeploymentRepo implements DeploymentRepository for testing
type mockDeploymentRepo struct {
	DeploymentRepository // embed to satisfy interface
	listFunc             func(ctx context.Context, filter domain.DeploymentFilter) ([]*models.Deployment, error)
	getAllFunc            func(ctx context.Context) ([]*models.Deployment, error)
}

func (m *mockDeploymentRepo) ListDeployments(ctx context.Context, filter domain.DeploymentFilter) ([]*models.Deployment, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, filter)
	}
	return nil, nil
}

func (m *mockDeploymentRepo) GetAllDeployments(ctx context.Context) ([]*models.Deployment, error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(ctx)
	}
	return nil, nil
}

// mockNetworkResolver implements NetworkResolver for testing
type mockNetworkResolver struct {
	networks map[string]*config.Network
}

func (m *mockNetworkResolver) GetNetworks(_ context.Context) []string {
	names := make([]string, 0, len(m.networks))
	for name := range m.networks {
		names = append(names, name)
	}
	return names
}

func (m *mockNetworkResolver) ResolveNetwork(_ context.Context, name string) (*config.Network, error) {
	if net, ok := m.networks[name]; ok {
		return net, nil
	}
	return nil, domain.ErrNotFound
}

// mockForkState implements ForkStateStore for testing (always returns no fork)
type mockForkState struct{}

func (m *mockForkState) Load(_ context.Context) (*domain.ForkState, error) {
	return nil, domain.ErrNotFound
}

func (m *mockForkState) Save(_ context.Context, _ *domain.ForkState) error {
	return nil
}

func (m *mockForkState) Delete(_ context.Context) error {
	return nil
}

func TestListDeployments_OtherNamespaces(t *testing.T) {
	t.Run("empty namespace with others available", func(t *testing.T) {
		repo := &mockDeploymentRepo{
			listFunc: func(_ context.Context, _ domain.DeploymentFilter) ([]*models.Deployment, error) {
				return nil, nil // current namespace is empty
			},
			getAllFunc: func(_ context.Context) ([]*models.Deployment, error) {
				return []*models.Deployment{
					{Namespace: "production", ChainID: 31337, ContractName: "Counter"},
					{Namespace: "production", ChainID: 31337, ContractName: "Token"},
					{Namespace: "staging", ChainID: 31337, ContractName: "Counter"},
				}, nil
			},
		}

		cfg := &config.RuntimeConfig{
			Namespace: "default",
			Network:   &config.Network{Name: "anvil-31337", ChainID: 31337},
		}

		uc := NewListDeployments(cfg, repo, &mockNetworkResolver{}, &mockForkState{})
		result, err := uc.Run(context.Background(), ListDeploymentsParams{})

		require.NoError(t, err)
		assert.Empty(t, result.Deployments)
		assert.Equal(t, map[string]int{
			"production": 2,
			"staging":    1,
		}, result.OtherNamespaces)
		assert.Equal(t, "default", result.CurrentNamespace)
		assert.Equal(t, "anvil-31337", result.CurrentNetwork)
	})

	t.Run("empty namespace with no others", func(t *testing.T) {
		repo := &mockDeploymentRepo{
			listFunc: func(_ context.Context, _ domain.DeploymentFilter) ([]*models.Deployment, error) {
				return nil, nil
			},
			getAllFunc: func(_ context.Context) ([]*models.Deployment, error) {
				return nil, nil // nothing anywhere
			},
		}

		cfg := &config.RuntimeConfig{
			Namespace: "default",
			Network:   &config.Network{Name: "anvil-31337", ChainID: 31337},
		}

		uc := NewListDeployments(cfg, repo, &mockNetworkResolver{}, &mockForkState{})
		result, err := uc.Run(context.Background(), ListDeploymentsParams{})

		require.NoError(t, err)
		assert.Empty(t, result.Deployments)
		assert.Nil(t, result.OtherNamespaces)
	})

	t.Run("non-empty namespace stays nil", func(t *testing.T) {
		repo := &mockDeploymentRepo{
			listFunc: func(_ context.Context, _ domain.DeploymentFilter) ([]*models.Deployment, error) {
				return []*models.Deployment{
					{Namespace: "default", ChainID: 31337, ContractName: "Counter"},
				}, nil
			},
			// getAllFunc should NOT be called when deployments exist
		}

		cfg := &config.RuntimeConfig{
			Namespace: "default",
			Network:   &config.Network{Name: "anvil-31337", ChainID: 31337},
		}

		uc := NewListDeployments(cfg, repo, &mockNetworkResolver{}, &mockForkState{})
		result, err := uc.Run(context.Background(), ListDeploymentsParams{})

		require.NoError(t, err)
		assert.Len(t, result.Deployments, 1)
		assert.Nil(t, result.OtherNamespaces)
	})

	t.Run("chain ID filtering applies to other-namespace counts", func(t *testing.T) {
		repo := &mockDeploymentRepo{
			listFunc: func(_ context.Context, _ domain.DeploymentFilter) ([]*models.Deployment, error) {
				return nil, nil // current namespace is empty
			},
			getAllFunc: func(_ context.Context) ([]*models.Deployment, error) {
				return []*models.Deployment{
					{Namespace: "production", ChainID: 31337, ContractName: "Counter"},
					{Namespace: "production", ChainID: 1, ContractName: "Token"}, // different chain
					{Namespace: "staging", ChainID: 1, ContractName: "Counter"},   // different chain
				}, nil
			},
		}

		cfg := &config.RuntimeConfig{
			Namespace: "default",
			Network:   &config.Network{Name: "anvil-31337", ChainID: 31337},
		}

		uc := NewListDeployments(cfg, repo, &mockNetworkResolver{}, &mockForkState{})
		result, err := uc.Run(context.Background(), ListDeploymentsParams{})

		require.NoError(t, err)
		assert.Empty(t, result.Deployments)
		// Only production has deployments on chain 31337
		assert.Equal(t, map[string]int{
			"production": 1,
		}, result.OtherNamespaces)
	})

	t.Run("no network set counts all chains", func(t *testing.T) {
		repo := &mockDeploymentRepo{
			listFunc: func(_ context.Context, _ domain.DeploymentFilter) ([]*models.Deployment, error) {
				return nil, nil
			},
			getAllFunc: func(_ context.Context) ([]*models.Deployment, error) {
				return []*models.Deployment{
					{Namespace: "production", ChainID: 31337, ContractName: "Counter"},
					{Namespace: "production", ChainID: 1, ContractName: "Token"},
					{Namespace: "staging", ChainID: 1, ContractName: "Counter"},
				}, nil
			},
		}

		cfg := &config.RuntimeConfig{
			Namespace: "default",
			// Network is nil - no chain filter
		}

		uc := NewListDeployments(cfg, repo, &mockNetworkResolver{}, &mockForkState{})
		result, err := uc.Run(context.Background(), ListDeploymentsParams{})

		require.NoError(t, err)
		assert.Empty(t, result.Deployments)
		assert.Equal(t, map[string]int{
			"production": 2,
			"staging":    1,
		}, result.OtherNamespaces)
		assert.Equal(t, "default", result.CurrentNamespace)
		assert.Empty(t, result.CurrentNetwork)
	})

	t.Run("excludes current namespace from other namespaces", func(t *testing.T) {
		repo := &mockDeploymentRepo{
			listFunc: func(_ context.Context, _ domain.DeploymentFilter) ([]*models.Deployment, error) {
				return nil, nil // empty due to other filters (e.g. contract name)
			},
			getAllFunc: func(_ context.Context) ([]*models.Deployment, error) {
				return []*models.Deployment{
					{Namespace: "default", ChainID: 31337, ContractName: "Token"},
					{Namespace: "production", ChainID: 31337, ContractName: "Counter"},
				}, nil
			},
		}

		cfg := &config.RuntimeConfig{
			Namespace: "default",
			Network:   &config.Network{Name: "anvil-31337", ChainID: 31337},
		}

		uc := NewListDeployments(cfg, repo, &mockNetworkResolver{}, &mockForkState{})
		result, err := uc.Run(context.Background(), ListDeploymentsParams{})

		require.NoError(t, err)
		// "default" namespace should be excluded even though it has deployments in GetAllDeployments
		assert.Equal(t, map[string]int{
			"production": 1,
		}, result.OtherNamespaces)
	})
}
