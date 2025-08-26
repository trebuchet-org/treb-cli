package usecase

import (
	"context"
	"fmt"

	"github.com/trebuchet-org/treb-cli/internal/domain"
)

// ManageAnvil handles anvil node management operations
type ManageAnvil struct {
	anvilManager AnvilManager
	progress     ProgressSink
}

// NewManageAnvil creates a new anvil management use case
func NewManageAnvil(anvilManager AnvilManager, progress ProgressSink) *ManageAnvil {
	return &ManageAnvil{
		anvilManager: anvilManager,
		progress:     progress,
	}
}

// ManageAnvilParams contains parameters for anvil operations
type ManageAnvilParams struct {
	Operation string // start, stop, restart, status, logs
	Name      string
	Port      string
	ChainID   string
}

// ManageAnvilResult contains the result of anvil operations
type ManageAnvilResult struct {
	Operation string
	Instance  *domain.AnvilInstance
	Status    *domain.AnvilStatus
	Success   bool
	Message   string
}

// Execute performs the anvil management operation
func (m *ManageAnvil) Execute(ctx context.Context, params ManageAnvilParams) (*ManageAnvilResult, error) {
	instance := &domain.AnvilInstance{
		Name:    params.Name,
		Port:    params.Port,
		ChainID: params.ChainID,
	}

	switch params.Operation {
	case "start":
		return m.start(ctx, instance)
	case "stop":
		return m.stop(ctx, instance)
	case "restart":
		return m.restart(ctx, instance)
	case "status":
		return m.status(ctx, instance)
	case "logs":
		return m.logs(ctx, instance)
	default:
		return nil, fmt.Errorf("unknown operation: %s", params.Operation)
	}
}

func (m *ManageAnvil) start(ctx context.Context, instance *domain.AnvilInstance) (*ManageAnvilResult, error) {
	m.progress.Info(fmt.Sprintf("🔨 Starting local anvil node '%s' on port %s...", instance.Name, instance.Port))

	// Check if already running
	status, err := m.anvilManager.GetStatus(ctx, instance)
	if err == nil && status.Running {
		return nil, fmt.Errorf("anvil '%s' is already running (PID %d)", instance.Name, status.PID)
	}

	// Start the instance
	if err := m.anvilManager.Start(ctx, instance); err != nil {
		return nil, fmt.Errorf("failed to start anvil: %w", err)
	}

	// Get updated status
	status, err = m.anvilManager.GetStatus(ctx, instance)
	if err != nil {
		return nil, fmt.Errorf("failed to get status after start: %w", err)
	}

	result := &ManageAnvilResult{
		Operation: "start",
		Instance:  instance,
		Status:    status,
		Success:   true,
		Message:   fmt.Sprintf("Anvil '%s' started with PID %d", instance.Name, status.PID),
	}

	return result, nil
}

func (m *ManageAnvil) stop(ctx context.Context, instance *domain.AnvilInstance) (*ManageAnvilResult, error) {
	m.progress.Info(fmt.Sprintf("🛑 Stopping anvil '%s'...", instance.Name))

	// Check if running
	status, err := m.anvilManager.GetStatus(ctx, instance)
	if err != nil || !status.Running {
		return &ManageAnvilResult{
			Operation: "stop",
			Instance:  instance,
			Success:   true,
			Message:   fmt.Sprintf("Anvil '%s' is not running", instance.Name),
		}, nil
	}

	// Stop the instance
	if err := m.anvilManager.Stop(ctx, instance); err != nil {
		return nil, fmt.Errorf("failed to stop anvil: %w", err)
	}

	result := &ManageAnvilResult{
		Operation: "stop",
		Instance:  instance,
		Success:   true,
		Message:   "Anvil stopped",
	}

	return result, nil
}

func (m *ManageAnvil) restart(ctx context.Context, instance *domain.AnvilInstance) (*ManageAnvilResult, error) {
	m.progress.Info(fmt.Sprintf("🔄 Restarting anvil '%s'...", instance.Name))

	// Stop if running
	status, err := m.anvilManager.GetStatus(ctx, instance)
	if err == nil && status.Running {
		if err := m.anvilManager.Stop(ctx, instance); err != nil {
			return nil, fmt.Errorf("failed to stop anvil: %w", err)
		}
	}

	// Start
	if err := m.anvilManager.Start(ctx, instance); err != nil {
		return nil, fmt.Errorf("failed to start anvil: %w", err)
	}

	// Get updated status
	status, err = m.anvilManager.GetStatus(ctx, instance)
	if err != nil {
		return nil, fmt.Errorf("failed to get status after restart: %w", err)
	}

	result := &ManageAnvilResult{
		Operation: "restart",
		Instance:  instance,
		Status:    status,
		Success:   true,
		Message:   fmt.Sprintf("Anvil '%s' restarted with PID %d", instance.Name, status.PID),
	}

	return result, nil
}

func (m *ManageAnvil) status(ctx context.Context, instance *domain.AnvilInstance) (*ManageAnvilResult, error) {
	status, err := m.anvilManager.GetStatus(ctx, instance)
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	result := &ManageAnvilResult{
		Operation: "status",
		Instance:  instance,
		Status:    status,
		Success:   true,
	}

	return result, nil
}

func (m *ManageAnvil) logs(ctx context.Context, instance *domain.AnvilInstance) (*ManageAnvilResult, error) {
	// For logs, we just need to check if the instance has a log file
	status, err := m.anvilManager.GetStatus(ctx, instance)
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	// The actual log streaming will be handled by the renderer
	result := &ManageAnvilResult{
		Operation: "logs",
		Instance:  instance,
		Status:    status,
		Success:   true,
	}

	return result, nil
}
