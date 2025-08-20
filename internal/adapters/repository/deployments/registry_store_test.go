package deployments_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trebuchet-org/treb-cli/internal/adapters/fs"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

func TestRegistryStoreAdapter(t *testing.T) {
	ctx := context.Background()

	t.Run("create and retrieve deployment", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, err := fs.NewRegistryStoreAdapter(tmpDir)
		require.NoError(t, err)

		// Create a test deployment
		deployment := &domain.Deployment{
			ID:           "test/31337/Counter:v1",
			Namespace:    "test",
			ChainID:      31337,
			ContractName: "Counter",
			Label:        "v1",
			Address:      "0x1234567890123456789012345678901234567890",
			Type:         domain.SingletonDeployment,
			DeploymentStrategy: domain.DeploymentStrategy{
				Method: domain.DeploymentMethodCreate3,
				Salt:   "0xabcd",
			},
			Artifact: domain.ArtifactInfo{
				Path:            "src/Counter.sol:Counter",
				CompilerVersion: "0.8.19",
			},
			Verification: domain.VerificationInfo{
				Status: domain.VerificationStatusUnverified,
			},
			Tags:      []string{"production"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Save deployment
		err = store.SaveDeployment(ctx, deployment)
		require.NoError(t, err)

		// Retrieve by ID
		retrieved, err := store.GetDeployment(ctx, deployment.ID)
		require.NoError(t, err)
		assert.Equal(t, deployment.ID, retrieved.ID)
		assert.Equal(t, deployment.Address, retrieved.Address)
		assert.Equal(t, deployment.ContractName, retrieved.ContractName)
	})

	t.Run("get deployment by address", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, err := fs.NewRegistryStoreAdapter(tmpDir)
		require.NoError(t, err)

		deployment := &domain.Deployment{
			ID:           "test/31337/Token:main",
			Namespace:    "test",
			ChainID:      31337,
			ContractName: "Token",
			Label:        "main",
			Address:      "0xabcdef0123456789012345678901234567890123",
			Type:         domain.SingletonDeployment,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		err = store.SaveDeployment(ctx, deployment)
		require.NoError(t, err)

		// Retrieve by address
		retrieved, err := store.GetDeploymentByAddress(ctx, 31337, deployment.Address)
		require.NoError(t, err)
		assert.Equal(t, deployment.ID, retrieved.ID)
		assert.Equal(t, deployment.Address, retrieved.Address)
	})

	t.Run("list deployments with filters", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, err := fs.NewRegistryStoreAdapter(tmpDir)
		require.NoError(t, err)

		// Create multiple deployments
		deployments := []*domain.Deployment{
			{
				ID:           "production/1/Counter:v1",
				Namespace:    "production",
				ChainID:      1,
				ContractName: "Counter",
				Label:        "v1",
				Address:      "0x1111111111111111111111111111111111111111",
				Type:         domain.SingletonDeployment,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			{
				ID:           "production/1/Token:v1",
				Namespace:    "production",
				ChainID:      1,
				ContractName: "Token",
				Label:        "v1",
				Address:      "0x2222222222222222222222222222222222222222",
				Type:         domain.SingletonDeployment,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			{
				ID:           "staging/1/Counter:v2",
				Namespace:    "staging",
				ChainID:      1,
				ContractName: "Counter",
				Label:        "v2",
				Address:      "0x3333333333333333333333333333333333333333",
				Type:         domain.ProxyDeployment,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			{
				ID:           "production/137/Counter:v1",
				Namespace:    "production",
				ChainID:      137,
				ContractName: "Counter",
				Label:        "v1",
				Address:      "0x4444444444444444444444444444444444444444",
				Type:         domain.SingletonDeployment,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
		}

		for _, dep := range deployments {
			err := store.SaveDeployment(ctx, dep)
			require.NoError(t, err)
		}

		// Test various filters
		tests := []struct {
			name     string
			filter   usecase.DeploymentFilter
			expected int
		}{
			{
				name:     "all deployments",
				filter:   usecase.DeploymentFilter{},
				expected: 4,
			},
			{
				name: "filter by namespace",
				filter: usecase.DeploymentFilter{
					Namespace: "production",
				},
				expected: 3,
			},
			{
				name: "filter by chain",
				filter: usecase.DeploymentFilter{
					ChainID: 1,
				},
				expected: 3,
			},
			{
				name: "filter by contract name",
				filter: usecase.DeploymentFilter{
					ContractName: "Counter",
				},
				expected: 3,
			},
			{
				name: "filter by type",
				filter: usecase.DeploymentFilter{
					Type: domain.ProxyDeployment,
				},
				expected: 1,
			},
			{
				name: "filter by namespace and chain",
				filter: usecase.DeploymentFilter{
					Namespace: "production",
					ChainID:   1,
				},
				expected: 2,
			},
			{
				name: "filter by all criteria",
				filter: usecase.DeploymentFilter{
					Namespace:    "production",
					ChainID:      1,
					ContractName: "Counter",
					Label:        "v1",
					Type:         domain.SingletonDeployment,
				},
				expected: 1,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := store.ListDeployments(ctx, tt.filter)
				require.NoError(t, err)
				assert.Len(t, result, tt.expected)
			})
		}
	})

	t.Run("not found errors", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, err := fs.NewRegistryStoreAdapter(tmpDir)
		require.NoError(t, err)

		// Test GetDeployment not found
		_, err = store.GetDeployment(ctx, "nonexistent")
		assert.ErrorIs(t, err, domain.ErrNotFound)

		// Test GetDeploymentByAddress not found
		_, err = store.GetDeploymentByAddress(ctx, 1, "0x0000000000000000000000000000000000000000")
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("proxy deployment conversion", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, err := fs.NewRegistryStoreAdapter(tmpDir)
		require.NoError(t, err)

		deployment := &domain.Deployment{
			ID:           "test/31337/Proxy:v1",
			Namespace:    "test",
			ChainID:      31337,
			ContractName: "Proxy",
			Label:        "v1",
			Address:      "0x5555555555555555555555555555555555555555",
			Type:         domain.ProxyDeployment,
			ProxyInfo: &domain.ProxyInfo{
				Type:           "ERC1967",
				Implementation: "0x6666666666666666666666666666666666666666",
				Admin:          "0x7777777777777777777777777777777777777777",
				History: []domain.ProxyUpgrade{
					{
						ImplementationID: "test/31337/Implementation:v1",
						UpgradedAt:       time.Now(),
						UpgradeTxID:      "tx-123",
					},
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = store.SaveDeployment(ctx, deployment)
		require.NoError(t, err)

		retrieved, err := store.GetDeployment(ctx, deployment.ID)
		require.NoError(t, err)

		assert.NotNil(t, retrieved.ProxyInfo)
		assert.Equal(t, deployment.ProxyInfo.Type, retrieved.ProxyInfo.Type)
		assert.Equal(t, deployment.ProxyInfo.Implementation, retrieved.ProxyInfo.Implementation)
		assert.Len(t, retrieved.ProxyInfo.History, 1)
	})
}

func TestRegistryStoreAdapter_PersistenceAcrossInstances(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Create first instance and save deployment
	store1, err := fs.NewRegistryStoreAdapter(tmpDir)
	require.NoError(t, err)

	deployment := &domain.Deployment{
		ID:           "test/31337/Persistent:v1",
		Namespace:    "test",
		ChainID:      31337,
		ContractName: "Persistent",
		Label:        "v1",
		Address:      "0x8888888888888888888888888888888888888888",
		Type:         domain.SingletonDeployment,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = store1.SaveDeployment(ctx, deployment)
	require.NoError(t, err)

	// Create second instance and verify persistence
	store2, err := fs.NewRegistryStoreAdapter(tmpDir)
	require.NoError(t, err)

	retrieved, err := store2.GetDeployment(ctx, deployment.ID)
	require.NoError(t, err)
	assert.Equal(t, deployment.ID, retrieved.ID)
	assert.Equal(t, deployment.Address, retrieved.Address)
}

func TestRegistryStoreAdapter_RegistryFileStructure(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	store, err := fs.NewRegistryStoreAdapter(tmpDir)
	require.NoError(t, err)

	deployment := &domain.Deployment{
		ID:           "test/31337/FileTest:v1",
		Namespace:    "test",
		ChainID:      31337,
		ContractName: "FileTest",
		Label:        "v1",
		Address:      "0x9999999999999999999999999999999999999999",
		Type:         domain.SingletonDeployment,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = store.SaveDeployment(ctx, deployment)
	require.NoError(t, err)

	// Verify registry files were created
	trebDir := filepath.Join(tmpDir, ".treb")
	assert.DirExists(t, trebDir)
	assert.FileExists(t, filepath.Join(trebDir, "deployments.json"))
	assert.FileExists(t, filepath.Join(trebDir, "registry.json"))
}

