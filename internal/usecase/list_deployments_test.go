package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/trebuchet-org/treb-cli/internal/domain"
	"github.com/trebuchet-org/treb-cli/internal/usecase"
)

// MockDeploymentStore is a mock implementation of DeploymentStore
type MockDeploymentStore struct {
	mock.Mock
}

func (m *MockDeploymentStore) GetDeployment(ctx context.Context, id string) (*domain.Deployment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Deployment), args.Error(1)
}

func (m *MockDeploymentStore) GetDeploymentByAddress(ctx context.Context, chainID uint64, address string) (*domain.Deployment, error) {
	args := m.Called(ctx, chainID, address)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Deployment), args.Error(1)
}

func (m *MockDeploymentStore) ListDeployments(ctx context.Context, filter usecase.DeploymentFilter) ([]*domain.Deployment, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Deployment), args.Error(1)
}

func (m *MockDeploymentStore) SaveDeployment(ctx context.Context, deployment *domain.Deployment) error {
	args := m.Called(ctx, deployment)
	return args.Error(0)
}

func (m *MockDeploymentStore) DeleteDeployment(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockProgressSink is a mock implementation of ProgressSink
type MockProgressSink struct {
	events []usecase.ProgressEvent
}

func (m *MockProgressSink) OnProgress(ctx context.Context, event usecase.ProgressEvent) {
	m.events = append(m.events, event)
}

func TestListDeployments(t *testing.T) {
	ctx := context.Background()

	t.Run("list all deployments", func(t *testing.T) {
		// Create test deployments
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
			},
			{
				ID:           "staging/1/Token:v2",
				Namespace:    "staging",
				ChainID:      1,
				ContractName: "Token",
				Label:        "v2",
				Address:      "0x2222222222222222222222222222222222222222",
				Type:         domain.SingletonDeployment,
				CreatedAt:    time.Now(),
			},
			{
				ID:           "production/137/Counter:v1",
				Namespace:    "production",
				ChainID:      137,
				ContractName: "Counter",
				Label:        "v1",
				Address:      "0x3333333333333333333333333333333333333333",
				Type:         domain.ProxyDeployment,
				CreatedAt:    time.Now(),
			},
		}

		// Setup mocks
		store := new(MockDeploymentStore)
		store.On("ListDeployments", ctx, usecase.DeploymentFilter{}).Return(deployments, nil)

		progress := &MockProgressSink{}

		// Create and run use case
		uc := usecase.NewListDeployments(store, progress)
		result, err := uc.Run(ctx, usecase.ListDeploymentsParams{})

		// Assertions
		require.NoError(t, err)
		assert.Len(t, result.Deployments, 3)
		assert.Equal(t, 3, result.Summary.Total)
		assert.Equal(t, 2, result.Summary.ByNamespace["production"])
		assert.Equal(t, 1, result.Summary.ByNamespace["staging"])
		assert.Equal(t, 2, result.Summary.ByChain[1])
		assert.Equal(t, 1, result.Summary.ByChain[137])
		assert.Equal(t, 2, result.Summary.ByType[domain.SingletonDeployment])
		assert.Equal(t, 1, result.Summary.ByType[domain.ProxyDeployment])

		// Check progress events
		assert.Len(t, progress.events, 2)
		assert.Equal(t, "loading", progress.events[0].Stage)
		assert.Equal(t, "complete", progress.events[1].Stage)

		// Verify deployments are sorted
		assert.Equal(t, "production", result.Deployments[0].Namespace)
		assert.Equal(t, uint64(1), result.Deployments[0].ChainID)
		assert.Equal(t, "production", result.Deployments[1].Namespace)
		assert.Equal(t, uint64(137), result.Deployments[1].ChainID)
		assert.Equal(t, "staging", result.Deployments[2].Namespace)

		store.AssertExpectations(t)
	})

	t.Run("list with filters", func(t *testing.T) {
		// Create test deployments
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
			},
		}

		// Setup mocks
		store := new(MockDeploymentStore)
		expectedFilter := usecase.DeploymentFilter{
			Namespace:    "production",
			ChainID:      1,
			ContractName: "Counter",
		}
		store.On("ListDeployments", ctx, expectedFilter).Return(deployments, nil)

		progress := &MockProgressSink{}

		// Create and run use case
		uc := usecase.NewListDeployments(store, progress)
		result, err := uc.Run(ctx, usecase.ListDeploymentsParams{
			Namespace:    "production",
			ChainID:      1,
			ContractName: "Counter",
		})

		// Assertions
		require.NoError(t, err)
		assert.Len(t, result.Deployments, 1)
		assert.Equal(t, "production/1/Counter:v1", result.Deployments[0].ID)

		store.AssertExpectations(t)
	})

	t.Run("empty result", func(t *testing.T) {
		// Setup mocks
		store := new(MockDeploymentStore)
		store.On("ListDeployments", ctx, usecase.DeploymentFilter{}).Return([]*domain.Deployment{}, nil)

		progress := &MockProgressSink{}

		// Create and run use case
		uc := usecase.NewListDeployments(store, progress)
		result, err := uc.Run(ctx, usecase.ListDeploymentsParams{})

		// Assertions
		require.NoError(t, err)
		assert.Len(t, result.Deployments, 0)
		assert.Equal(t, 0, result.Summary.Total)
		assert.Empty(t, result.Summary.ByNamespace)
		assert.Empty(t, result.Summary.ByChain)
		assert.Empty(t, result.Summary.ByType)

		store.AssertExpectations(t)
	})

	t.Run("store error", func(t *testing.T) {
		// Setup mocks
		store := new(MockDeploymentStore)
		expectedErr := errors.New("store error")
		store.On("ListDeployments", ctx, usecase.DeploymentFilter{}).Return(nil, expectedErr)

		progress := &MockProgressSink{}

		// Create and run use case
		uc := usecase.NewListDeployments(store, progress)
		result, err := uc.Run(ctx, usecase.ListDeploymentsParams{})

		// Assertions
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)

		// Should have loading event but not complete event
		assert.Len(t, progress.events, 1)
		assert.Equal(t, "loading", progress.events[0].Stage)

		store.AssertExpectations(t)
	})

	t.Run("sorting behavior", func(t *testing.T) {
		// Create deployments in unsorted order
		deployments := []*domain.Deployment{
			{
				ID:           "staging/137/Token:b",
				Namespace:    "staging",
				ChainID:      137,
				ContractName: "Token",
				Label:        "b",
			},
			{
				ID:           "production/1/Counter:a",
				Namespace:    "production",
				ChainID:      1,
				ContractName: "Counter",
				Label:        "a",
			},
			{
				ID:           "staging/137/Token:a",
				Namespace:    "staging",
				ChainID:      137,
				ContractName: "Token",
				Label:        "a",
			},
			{
				ID:           "production/137/Counter:b",
				Namespace:    "production",
				ChainID:      137,
				ContractName: "Counter",
				Label:        "b",
			},
			{
				ID:           "production/1/Token:a",
				Namespace:    "production",
				ChainID:      1,
				ContractName: "Token",
				Label:        "a",
			},
		}

		// Setup mocks
		store := new(MockDeploymentStore)
		store.On("ListDeployments", ctx, usecase.DeploymentFilter{}).Return(deployments, nil)

		progress := &MockProgressSink{}

		// Create and run use case
		uc := usecase.NewListDeployments(store, progress)
		result, err := uc.Run(ctx, usecase.ListDeploymentsParams{})

		// Assertions
		require.NoError(t, err)
		assert.Len(t, result.Deployments, 5)

		// Check sorting order
		expected := []string{
			"production/1/Counter:a",
			"production/1/Token:a",
			"production/137/Counter:b",
			"staging/137/Token:a",
			"staging/137/Token:b",
		}

		for i, dep := range result.Deployments {
			assert.Equal(t, expected[i], dep.ID, "Deployment at index %d", i)
		}

		store.AssertExpectations(t)
	})
}